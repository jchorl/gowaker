package alarmrun

import (
	log "github.com/golang/glog"
)

func AlarmRun() error {
	log.Info("running job")
	return nil
}

func AlarmRunOnce() error {
	AlarmRun()

	// TODO deschedule the job

	return nil
}
