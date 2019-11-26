package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/fasthttp/router"
	log "github.com/golang/glog"
	"github.com/jasonlvhit/gocron"
	"github.com/jchorl/watchdog"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"

	"github.com/jchorl/gowaker/alarms"
	"github.com/jchorl/gowaker/config"
	"github.com/jchorl/gowaker/plugin/calendar"
	"github.com/jchorl/gowaker/plugin/weather"
	"github.com/jchorl/gowaker/spotify"
)

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./waker.db")
	if err != nil {
		return nil, errors.Wrap(err, "error opening db file")
	}

	sqlStmt := `
	create table if not exists alarms (
		id text not null primary key,
		hour int not null,
		minute int not null,
		repeat bool not null,
		days string -- csv of days to repeat
	);
	create table if not exists spotify_config (
		key text not null primary key,
		value text not null
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrapf(err, "error executing sql statement: %s", sqlStmt)
		return nil, err
	}

	return db, nil
}

func main() {
	flag.Parse()

	db, err := initDB()
	if err != nil {
		log.Fatalf("error initing db: %s", err)
	}
	defer db.Close()

	scheduler := gocron.NewScheduler()
	scheduler.ChangeLoc(time.UTC) // all timestamps are in UTC
	job := scheduler.Every(1).Hour()
	job.Tag("watchdog")
	job.Do(func() {
		wdClient := watchdog.Client{Domain: "https://watchdog.joshchorlton.com"}
		wdClient.Ping("waker", watchdog.Watch_DAILY)
	})

	spotifyClient, err := spotify.New()
	if err != nil {
		log.Fatalf("creating spotify client: %s", err)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	weatherPlugin := weather.Weather{
		APIKey:   os.Getenv("OPENWEATHERMAP_API_KEY"),
		TempUnit: weather.Celsius,
		OWMID:    config.OWMID,
	}

	calendars, err := csv.NewReader(strings.NewReader(config.GoogleCalendars)).Read()
	if err != nil {
		log.Fatalf("parsing calendars: %s", err)
	}
	calendarPlugin, err := calendar.New(calendars)
	if err != nil {
		log.Fatalf("creating calendar plugin: %s", err)
	}

	middlewares := []middleware{
		dbMiddleware(db),
		schedulerMiddleware(scheduler),
		spotifyMiddleware(spotifyClient),
		randMiddleware(rng),
		pluginsMiddleware(weatherPlugin, calendarPlugin),
	}

	middlewareApplier := func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		wrapped := handler
		for _, m := range middlewares {
			wrapped = m(wrapped)
		}
		return wrapped
	}

	// this is a fun hack that uses the middlewares to create a ctx that we use to restore crons
	fakeHandler := middlewareApplier(func(ctx *fasthttp.RequestCtx) {
		err = alarms.RestoreAlarmsFromDB(ctx)
		if err != nil {
			log.Fatalf("restoring db: %s", err)
		}
	})
	fakeHandler(&fasthttp.RequestCtx{})

	r := router.New()
	r.GET("/alarms", middlewareApplier(alarms.HandlerGet))
	r.DELETE("/alarms", middlewareApplier(alarms.HandlerDelete))
	r.POST("/alarms", middlewareApplier(alarms.HandlerPost))

	r.GET("/spotify/playlists", middlewareApplier(spotify.HandlerGetPlaylists))
	r.GET("/spotify/default_playlist", middlewareApplier(spotify.HandlerGetDefaultPlaylist))
	r.PUT("/spotify/default_playlist", middlewareApplier(spotify.HandlerSetDefaultPlaylist))
	r.GET("/spotify/next_wakeup_song", middlewareApplier(spotify.HandlerGetNextWakeupSong))
	r.PUT("/spotify/next_wakeup_song", middlewareApplier(spotify.HandlerSetNextWakeupSong))
	r.GET("/spotify/search", middlewareApplier(spotify.HandlerSearch))
	r.GET("/spotify/devices", middlewareApplier(spotify.HandlerDevices))

	serverDone := make(chan struct{})
	go func() {
		port := ":8080"
		log.Infof("listening on %s", port)
		log.Error(fasthttp.ListenAndServe(port, r.Handler))
		serverDone <- struct{}{}
	}()

	schedulerDone := make(chan struct{})
	go func() {
		log.Info("starting the job processor")
		ticker := time.NewTicker(30 * time.Second)
		for {
			<-ticker.C
			scheduler.RunPending()
		}
		schedulerDone <- struct{}{}
	}()

	select {
	case <-schedulerDone:
		log.Fatalf("scheduler crashed")
	case <-serverDone:
		log.Fatalf("server crashed")
	}
}
