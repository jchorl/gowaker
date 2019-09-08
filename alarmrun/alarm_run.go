package alarmrun

import (
	"os/exec"
	"time"

	log "github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"

	"github.com/jchorl/gowaker/config"
	"github.com/jchorl/gowaker/spotify"
)

func AlarmRun(ctx *fasthttp.RequestCtx) error {
	log.Infof("running job at %s", time.Now())
	err := setVolume()
	if err != nil {
		err = errors.Wrap(err, "error setting volume")
		log.Error(err)
		return err
	}

	err = playSong(ctx)
	if err != nil {
		err = errors.Wrap(err, "error playing song")
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
		return err
	}

	cmd := exec.Command("amixer", "sset", "Line Out", "100%")
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func playSong(ctx *fasthttp.RequestCtx) error {
	devices, err := spotify.GetDevices(ctx)
	if err != nil {
		return err
	}

	for _, d := range devices {
		if d.Name == config.SpotifyDeviceName {
			// TODO this
		}
	}
}
