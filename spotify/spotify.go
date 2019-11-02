package spotify

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
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
		err = fmt.Errorf("retrieving spotify playlists: %w", err)
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
		err = fmt.Errorf("retrieving default spotify playlist: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	var playlistID string
	err = stmt.QueryRow(defaultPlaylistKey).Scan(&playlistID)
	if err != nil {
		err = fmt.Errorf("querying/scanning default playlist: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	playlist, err := spotifyClient.GetPlaylist(spotify.ID(playlistID))
	if err != nil {
		err = fmt.Errorf("fetching playlist details from spotify: %w", err)
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
		err = fmt.Errorf("decoding body: %w", err)
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
		err = fmt.Errorf("retrieving next wakeup song: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	var songID string
	err = stmt.QueryRow(nextWakeupSongKey).Scan(&songID)
	if err != nil {
		err = fmt.Errorf("querying next wakeup song: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	song, err := spotifyClient.GetTrack(spotify.ID(songID))
	if err != nil {
		err = fmt.Errorf("fetching track details from spotify: %w", err)
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
		err = fmt.Errorf("decoding body: %w", err)
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
		return fmt.Errorf("preparing next wakeup song clear stmt: %w", err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(nextWakeupSongKey)
	if err != nil {
		return fmt.Errorf("executing next wakeup song clear stmt: %w", err)
	}

	return nil
}

func HandlerSearch(ctx *fasthttp.RequestCtx) {
	spotifyClient := requestcontext.Spotify(ctx)
	query := ctx.FormValue("q")

	results, err := spotifyClient.Search(string(query), spotify.SearchTypeTrack)
	if err != nil {
		err = fmt.Errorf("searching spotify: %w", err)
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
		return nil, fmt.Errorf("fetching devices from spotify: %w", err)
	}

	return devices, nil
}
