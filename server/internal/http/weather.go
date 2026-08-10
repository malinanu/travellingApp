package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/spotto/server/internal/weather"
)

// handleWeather serves real conditions for the app's Discover-tab bar and
// Weather screen.
func (s *Server) handleWeather(w http.ResponseWriter, r *http.Request) {
	lat, latErr := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, lngErr := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if latErr != nil || lngErr != nil {
		writeErr(w, http.StatusBadRequest, "lat and lng query params are required")
		return
	}

	snap, _, err := s.getOrFetchWeather(r.Context(), lat, lng)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "weather unavailable")
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// handleWeatherBatch retrieves weather for multiple locations in a single API request,
// saving network overhead and API call costs.
func (s *Server) handleWeatherBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Locations []struct {
			ID  string  `json:"id"`
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"locations"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	results := make(map[string]*weather.Snapshot)
	for _, loc := range req.Locations {
		snap, _, err := s.getOrFetchWeather(r.Context(), loc.Lat, loc.Lng)
		if err == nil && snap != nil {
			results[loc.ID] = snap
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

// getOrFetchWeather is the single three-tier cache path shared by
// handleWeather (live traveller requests) and the background scheduler
// (weather_scheduler.go): in-memory cache, then the Postgres-backed cache
// (survives a process restart, e.g. every deploy), and only then a real
// Tomorrow.io or Open-Meteo call â€” so nothing burns quota on a cell that's
// already warm.
func (s *Server) getOrFetchWeather(ctx context.Context, lat, lng float64) (snap *weather.Snapshot, fromCache bool, err error) {
	key := weather.CellKey(lat, lng)
	if cached, ok := s.WeatherCache.Get(key); ok {
		return cached, true, nil
	}

	if raw, ok, err := s.Store.GetWeatherCache(ctx, key); err != nil {
		log.Printf("weather: read persisted cache for cell=%s: %v", key, err)
	} else if ok {
		var cached weather.Snapshot
		if err := json.Unmarshal(raw, &cached); err == nil {
			s.WeatherCache.Set(key, &cached)
			return &cached, true, nil
		}
	}

	fresh, err := weather.Fetch(ctx, s.TomorrowIOAPIKey, lat, lng)
	if err != nil {
		log.Printf("weather: fetch failed for cell=%s: %v", key, err)
		return nil, false, err
	}
	s.WeatherCache.Set(key, fresh)
	if raw, err := json.Marshal(fresh); err != nil {
		log.Printf("weather: marshal cache for cell=%s: %v", key, err)
	} else if err := s.Store.SetWeatherCache(ctx, key, raw, time.Now().Add(weather.TTL)); err != nil {
		log.Printf("weather: persist cache for cell=%s: %v", key, err)
	}
	return fresh, false, nil
}
