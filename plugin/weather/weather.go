package weather

import (
	"fmt"
	"math"
	"os"

	owm "github.com/briandowns/openweathermap"

	"github.com/jchorl/gowaker/config"
)

func Text() (string, error) {
	w, err := owm.NewForecast("5", "C", "EN", os.Getenv("OPENWEATHERMAP_API_KEY"))
	if err != nil {
		return "", fmt.Errorf("creating owm client: %w", err)
	}

	err = w.DailyByID(config.OWMID, 1)
	if err != nil {
		return "", fmt.Errorf("w.DailyByID: %w", err)
	}

	forecast := w.ForecastWeatherJson.(*owm.Forecast5WeatherData).List[0]

	return fmt.Sprintf(
		"Today's forecast is %s with a high of %.0f degrees and a low of %.0f degrees.",
		forecast.Weather[0].Description,
		math.Round(forecast.Main.TempMax),
		math.Round(forecast.Main.TempMin),
	), nil
}
