package alarms

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"

	"github.com/jasonlvhit/gocron"
	"github.com/jchorl/gowaker/alarmrun"
	"github.com/jchorl/gowaker/requestcontext"
)

type Alarm struct {
	ID      string    `json:"id"`
	Time    Time      `json:"time"`
	Repeat  bool      `json:"repeat"`
	Days    []string  `json:"days"`
	NextRun time.Time `json:"next_run"`
}

type Time struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

const alarmCronType = "alarm"

func HandlerPost(ctx *fasthttp.RequestCtx) {
	alarm := Alarm{}
	err := json.Unmarshal(ctx.Request.Body(), &alarm)
	if err != nil {
		err = errors.Wrap(err, "error decoding body")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	alarm, err = newAlarm(ctx, alarm)
	if err != nil {
		err = errors.Wrap(err, "error creating new alarm")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(&alarm)
}

func newAlarm(ctx *fasthttp.RequestCtx, alarm Alarm) (Alarm, error) {
	alarm.ID = uuid.New().String()

	alarm = newAlarmCron(ctx, alarm)

	err := newAlarmDB(ctx, alarm)
	if err != nil {
		return Alarm{}, err
	}

	return alarm, nil
}

func newAlarmCron(ctx *fasthttp.RequestCtx, alarm Alarm) Alarm {
	scheduler := requestcontext.Scheduler(ctx)
	if !alarm.Repeat {
		job := scheduler.
			Every(1).
			Day().
			At(
				fmt.Sprintf("%d:%d", alarm.Time.Hour, alarm.Time.Minute),
			).
			Tag(jobTag("id", alarm.ID), jobTag("type", alarmCronType))

		job.Do(alarmrun.AlarmRunOnce)
		alarm.NextRun = job.NextScheduledTime()
	} else {
		// create an alarm for each day it should run
		for _, day := range alarm.Days {
			job := scheduler.
				Every(1).
				Weekday(dayStrToTimeDay[day]).
				At(
					fmt.Sprintf("%d:%d", alarm.Time.Hour, alarm.Time.Minute),
				).
				Tag(jobTag("id", alarm.ID), jobTag("type", alarmCronType))

			job.Do(alarmrun.AlarmRun)

			thisNextTime := job.NextScheduledTime()
			if alarm.NextRun.Equal(time.Time{}) || thisNextTime.Before(alarm.NextRun) {
				alarm.NextRun = thisNextTime
			}
		}
	}

	return alarm
}

func newAlarmDB(ctx *fasthttp.RequestCtx, alarm Alarm) error {
	db := requestcontext.DB(ctx)

	daysCSV := strings.Join(alarm.Days, ",")

	stmt, err := db.Prepare(`
		insert into alarms(id, hour, minute, repeat, days)
		values(?, ?, ?, ?, ?)
	`,
	)
	if err != nil {
		return errors.Wrap(err, "error preparing alarm insert stmt")
	}
	defer stmt.Close()
	_, err = stmt.Exec(alarm.ID, alarm.Time.Hour, alarm.Time.Minute, alarm.Repeat, daysCSV)
	if err != nil {
		return errors.Wrap(err, "error executing alarm insert stmt")
	}

	return nil
}

func HandlerGet(ctx *fasthttp.RequestCtx) {
	scheduler := requestcontext.Scheduler(ctx)
	allJobs := scheduler.Jobs()

	groupedByID := map[string][]*gocron.Job{}
	for _, job := range allJobs {
		jobType := getJobTagValue(job, "type")
		if jobType != alarmCronType {
			continue
		}

		jobID := getJobTagValue(job, "id")
		groupedByID[jobID] = append(groupedByID[jobID], job)
	}

	alarms := []Alarm{}
	for jobID, jobs := range groupedByID {
		// for the time, just take the first job, they should all be the same
		sample := jobs[0]
		timeStr := sample.GetAt()
		timeStrSplit := strings.Split(timeStr, ":")
		hour, _ := strconv.Atoi(timeStrSplit[0])
		minute, _ := strconv.Atoi(timeStrSplit[1])

		alarm := Alarm{
			ID: jobID,
			Time: Time{
				Hour:   hour,
				Minute: minute,
			},
		}

		if len(jobs) == 1 {
			alarm.NextRun = jobs[0].NextScheduledTime()
		} else {
			alarm.Repeat = true

			for _, job := range jobs {
				alarm.Days = append(
					alarm.Days,
					strings.ToLower(job.GetWeekday().String()),
				)

				thisNextTime := job.NextScheduledTime()
				if alarm.NextRun.Equal(time.Time{}) || thisNextTime.Before(alarm.NextRun) {
					alarm.NextRun = thisNextTime
				}
			}
		}

		alarms = append(alarms, alarm)
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(alarms)
}

func HandlerDelete(ctx *fasthttp.RequestCtx) {
	// TODO modify client to send body
	alarm := Alarm{}
	err := json.Unmarshal(ctx.Request.Body(), &alarm)
	if err != nil {
		err = errors.Wrap(err, "error decoding body")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	err = deleteAlarm(ctx, alarm.ID)
	if err != nil {
		err = errors.Wrap(err, "error deleting alarm")
		log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
}

func deleteAlarm(ctx *fasthttp.RequestCtx, id string) error {
	// TODO figure out how to delete an alarm
	// need to delete from db and scheduler
	log.Errorf("should have deleted alarm %d but don't know how", id)
	return errors.New("unimplemented")
}

// RestoreAlarmsFromDB restores alarms into the scheduler
func RestoreAlarmsFromDB(ctx *fasthttp.RequestCtx) error {
	db := requestcontext.DB(ctx)

	// query example
	rows, err := db.Query("select id, hour, minute, repeat, days from alarms")
	if err != nil {
		return errors.Wrap(err, "unable to query existing alarms")
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var hour int
		var minute int
		var repeat bool
		var days string

		err = rows.Scan(&id, &hour, &minute, &repeat, &days)
		if err != nil {
			return errors.Wrap(err, "unable to scan row")
		}

		daysSplit := strings.Split(days, ",")
		alarm := Alarm{
			ID: id,
			Time: Time{
				Hour:   hour,
				Minute: minute,
			},
			Repeat: repeat,
			Days:   daysSplit,
		}
		log.Infof("restoring alarm: %+v", alarm)
		newAlarmCron(ctx, alarm)
	}
	err = rows.Err()
	if err != nil {
		return errors.Wrap(err, "error iterating over alarm query results")
	}

	return nil
}

var dayStrToTimeDay = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

func jobTag(key, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}

func getJobTagValue(job *gocron.Job, key string) string {
	for _, tag := range job.GetAllTags() {
		sp := strings.Split(tag, ":")
		if sp[0] == key && len(sp) == 2 {
			return sp[1]
		}
	}

	return ""
}
