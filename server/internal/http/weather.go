package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/spotto/server/internal/weather"
)

// handleWeather serves real conditions for the app's Discover-tab bar and
// Weather screen. Two cache layers guard Tomorrow.io's tight free-tier
// limit (25 req/hour, 500/day): internal/weather.Cache in memory (fast
// path), backed by the weather_cache Postgres table (see migrations/013_
// weather_cache.sql) so a process restart — which happens on every deploy,
// since importsrilanka runs before the API starts — doesn't throw away a
// still-valid cached response and force an immediate re-fetch.
func (s *Server) handleWeather(w http.ResponseWriter, r *http.Request) {
	lat, latErr := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, lngErr := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if latErr != nil || lngErr != nil {
		writeErr(w, http.StatusBadRequest, "lat and lng query params are required")
		return
	}
	if s.TomorrowIOAPIKey == "" {
		writeErr(w, http.StatusNotImplemented, "weather not configured")
		return
	}

	key := weather.CellKey(lat, lng)
	if snap, ok := s.WeatherCache.Get(key); ok {
		writeJSON(w, http.StatusOK, snap)
		return
	}

	if raw, ok, err := s.Store.GetWeatherCache(r.Context(), key); err != nil {
		log.Printf("weather: read persisted cache for cell=%s: %v", key, err)
	} else if ok {
		var snap weather.Snapshot
		if err := json.Unmarshal(raw, &snap); err == nil {
			s.WeatherCache.Set(key, &snap)
			writeJSON(w, http.StatusOK, snap)
			return
		}
	}

	snap, err := weather.Fetch(r.Context(), s.TomorrowIOAPIKey, lat, lng)
	if err != nil {
		log.Printf("weather: fetch failed for cell=%s: %v", key, err)
		writeErr(w, http.StatusBadGateway, "weather unavailable")
		return
	}
	s.WeatherCache.Set(key, snap)
	if raw, err := json.Marshal(snap); err != nil {
		log.Printf("weather: marshal cache for cell=%s: %v", key, err)
	} else if err := s.Store.SetWeatherCache(r.Context(), key, raw, time.Now().Add(weather.TTL)); err != nil {
		log.Printf("weather: persist cache for cell=%s: %v", key, err)
	}
	writeJSON(w, http.StatusOK, snap)
}
