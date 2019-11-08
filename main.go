package main

import (
	"database/sql"
	"flag"
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

type middleware func(fasthttp.RequestHandler) fasthttp.RequestHandler

func dbMiddleware(db *sql.DB) middleware {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			requestcontext.SetDB(ctx, db)
			handler(ctx)
		}
	}
}

func schedulerMiddleware(scheduler *gocron.Scheduler) middleware {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			requestcontext.SetScheduler(ctx, scheduler)
			handler(ctx)
		}
	}
}

func spotifyMiddleware(client *upstreamspotify.Client) middleware {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			requestcontext.SetSpotify(ctx, client)
			handler(ctx)
		}
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
		log.Fatalf("restoring db: %s", err)
	}

	spotifyClient, err := spotify.New()
	if err != nil {
		log.Fatalf("creating spotify client: %s", err)
	}

	middlewares := []middleware{
		dbMiddleware(db),
		schedulerMiddleware(scheduler),
		spotifyMiddleware(spotifyClient),
	}

	middlewareApplier := func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		wrapped := handler
		for _, m := range middlewares {
			wrapped = m(wrapped)
		}
		return wrapped
	}

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
