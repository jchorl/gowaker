package alarmrun

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	log "github.com/golang/glog"
	upstreamspotify "github.com/jchorl/spotify"
	"github.com/valyala/fasthttp"

	"github.com/jchorl/gowaker/config"
	"github.com/jchorl/gowaker/requestcontext"
	"github.com/jchorl/gowaker/speech"
	"github.com/jchorl/gowaker/spotify"
)

func AlarmRun(ctx *fasthttp.RequestCtx) error {
	log.Infof("running job at %s", time.Now())
	err := setVolume()
	if err != nil {
		log.Error("setting volume: %s", err)
		return err
	}

	speechChan := make(chan []byte)
	speechErrChan := make(chan error)

	go func() {
		speechStr, err := generateSpeechStr(ctx)
		if err != nil {
			speechErrChan <- fmt.Errorf("generating speech: %s", err)
			return
		}

		contents, err := speech.GetAudioContent(ctx, speechStr)
		if err != nil {
			speechErrChan <- fmt.Errorf("getting audio content: %s", err)
			return
		}

		speechChan <- contents
	}()

	err = playSong(ctx)
	if err != nil {
		log.Error(err)
		return err
	}

	var contents []byte
	select {
	case err = <-speechErrChan:
		log.Error(err)
		return err
	case contents = <-speechChan:
	}

	streamer, format, err := wav.Decode(bytes.NewReader(contents))
	if err != nil {
		log.Error("wav decoding: %s", err)
		return err
	}
	defer streamer.Close()

	done := make(chan bool)
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))
	<-done

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

func generateSpeechStr(ctx *fasthttp.RequestCtx) (string, error) {
	plugins := requestcontext.Plugins(ctx)

	var fullStr string
	for _, p := range plugins {
		pText, err := p.Text()
		if err != nil {
			return "", fmt.Errorf("p.Text(): %w", err)
		}
		fullStr += pText
	}

	fullStr += "Have a great day!"

	return fullStr, nil
}
