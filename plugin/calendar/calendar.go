package calendar

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"

	"github.com/jchorl/gowaker/util"
)

// Calendar is a plugin that interacts with google calendar
type Calendar struct {
	calendars map[string]bool
	client    *calendar.Service
}

// New creates a new calendar client.
func New(calendars []string, oauthConfigFile, oauthCredFile string) (Calendar, error) {
	b, err := ioutil.ReadFile(oauthConfigFile)
	if err != nil {
		return Calendar{}, fmt.Errorf("reading %s: %w", oauthConfigFile, err)
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		return Calendar{}, fmt.Errorf("google.ConfigFromJSON: %w", err)
	}

	httpClient, err := util.GetOauthClient(context.TODO(), config, oauthCredFile)
	if err != nil {
		return Calendar{}, fmt.Errorf("GetOauthClient(): %w", err)
	}

	srv, err := calendar.New(httpClient)
	if err != nil {
		return Calendar{}, fmt.Errorf("calendar.New(): %w", err)
	}

	cals := map[string]bool{}
	for _, cal := range calendars {
		cals[cal] = true
	}

	return Calendar{client: srv, calendars: cals}, nil
}

// Text returns a string of upcoming calendar events
func (c Calendar) Text() (string, error) {
	cals, err := c.client.CalendarList.List().ShowHidden(true).Do()
	if err != nil {
		return "", fmt.Errorf("fetching cals: %w", err)
	}

	start := time.Now()

	// go to midnight Local time
	year, month, day := start.Date()
	end := time.Date(year, month, day, 23, 59, 59, 0, time.Local)

	var events []*calendar.Event
	for _, cal := range cals.Items {
		if _, ok := c.calendars[cal.Summary]; !ok {
			continue
		}

		calEvents, err := c.client.Events.List(cal.Id).ShowDeleted(false).SingleEvents(true).
			TimeMin(start.Format(time.RFC3339)).TimeMax(end.Format(time.RFC3339)).Do()
		if err != nil {
			return "", fmt.Errorf("listing events: %w", err)
		}

		for _, e := range calEvents.Items {
			events = append(events, e)
		}
	}

	if len(events) == 0 {
		return "There are no calendar events today.", nil
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].Start.DateTime != "" && events[j].Start.DateTime != "" {
			evI, _ := time.Parse(time.RFC3339, events[i].Start.DateTime)
			evJ, _ := time.Parse(time.RFC3339, events[j].Start.DateTime)
			return evI.Before(evJ)
		}

		// if i is an all-day event
		if events[i].Start.Date != "" {
			return true
		}

		return false
	})

	str := "Here are the upcoming calendar events for today. "
	for _, item := range events {
		if item.Start.DateTime == "" {
			str += item.Summary + ". "
			continue
		}

		t, _ := time.Parse(time.RFC3339, item.Start.DateTime)
		str += item.Summary + " at " + t.In(time.Local).Format("15:04") + ". "
	}

	return str, nil
}
