package main

import (
	"database/sql"
	"math/rand"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	log "github.com/golang/glog"
	"github.com/jasonlvhit/gocron"
	upstreamspotify "github.com/jchorl/spotify"
	"github.com/valyala/fasthttp"

	"github.com/jchorl/gowaker/plugin"
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

func spotifyMiddleware(client upstreamspotify.Client) middleware {
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

func speechMiddleware(client *texttospeech.Client) middleware {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			requestcontext.SetSpeech(ctx, client)
			handler(ctx)
		}
	}
}

func pluginsMiddleware(plugins ...plugin.Plugin) middleware {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			requestcontext.SetPlugins(ctx, plugins)
			handler(ctx)
		}
	}
}

func logMiddleware() middleware {
	return func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			handler(ctx)
			log.Infof("CANONICAL-REQUEST-LINE method=%s path=%s status_code=%d", ctx.Method(), ctx.Path(), ctx.Response.StatusCode())
		}
	}
}
