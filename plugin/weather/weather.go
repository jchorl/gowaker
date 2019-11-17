package weather

import (
	"fmt"
	"math"

	owm "github.com/briandowns/openweathermap"
)

type TempUnit string

func (t TempUnit) String() string {
	return string(t)
}

const (
	Celsius    TempUnit = "C"
	Fahrenheit TempUnit = "F"
)

type Weather struct {
	APIKey   string
	TempUnit TempUnit
	OWMID    int
}

func (w Weather) Text() (string, error) {
	o, err := owm.NewForecast("5", w.TempUnit.String(), "EN", w.APIKey)
	if err != nil {
		return "", fmt.Errorf("creating owm client: %w", err)
	}

	err = o.DailyByID(w.OWMID, 1)
	if err != nil {
		return "", fmt.Errorf("w.DailyByID: %w", err)
	}

	forecast := o.ForecastWeatherJson.(*owm.Forecast5WeatherData).List[0]

	return fmt.Sprintf(
		"Today's forecast is %s with a high of %.0f degrees and a low of %.0f degrees.",
		forecast.Weather[0].Description,
		math.Round(forecast.Main.TempMax),
		math.Round(forecast.Main.TempMin),
	), nil
}
