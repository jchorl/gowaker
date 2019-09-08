package spotify

import (
	"encoding/json"

	log "github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"github.com/zmb3/spotify"

	"github.com/jchorl/gowaker/requestcontext"
)

const defaultPlaylistKey = "default_playlist"
const nextWakeupSongKey = "next_wakeup_song"

// TODO figure out how to use a service account to get a spotify client with
// ScopePlaylistReadCollaborative
// spotify.ScopePlaylistReadPrivate
// spotify.ScopeUserReadPlaybackState
// spotify.ScopeUserModifyPlaybackState

func HandlerGetPlaylists(ctx *fasthttp.RequestCtx) {
	spotifyClient := requestcontext.Spotify(ctx)
	playlistPage, err := spotifyClient.CurrentUsersPlaylists()
	if err != nil {
		err = errors.Wrap(err, "error retrieving spotify playlists")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(playlistPage.Playlists)
}

func HandlerGetDefaultPlaylist(ctx *fasthttp.RequestCtx) {
	spotifyClient := requestcontext.Spotify(ctx)
	db := requestcontext.DB(ctx)

	stmt, err := db.Prepare("select value from spotify_config where key = ?")
	if err != nil {
		err = errors.Wrap(err, "error retrieving default spotify playlist")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	var playlistID string
	err = stmt.QueryRow(defaultPlaylistKey).Scan(&playlistID)
	if err != nil {
		err = errors.Wrap(err, "error querying/scanning default playlist")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	playlist, err := spotifyClient.GetPlaylist(spotify.ID(playlistID))
	if err != nil {
		err = errors.Wrap(err, "error fetching playlist details from spotify")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(playlist)
}

func HandlerSetDefaultPlaylist(ctx *fasthttp.RequestCtx) {
	playlist := spotify.SimplePlaylist{}
	err := json.Unmarshal(ctx.Request.Body(), &playlist)
	if err != nil {
		err = errors.Wrap(err, "error decoding body")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	// TODO figure out upsert in sqlite
	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(playlist)
}

func HandlerGetNextWakeupSong(ctx *fasthttp.RequestCtx) {
	spotifyClient := requestcontext.Spotify(ctx)
	db := requestcontext.DB(ctx)

	stmt, err := db.Prepare("select value from spotify_config where key = ?")
	if err != nil {
		err = errors.Wrap(err, "error retrieving next wakeup song")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	var songID string
	err = stmt.QueryRow(nextWakeupSongKey).Scan(&songID)
	if err != nil {
		err = errors.Wrap(err, "error querying next wakeup song")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	song, err := spotifyClient.GetTrack(spotify.ID(songID))
	if err != nil {
		err = errors.Wrap(err, "error fetching track details from spotify")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(song)
}

func HandlerSetNextWakeupSong(ctx *fasthttp.RequestCtx) {
	song := spotify.FullTrack{}
	err := json.Unmarshal(ctx.Request.Body(), &song)
	if err != nil {
		err = errors.Wrap(err, "error decoding body")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	// TODO figure out upsert in sqlite
	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(song)
}

func ClearNextWakeupSong(ctx *fasthttp.RequestCtx) error {
	db := requestcontext.DB(ctx)
	stmt, err := db.Prepare(`
		delete from spotify_config where key = ?
	`,
	)
	if err != nil {
		return errors.Wrap(err, "error preparing next wakeup song clear stmt")
	}
	defer stmt.Close()
	_, err = stmt.Exec(nextWakeupSongKey)
	if err != nil {
		return errors.Wrap(err, "error executing next wakeup song clear stmt")
	}

	return nil
}

func HandlerSearch(ctx *fasthttp.RequestCtx) {
	spotifyClient := requestcontext.Spotify(ctx)
	query := ctx.FormValue("q")

	results, err := spotifyClient.Search(string(query), spotify.SearchTypeTrack)
	if err != nil {
		err = errors.Wrap(err, "error searching spotify")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(results)
}

func HandlerDevices(ctx *fasthttp.RequestCtx) {
	devices, err := GetDevices(ctx)
	if err != nil {
		err = errors.Wrap(err, "error fetching devices from spotify")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(devices)
}

func GetDevices(ctx *fasthttp.RequestCtx) ([]spotify.PlayerDevice, error) {
	spotifyClient := requestcontext.Spotify(ctx)
	devices, err := spotifyClient.PlayerDevices()
	if err != nil {
		return nil, errors.Wrap("error fetching devices from spotify")
	}

	return devices, nil
}
