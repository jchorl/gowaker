package requestcontext

import (
	"database/sql"
	"math/rand"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/jasonlvhit/gocron"
	"github.com/jchorl/spotify"
	"github.com/valyala/fasthttp"

	"github.com/jchorl/gowaker/plugin"
)

// we hold internal state that no other package can touch.
// fasthttp closes io.Closers when reqs finish, but our clients
// tend to be longlived. This pattern avoids them being closed.
type intCtxKey string
type intCtx map[intCtxKey]interface{}

const dbKey intCtxKey = "db"

func SetDB(ctx *fasthttp.RequestCtx, db *sql.DB) {
	set(ctx, dbKey, db)
}

func DB(ctx *fasthttp.RequestCtx) *sql.DB {
	return get(ctx, dbKey).(*sql.DB)
}

const schedulerKey intCtxKey = "scheduler"

func SetScheduler(ctx *fasthttp.RequestCtx, scheduler *gocron.Scheduler) {
	set(ctx, schedulerKey, scheduler)
}

func Scheduler(ctx *fasthttp.RequestCtx) *gocron.Scheduler {
	return get(ctx, schedulerKey).(*gocron.Scheduler)
}

const spotifyKey intCtxKey = "spotify"

func SetSpotify(ctx *fasthttp.RequestCtx, client spotify.Client) {
	set(ctx, spotifyKey, client)
}

func Spotify(ctx *fasthttp.RequestCtx) spotify.Client {
	return get(ctx, spotifyKey).(spotify.Client)
}

const randKey = "rand"

func SetRand(ctx *fasthttp.RequestCtx, r *rand.Rand) {
	set(ctx, randKey, r)
}

func Rand(ctx *fasthttp.RequestCtx) *rand.Rand {
	return get(ctx, randKey).(*rand.Rand)
}

const speechKey = "speech"

func SetSpeech(ctx *fasthttp.RequestCtx, c *texttospeech.Client) {
	set(ctx, speechKey, c)
}

func Speech(ctx *fasthttp.RequestCtx) *texttospeech.Client {
	return get(ctx, speechKey).(*texttospeech.Client)
}

const pluginsKey = "plugins"

func SetPlugins(ctx *fasthttp.RequestCtx, plugins []plugin.Plugin) {
	set(ctx, pluginsKey, plugins)
}

func Plugins(ctx *fasthttp.RequestCtx) []plugin.Plugin {
	return get(ctx, pluginsKey).([]plugin.Plugin)
}

const internalCtxKey = "__gowaker_internal"

func Clone(ctx *fasthttp.RequestCtx) *fasthttp.RequestCtx {
	cl := &fasthttp.RequestCtx{}
	cl.SetUserValue(internalCtxKey, ctx.UserValue(internalCtxKey))
	return cl
}

func set(ctx *fasthttp.RequestCtx, key intCtxKey, value interface{}) {
	// make sure internal ctx is already on the request
	intCtxUntyped := ctx.UserValue(internalCtxKey)
	if intCtxUntyped == nil {
		intCtxUntyped = intCtx(map[intCtxKey]interface{}{})
		ctx.SetUserValue(internalCtxKey, intCtxUntyped)
	}

	c := intCtxUntyped.(intCtx)
	c[key] = value
}

func get(ctx *fasthttp.RequestCtx, key intCtxKey) interface{} {
	return ctx.UserValue(internalCtxKey).(intCtx)[key]
}
