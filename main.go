package main

import (
	"database/sql"
	"flag"
	"time"

	"github.com/fasthttp/router"
	log "github.com/golang/glog"
	"github.com/jasonlvhit/gocron"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"

	"github.com/jchorl/gowaker/alarms"
	"github.com/jchorl/gowaker/requestcontext"
)

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./waker.db")
	if err != nil {
		return nil, errors.Wrap(err, "error opening db file")
	}

	sqlStmt := `
	create table if not exists spotify_config (key text not null primary key, value text not null);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrapf(err, "error executing sql statement: %s", sqlStmt)
		return nil, err
	}

	sqlStmt = `
	create table if not exists alarms (
		id text not null primary key,
		hour int not null,
		minute int not null,
		repeat bool not null,
		days string -- csv of days to repeat
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrapf(err, "error executing sql statement: %s", sqlStmt)
		return nil, err
	}

	return db, nil

	// insert example
	// stmt, err := db.Prepare("insert into foo(id, name) values(?, ?)")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer stmt.Close()
	// _, err = stmt.Exec(i, "hello", i))
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	// query example
	// rows, err := db.Query("select id, name from foo")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer rows.Close()
	// for rows.Next() {
	// 	var id int
	// 	var name string
	// 	err = rows.Scan(&id, &name)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Println(id, name)
	// }
	// err = rows.Err()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// get by ID example
	// stmt, err = db.Prepare("select name from foo where id = ?")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer stmt.Close()
	// var name string
	// err = stmt.QueryRow("3").Scan(&name)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(name)
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

func middlewareApplier(db *sql.DB, scheduler *gocron.Scheduler) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return dbMiddleware(
			schedulerMiddleware(
				handler,
				scheduler,
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

	// TODO patch gocron to allow a per-scheduler timezone
	gocron.ChangeLoc(time.UTC) // all timestamps are in UTC
	scheduler := gocron.NewScheduler()

	mockCtx := fasthttp.RequestCtx{}
	requestcontext.SetDB(&mockCtx, db)
	requestcontext.SetScheduler(&mockCtx, scheduler)
	err = alarms.RestoreAlarmsFromDB(&mockCtx)
	if err != nil {
		log.Fatalf("error restoring db: %s", err)
	}

	middlewares := middlewareApplier(db, scheduler)

	r := router.New()
	r.GET("/alarms", middlewares(alarms.HandlerGet))
	r.DELETE("/alarms", middlewares(alarms.HandlerDelete))
	r.POST("/alarms", middlewares(alarms.HandlerPost))

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
