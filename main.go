package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
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
	upstreamspotify "github.com/zmb3/spotify"

	"github.com/jchorl/gowaker/alarms"
	"github.com/jchorl/gowaker/requestcontext"
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

func dbMiddleware(handler fasthttp.RequestHandler, db *sql.DB) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		requestcontext.SetDB(ctx, db)
		handler(ctx)
	}
}

func schedulerMiddleware(handler fasthttp.RequestHandler, scheduler *gocron.Scheduler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		requestcontext.SetScheduler(ctx, scheduler)
		handler(ctx)
	}
}

func spotifyMiddleware(handler fasthttp.RequestHandler, client *upstreamspotify.Client) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		requestcontext.SetSpotify(ctx, client)
		handler(ctx)
	}
}

func middlewareApplier(db *sql.DB, spotifyClient *upstreamspotify.Client, scheduler *gocron.Scheduler) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return dbMiddleware(
			spotifyMiddleware(
				schedulerMiddleware(
					handler,
					scheduler,
				),
				spotifyClient,
			),
			db,
		)
	}
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

	mockCtx := fasthttp.RequestCtx{}
	requestcontext.SetDB(&mockCtx, db)
	requestcontext.SetScheduler(&mockCtx, scheduler)
	err = alarms.RestoreAlarmsFromDB(&mockCtx)
	if err != nil {
		log.Fatalf("error restoring db: %s", err)
	}

	auth := upstreamspotify.NewAuthenticator("http://localhost:5000/spotify/auth", spotify.RequiredScopes...)
	url := auth.AuthURL("")
	fmt.Printf("Visit %s and OAuth\n", url)
	fmt.Print("Enter code: ")
	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("getting code: %s", err)
	}
	token, err := auth.Exchange(strings.TrimSpace(code))
	if err != nil {
		log.Fatalf("getting spotify token: %s", err)
	}
	spotifyClient := auth.NewClient(token)

	middlewares := middlewareApplier(db, &spotifyClient, scheduler)

	r := router.New()
	r.GET("/alarms", middlewares(alarms.HandlerGet))
	r.DELETE("/alarms", middlewares(alarms.HandlerDelete))
	r.POST("/alarms", middlewares(alarms.HandlerPost))

	r.GET("/spotify/playlists", middlewares(spotify.HandlerGetPlaylists))
	r.GET("/spotify/default_playlist", middlewares(spotify.HandlerGetDefaultPlaylist))
	r.PUT("/spotify/default_playlist", middlewares(spotify.HandlerSetDefaultPlaylist))
	r.GET("/spotify/next_wakeup_song", middlewares(spotify.HandlerGetNextWakeupSong))
	r.PUT("/spotify/next_wakeup_song", middlewares(spotify.HandlerSetNextWakeupSong))
	r.GET("/spotify/search", middlewares(spotify.HandlerSearch))
	r.GET("/spotify/devices", middlewares(spotify.HandlerDevices))

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
