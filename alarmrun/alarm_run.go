package alarmrun

import (
	"fmt"
	"os/exec"
	"time"

	log "github.com/golang/glog"
	"github.com/valyala/fasthttp"
	upstreamspotify "github.com/zmb3/spotify"

	"github.com/jchorl/gowaker/config"
	"github.com/jchorl/gowaker/spotify"
)

func AlarmRun(ctx *fasthttp.RequestCtx) error {
	log.Infof("running job at %s", time.Now())
	err := setVolume()
	// if err != nil {
	// 	log.Error(err)
	// 	return err
	// }

	err = playSong(ctx)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Infof("finished job at %s", time.Now())
	return nil
}

func setVolume() error {
	cmd := exec.Command("amixer", "sset", "DAC", "100%")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("setting volume on DAC: %w", err)
	}

	cmd = exec.Command("amixer", "sset", "Line Out", "100%")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("setting volume on Line Out: %w", err)
	}
	return nil
}

func playSong(ctx *fasthttp.RequestCtx) error {
	var device *upstreamspotify.PlayerDevice

	devices, err := spotify.GetDevices(ctx)
	if err != nil {
		return err
	}

	for _, d := range devices {
		if d.Name == config.SpotifyDeviceName {
			device = &d
			break
		}
	}

	if device == nil {
		return fmt.Errorf("finding device %s: %w", config.SpotifyDeviceName, err)
	}

	wakeupSong, err := spotify.GetNextWakeupSong(ctx)
	if err != nil {
		return fmt.Errorf("getting next wakeup song: %w", err)
	}

	err = spotify.PlaySong(ctx, wakeupSong, device)
	if err != nil {
		return fmt.Errorf("playing wakeup song: %w", err)
	}

	err = <-spotify.WaitForSong(ctx)
	if err != nil {
		return fmt.Errorf("waiting for spotify to finish playing: %w", err)
	}

	err = spotify.PauseSong(ctx)
	if err != nil {
		return fmt.Errorf("pausing song: %w", err)
	}

	err = spotify.ClearNextWakeupSong(ctx)
	if err != nil {
		return fmt.Errorf("clearing wakeup song: %w", err)
	}

	return nil
}
