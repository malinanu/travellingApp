// Package weather fetches real conditions from Tomorrow.io for the app's
// Weather screen and Discover-tab bar. See internal/http/weather.go for the
// caching HTTP handler that wraps Fetch — this package is just the API call
// and response shaping, no caching of its own.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// HourlyPoint is one hour of forecast, already unit-converted and with a
// weatherCode reduced to the app's existing three-icon vocabulary (see
// conditionFor) — the Flutter client needs no Tomorrow.io-specific mapping.
type HourlyPoint struct {
	Time              time.Time `json:"time"`
	Temp              float64   `json:"temp"`
	Condition         string    `json:"condition"`      // sunny | cloudy | rainy
	ConditionLabel    string    `json:"conditionLabel"` // e.g. "Partly Cloudy"
	Icon              string    `json:"icon"`           // Sun | CloudSun | CloudRain
	PrecipProbability int       `json:"precipProbability"`
	PrecipMm          float64   `json:"precipMm"`
	WindSpeedKmh      float64   `json:"windSpeed"`
	UVIndex           float64   `json:"uvIndex"`
	CloudCoverPct     int       `json:"cloudCover"`
}

// Now is the first hourly entry, plus a couple of fields only meaningful as
// a live snapshot (feels-like, pressure, visibility) rather than per-hour.
type Now struct {
	Temp              float64 `json:"temp"`
	FeelsLike         float64 `json:"feelsLike"`
	Humidity          int     `json:"humidity"`
	WindSpeedKmh      float64 `json:"windSpeed"`
	WindDirection     int     `json:"windDirection"`
	UVIndex           float64 `json:"uvIndex"`
	VisibilityKm      float64 `json:"visibility"`
	PressureHpa       float64 `json:"pressure"`
	PrecipProbability int     `json:"precipProbability"`
	CloudCoverPct     int     `json:"cloudCover"`
	Condition         string  `json:"condition"`
	ConditionLabel    string  `json:"conditionLabel"`
	Icon              string  `json:"icon"`
}

// Snapshot is the full GET /weather response.
type Snapshot struct {
	Now       Now           `json:"now"`
	Hourly    []HourlyPoint `json:"hourly"`
	High      float64       `json:"high"`
	Low       float64       `json:"low"`
	Sunrise   time.Time     `json:"sunrise"`
	Sunset    time.Time     `json:"sunset"`
	FetchedAt time.Time     `json:"fetchedAt"`
}

// tomorrowResponse mirrors only the fields of Tomorrow.io's v4 forecast
// response (https://api.tomorrow.io/v4/weather/forecast) that Snapshot
// needs — see https://docs.tomorrow.io/reference/weather-forecast.
type tomorrowResponse struct {
	Timelines struct {
		Hourly []struct {
			Time   string `json:"time"`
			Values struct {
				Temperature              float64 `json:"temperature"`
				TemperatureApparent      float64 `json:"temperatureApparent"`
				Humidity                 float64 `json:"humidity"`
				WindSpeed                float64 `json:"windSpeed"` // m/s
				WindDirection            float64 `json:"windDirection"`
				UVIndex                  float64 `json:"uvIndex"`
				Visibility               float64 `json:"visibility"`
				PressureSurfaceLevel     float64 `json:"pressureSurfaceLevel"`
				PrecipitationProbability float64 `json:"precipitationProbability"`
				PrecipitationIntensity   float64 `json:"precipitationIntensity"` // mm/hr
				CloudCover               float64 `json:"cloudCover"`
				WeatherCode              int     `json:"weatherCode"`
			} `json:"values"`
		} `json:"hourly"`
		Daily []struct {
			Time   string `json:"time"`
			Values struct {
				TemperatureMax float64 `json:"temperatureMax"`
				TemperatureMin float64 `json:"temperatureMin"`
				SunriseTime    string  `json:"sunriseTime"`
				SunsetTime     string  `json:"sunsetTime"`
			} `json:"values"`
		} `json:"daily"`
	} `json:"timelines"`
}

// Fetch calls Tomorrow.io's forecast endpoint once and shapes the result
// into a Snapshot — current conditions (the first hourly entry), the next
// 24 hours, and today's high/low/sunrise/sunset (from the first daily
// entry), so a single request covers everything the app needs.
func Fetch(ctx context.Context, apiKey string, lat, lng float64) (*Snapshot, error) {
	q := url.Values{
		"location":  {fmt.Sprintf("%f,%f", lat, lng)},
		"timesteps": {"1h,1d"},
		"units":     {"metric"},
		"apikey":    {apiKey},
	}
	reqURL := "https://api.tomorrow.io/v4/weather/forecast?" + q.Encode()

	// Bounded independently of the caller's own context — a slow/hanging
	// Tomorrow.io response shouldn't be able to hold this goroutine open
	// indefinitely just because the request's context has no deadline.
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tomorrow.io: unexpected status %d", res.StatusCode)
	}

	var parsed tomorrowResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("tomorrow.io: decode response: %w", err)
	}
	if len(parsed.Timelines.Hourly) == 0 {
		return nil, fmt.Errorf("tomorrow.io: empty hourly timeline")
	}

	const maxHours = 24
	hourly := make([]HourlyPoint, 0, maxHours)
	for i, h := range parsed.Timelines.Hourly {
		if i >= maxHours {
			break
		}
		t, _ := time.Parse(time.RFC3339, h.Time)
		condition, label, icon := conditionFor(h.Values.WeatherCode)
		hourly = append(hourly, HourlyPoint{
			Time:              t,
			Temp:              h.Values.Temperature,
			Condition:         condition,
			ConditionLabel:    label,
			Icon:              icon,
			PrecipProbability: int(h.Values.PrecipitationProbability),
			PrecipMm:          h.Values.PrecipitationIntensity,
			WindSpeedKmh:      h.Values.WindSpeed * 3.6,
			UVIndex:           h.Values.UVIndex,
			CloudCoverPct:     int(h.Values.CloudCover),
		})
	}

	first := parsed.Timelines.Hourly[0].Values
	condition, label, icon := conditionFor(first.WeatherCode)
	now := Now{
		Temp:              first.Temperature,
		FeelsLike:         first.TemperatureApparent,
		Humidity:          int(first.Humidity),
		WindSpeedKmh:      first.WindSpeed * 3.6,
		WindDirection:     int(first.WindDirection),
		UVIndex:           first.UVIndex,
		VisibilityKm:      first.Visibility,
		PressureHpa:       first.PressureSurfaceLevel,
		PrecipProbability: int(first.PrecipitationProbability),
		CloudCoverPct:     int(first.CloudCover),
		Condition:         condition,
		ConditionLabel:    label,
		Icon:              icon,
	}

	snap := &Snapshot{Now: now, Hourly: hourly, FetchedAt: time.Now()}
	if len(parsed.Timelines.Daily) > 0 {
		d := parsed.Timelines.Daily[0].Values
		snap.High = d.TemperatureMax
		snap.Low = d.TemperatureMin
		snap.Sunrise, _ = time.Parse(time.RFC3339, d.SunriseTime)
		snap.Sunset, _ = time.Parse(time.RFC3339, d.SunsetTime)
	}
	return snap, nil
}

// conditionFor reduces Tomorrow.io's ~30 weather codes
// (https://docs.tomorrow.io/reference/data-layers-weather-codes) to the
// app's existing 3-icon vocabulary (Sun/CloudSun/CloudRain already exist in
// app/lib/core/widgets/spotto_icons.dart) so the client needs no new icon
// map — only the human label is kept from the finer-grained code.
func conditionFor(code int) (condition, label, icon string) {
	switch code {
	case 1000:
		return "sunny", "Clear", "Sun"
	case 1100:
		return "sunny", "Mostly Clear", "Sun"
	case 1101:
		return "cloudy", "Partly Cloudy", "CloudSun"
	case 1102:
		return "cloudy", "Mostly Cloudy", "CloudSun"
	case 1001:
		return "cloudy", "Cloudy", "CloudSun"
	case 2000, 2100:
		return "cloudy", "Foggy", "CloudSun"
	case 4000:
		return "rainy", "Drizzle", "CloudRain"
	case 4001:
		return "rainy", "Rain", "CloudRain"
	case 4200:
		return "rainy", "Light Rain", "CloudRain"
	case 4201:
		return "rainy", "Heavy Rain", "CloudRain"
	case 5000, 5001, 5100, 5101:
		return "rainy", "Snow", "CloudRain"
	case 6000, 6001, 6200, 6201:
		return "rainy", "Freezing Rain", "CloudRain"
	case 7000, 7101, 7102:
		return "rainy", "Ice Pellets", "CloudRain"
	case 8000:
		return "rainy", "Thunderstorm", "CloudRain"
	default:
		return "cloudy", "Unsettled", "CloudSun"
	}
}
