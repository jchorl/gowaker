package main

import (
	"database/sql"
	"math/rand"

	"github.com/jasonlvhit/gocron"
	"github.com/valyala/fasthttp"
	upstreamspotify "github.com/zmb3/spotify"

	"github.com/jchorl/gowaker/requestcontext"
)

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

func randMiddleware(r *rand.Rand) middleware {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			requestcontext.SetRand(ctx, r)
			handler(ctx)
		}
	}
}
