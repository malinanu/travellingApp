// Package weather fetches real conditions from Tomorrow.io or Open-Meteo for the
// app's Weather screen and Discover-tab bar. See internal/http/weather.go for
// the caching HTTP handler that wraps Fetch.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

// HourlyPoint is one hour of forecast.
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

// Now is current conditions.
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

type tomorrowResponse struct {
	Timelines struct {
		Hourly []struct {
			Time   string `json:"time"`
			Values struct {
				Temperature              float64 `json:"temperature"`
				TemperatureApparent      float64 `json:"temperatureApparent"`
				Humidity                 float64 `json:"humidity"`
				WindSpeed                float64 `json:"windSpeed"`
				WindDirection            float64 `json:"windDirection"`
				UVIndex                  float64 `json:"uvIndex"`
				Visibility               float64 `json:"visibility"`
				PressureSurfaceLevel     float64 `json:"pressureSurfaceLevel"`
				PrecipitationProbability float64 `json:"precipitationProbability"`
				PrecipitationIntensity   float64 `json:"precipitationIntensity"`
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

type openMeteoResponse struct {
	Current struct {
		Time                string  `json:"time"`
		Temperature2m       float64 `json:"temperature_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		RelativeHumidity2m  int     `json:"relative_humidity_2m"`
		WindSpeed10m        float64 `json:"wind_speed_10m"`
		WindDirection10m    int     `json:"wind_direction_10m"`
		Precipitation       float64 `json:"precipitation"`
		CloudCover          int     `json:"cloud_cover"`
		PressureMsl         float64 `json:"pressure_msl"`
		WeatherCode         int     `json:"weather_code"`
	} `json:"current"`
	Hourly struct {
		Time                     []string  `json:"time"`
		Temperature2m            []float64 `json:"temperature_2m"`
		PrecipitationProbability []int     `json:"precipitation_probability"`
		Precipitation            []float64 `json:"precipitation"`
		WeatherCode              []int     `json:"weather_code"`
		CloudCover               []int     `json:"cloud_cover"`
		WindSpeed10m             []float64 `json:"wind_speed_10m"`
		UVIndex                  []float64 `json:"uv_index"`
	} `json:"hourly"`
	Daily struct {
		Temperature2mMax []float64 `json:"temperature_2m_max"`
		Temperature2mMin []float64 `json:"temperature_2m_min"`
		Sunrise          []string  `json:"sunrise"`
		Sunset           []string  `json:"sunset"`
	} `json:"daily"`
}

/// Fetch retrieves weather forecast. Tries Tomorrow.io if apiKey is set, otherwise
/// or on failure, falls back to Open-Meteo API (100% free, 0 API key required).
func Fetch(ctx context.Context, apiKey string, lat, lng float64) (*Snapshot, error) {
	if apiKey != "" {
		snap, err := fetchTomorrow(ctx, apiKey, lat, lng)
		if err == nil {
			return snap, nil
		}
		log.Printf("weather: Tomorrow.io failed (%v), falling back to Open-Meteo for (%f,%f)", err, lat, lng)
	}
	return FetchOpenMeteo(ctx, lat, lng)
}

func fetchTomorrow(ctx context.Context, apiKey string, lat, lng float64) (*Snapshot, error) {
	q := url.Values{
		"location":  {fmt.Sprintf("%f,%f", lat, lng)},
		"timesteps": {"1h,1d"},
		"units":     {"metric"},
		"apikey":    {apiKey},
	}
	reqURL := "https://api.tomorrow.io/v4/weather/forecast?" + q.Encode()

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
		return nil, fmt.Errorf("tomorrow.io: status %d", res.StatusCode)
	}

	var parsed tomorrowResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("tomorrow.io: decode: %w", err)
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

/// FetchOpenMeteo fetches real weather from Open-Meteo API (100% free, no API key required).
func FetchOpenMeteo(ctx context.Context, lat, lng float64) (*Snapshot, error) {
	reqURL := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f"+
			"&current=temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,cloud_cover,pressure_msl,wind_speed_10m,wind_direction_10m"+
			"&hourly=temperature_2m,precipitation_probability,precipitation,weather_code,cloud_cover,wind_speed_10m,uv_index"+
			"&daily=temperature_2m_max,temperature_2m_min,sunrise,sunset"+
			"&timezone=Asia%%2FColombo&forecast_days=2",
		lat, lng,
	)

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
		return nil, fmt.Errorf("open-meteo: status %d", res.StatusCode)
	}

	var parsed openMeteoResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("open-meteo: decode: %w", err)
	}

	c := parsed.Current
	condition, label, icon := conditionForWMO(c.WeatherCode)
	now := Now{
		Temp:              c.Temperature2m,
		FeelsLike:         c.ApparentTemperature,
		Humidity:          c.RelativeHumidity2m,
		WindSpeedKmh:      c.WindSpeed10m,
		WindDirection:     c.WindDirection10m,
		UVIndex:           5.0,
		VisibilityKm:      10.0,
		PressureHpa:       c.PressureMsl,
		PrecipProbability: int(c.Precipitation * 10),
		CloudCoverPct:     c.CloudCover,
		Condition:         condition,
		ConditionLabel:    label,
		Icon:              icon,
	}
	if len(parsed.Hourly.UVIndex) > 0 {
		now.UVIndex = parsed.Hourly.UVIndex[0]
	}

	hourly := make([]HourlyPoint, 0, 24)
	for i := 0; i < len(parsed.Hourly.Time) && i < 24; i++ {
		t, _ := time.Parse("2006-01-02T15:04", parsed.Hourly.Time[i])
		code := parsed.Hourly.WeatherCode[i]
		hCond, hLabel, hIcon := conditionForWMO(code)
		precipProb := 0
		if i < len(parsed.Hourly.PrecipitationProbability) {
			precipProb = parsed.Hourly.PrecipitationProbability[i]
		}
		uv := 0.0
		if i < len(parsed.Hourly.UVIndex) {
			uv = parsed.Hourly.UVIndex[i]
		}
		precipMm := 0.0
		if i < len(parsed.Hourly.Precipitation) {
			precipMm = parsed.Hourly.Precipitation[i]
		}
		windSpeed := 0.0
		if i < len(parsed.Hourly.WindSpeed10m) {
			windSpeed = parsed.Hourly.WindSpeed10m[i]
		}
		cloudCover := 0
		if i < len(parsed.Hourly.CloudCover) {
			cloudCover = parsed.Hourly.CloudCover[i]
		}
		temp := 0.0
		if i < len(parsed.Hourly.Temperature2m) {
			temp = parsed.Hourly.Temperature2m[i]
		}

		hourly = append(hourly, HourlyPoint{
			Time:              t,
			Temp:              temp,
			Condition:         hCond,
			ConditionLabel:    hLabel,
			Icon:              hIcon,
			PrecipProbability: precipProb,
			PrecipMm:          precipMm,
			WindSpeedKmh:      windSpeed,
			UVIndex:           uv,
			CloudCoverPct:     cloudCover,
		})
	}

	snap := &Snapshot{Now: now, Hourly: hourly, FetchedAt: time.Now()}
	if len(parsed.Daily.Temperature2mMax) > 0 {
		snap.High = parsed.Daily.Temperature2mMax[0]
		snap.Low = parsed.Daily.Temperature2mMin[0]
	}
	if len(parsed.Daily.Sunrise) > 0 {
		snap.Sunrise, _ = time.Parse("2006-01-02T15:04", parsed.Daily.Sunrise[0])
	}
	if len(parsed.Daily.Sunset) > 0 {
		snap.Sunset, _ = time.Parse("2006-01-02T15:04", parsed.Daily.Sunset[0])
	}

	return snap, nil
}

func conditionForWMO(code int) (condition, label, icon string) {
	switch code {
	case 0:
		return "sunny", "Clear", "Sun"
	case 1:
		return "sunny", "Mainly Clear", "Sun"
	case 2:
		return "cloudy", "Partly Cloudy", "CloudSun"
	case 3:
		return "cloudy", "Overcast", "CloudSun"
	case 45, 48:
		return "cloudy", "Foggy", "CloudSun"
	case 51, 53, 55:
		return "rainy", "Drizzle", "CloudRain"
	case 61, 63, 65, 80, 81, 82:
		return "rainy", "Rain", "CloudRain"
	case 95, 96, 99:
		return "rainy", "Thunderstorm", "CloudRain"
	default:
		return "cloudy", "Partly Cloudy", "CloudSun"
	}
}

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
