package weather

import (
	"fmt"
	"os"

	owm "github.com/briandowns/openweathermap"
	log "github.com/golang/glog"
)

func GetForecastStr() (string, error) {
	w, err := owm.NewForecast("5", "C", "EN", os.Getenv("OPENWEATHERMAP_API_KEY"))
	if err != nil {
		return "", fmt.Errorf("creating owm client: %w", err)
	}

	// w.CurrentByZip(config.OWMZip, config.OWMCountry)
	err = w.DailyByID(5391959, 3)
	if err != nil {
		return "", fmt.Errorf("w.DailyByID: %w", err)
	}
	log.Infof("%+v", w.ForecastWeatherJson)
	return "", nil
}
