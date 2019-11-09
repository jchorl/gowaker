package spotify

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	log "github.com/golang/glog"
	"github.com/valyala/fasthttp"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"

	"github.com/jchorl/gowaker/requestcontext"
)

const defaultPlaylistKey = "default_playlist"
const nextWakeupSongKey = "next_wakeup_song"

var requiredScopes = []string{
	spotify.ScopePlaylistReadCollaborative,
	spotify.ScopePlaylistReadPrivate,
	spotify.ScopeUserReadPlaybackState,
	spotify.ScopeUserModifyPlaybackState,
}

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
	playlist, err := getDefaultPlaylist(ctx)
	if err != nil {
		err = fmt.Errorf("getting default playlist: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(playlist)
}

func getDefaultPlaylist(ctx *fasthttp.RequestCtx) (*spotify.FullPlaylist, error) {
	spotifyClient := requestcontext.Spotify(ctx)
	db := requestcontext.DB(ctx)

	stmt, err := db.Prepare("select value from spotify_config where key = ?")
	if err != nil {
		return nil, fmt.Errorf("retrieving default spotify playlist: %w", err)
	}
	defer stmt.Close()

	var playlistID string
	err = stmt.QueryRow(defaultPlaylistKey).Scan(&playlistID)
	if err != nil {
		return nil, fmt.Errorf("querying/scanning default playlist: %w", err)
	}

	playlist, err := spotifyClient.GetPlaylist(spotify.ID(playlistID))
	if err != nil {
		return nil, fmt.Errorf("fetching playlist details from spotify: %w", err)
	}

	return playlist, nil
}

func HandlerSetDefaultPlaylist(ctx *fasthttp.RequestCtx) {
	db := requestcontext.DB(ctx)

	playlist := spotify.SimplePlaylist{}
	err := json.Unmarshal(ctx.Request.Body(), &playlist)
	if err != nil {
		err = fmt.Errorf("decoding body: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare(`
		insert into spotify_config(key, value) values(?, ?) on conflict(key) do update set value = ?
	`,
	)
	if err != nil {
		err = fmt.Errorf("preparing default playlist upsert stmt: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(defaultPlaylistKey, playlist.ID, playlist.ID)
	if err != nil {
		err = fmt.Errorf("executing default playlist upsert stmt: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(playlist)
}

func HandlerGetNextWakeupSong(ctx *fasthttp.RequestCtx) {
	song, err := GetNextWakeupSong(ctx)
	if err != nil {
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(song)
}

func GetNextWakeupSong(ctx *fasthttp.RequestCtx) (*spotify.FullTrack, error) {
	spotifyClient := requestcontext.Spotify(ctx)
	db := requestcontext.DB(ctx)

	stmt, err := db.Prepare("select value from spotify_config where key = ?")
	if err != nil {
		return nil, fmt.Errorf("retrieving next wakeup song: %w", err)
	}
	defer stmt.Close()

	var songID string
	err = stmt.QueryRow(nextWakeupSongKey).Scan(&songID)
	if err == sql.ErrNoRows {
		return getRandomSongFromWakeupPlaylist(ctx)
	} else if err != nil {
		return nil, fmt.Errorf("querying next wakeup song: %w", err)
	}

	song, err := spotifyClient.GetTrack(spotify.ID(songID))
	if err != nil {
		return nil, fmt.Errorf("fetching track details from spotify: %w", err)
	}

	return song, err
}

func getRandomSongFromWakeupPlaylist(ctx *fasthttp.RequestCtx) (*spotify.FullTrack, error) {
	playlist, err := getDefaultPlaylist(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting default playlist: %w", err)
	}

	tracks := playlist.Tracks.Tracks
	return &tracks[rand.Intn(len(tracks))].Track, nil
}

func HandlerSetNextWakeupSong(ctx *fasthttp.RequestCtx) {
	db := requestcontext.DB(ctx)

	song := spotify.FullTrack{}
	err := json.Unmarshal(ctx.Request.Body(), &song)
	if err != nil {
		err = fmt.Errorf("decoding body: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare(`
		insert into spotify_config(key, value) values(?, ?) on conflict(key) do update set value = ?
	`,
	)
	if err != nil {
		err = fmt.Errorf("preparing next wakeup song upsert stmt: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(nextWakeupSongKey, song.ID, song.ID)
	if err != nil {
		err = fmt.Errorf("executing next wakeup song upsert stmt: %w", err)
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

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

func PlaySong(ctx *fasthttp.RequestCtx, song *spotify.FullTrack, device *spotify.PlayerDevice) error {
	spotifyClient := requestcontext.Spotify(ctx)
	err := spotifyClient.PlayOpt(&spotify.PlayOptions{
		DeviceID: &device.ID,
		URIs:     []spotify.URI{song.URI},
	})
	if err != nil {
		return fmt.Errorf("spotify playopt: %w", err)
	}

	return nil
}

func PauseSong(ctx *fasthttp.RequestCtx) error {
	spotifyClient := requestcontext.Spotify(ctx)
	err := spotifyClient.Pause()
	if err != nil {
		return fmt.Errorf("spotify pause: %w", err)
	}

	return nil
}

func WaitForSong(ctx *fasthttp.RequestCtx) <-chan error {
	spotifyClient := requestcontext.Spotify(ctx)

	errChan := make(chan error)

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-time.After(10 * time.Minute):
				errChan <- errors.New("timed out polling for spotify song")
				return
			case <-ticker.C:
				currentlyPlaying, err := spotifyClient.PlayerCurrentlyPlaying()
				if err != nil {
					errChan <- fmt.Errorf("checking currently playing: %w", err)
					return
				}

				progressMs := currentlyPlaying.Progress
				durationMs := currentlyPlaying.Item.Duration
				if durationMs-progressMs <= 6000 {
					errChan <- nil
					return
				}
			}
		}
	}()

	return errChan
}

func New() (*spotify.Client, error) {
	auth := spotify.NewAuthenticator("http://localhost:5000/spotify/auth", requiredScopes...)

	bts, err := ioutil.ReadFile("./spotifycreds.json")
	// the no error cases is the exception, it means creds are cached
	if err == nil {
		var token oauth2.Token
		err = json.Unmarshal(bts, &token)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling cached creds, delete ./spotifycreds.json and try again: %w", err)
		}

		client := auth.NewClient(&token)
		return &client, nil
	}

	log.Infof("reading spotifycreds.json: %s", err)

	url := auth.AuthURL("")
	fmt.Printf("Visit %s and OAuth\n", url)
	fmt.Print("Enter code: ")
	reader := bufio.NewReader(os.Stdin)
	code, _ := reader.ReadString('\n')
	token, err := auth.Exchange(strings.TrimSpace(code))
	if err != nil {
		return nil, fmt.Errorf("getting spotify token: %w", err)
	}

	encodedToken, err := json.Marshal(&token)
	if err != nil {
		return nil, fmt.Errorf("marshaling spotify token: %w", err)
	}

	err = ioutil.WriteFile("./spotifycreds.json", encodedToken, 0600)
	if err != nil {
		return nil, fmt.Errorf("saving spotify token: %w", err)
	}

	client := auth.NewClient(token)
	return &client, nil
}
