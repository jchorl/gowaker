package requestcontext

import (
	"database/sql"

	"github.com/jasonlvhit/gocron"
	"github.com/valyala/fasthttp"
)

const (
	dbKey = "db"
)

// fasthttp closes all context objs when the request completes
// so we wrap it in a struct that does not implement io.Closer :(
type dbWrapper struct {
	db *sql.DB
}

func SetDB(ctx *fasthttp.RequestCtx, db *sql.DB) {
	wrapped := dbWrapper{db}
	ctx.SetUserValue(dbKey, wrapped)
}

func DB(ctx *fasthttp.RequestCtx) *sql.DB {
	wrapped := ctx.UserValue(dbKey).(dbWrapper)
	return wrapped.db
}

const schedulerKey = "scheduler"

func SetScheduler(ctx *fasthttp.RequestCtx, scheduler *gocron.Scheduler) {
	ctx.SetUserValue(schedulerKey, scheduler)
}

func Scheduler(ctx *fasthttp.RequestCtx) *gocron.Scheduler {
	return ctx.UserValue(schedulerKey).(*gocron.Scheduler)
}
