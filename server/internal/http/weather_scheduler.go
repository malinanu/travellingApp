package httpapi

// The weather scheduler proactively warms the weather cache for every
// distinct location the real vendor dataset spans, so a traveller's live
// /weather request almost always hits an already-warm cache instead of
// spending one of Tomorrow.io's tight free-tier calls (25/hour, 500/day)
// per request. Mirrors the holiday scheduler's shape (see
// holiday_scheduler.go): runs once immediately — so a fresh deploy/restart
// (e.g. pulling a new image in Portainer, which restarts the whole
// container) repopulates the cache right away rather than waiting on the
// first traveller to hit each cell — then on a recurring tick.

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/spotto/server/internal/weather"
)

// weatherSchedulerInterval is kept comfortably under weather.TTL (60 min)
// so a cell is always refreshed before it goes stale. At the dataset's
// ~12-15 distinct location cells, a sweep every 45 min costs roughly
// 16-20 calls/hour and under 400/day — well inside Tomorrow.io's
// free-tier limits (25/hour, 500/day), leaving headroom for the rare
// on-demand fetch from a cell this sweep hasn't covered yet.
const weatherSchedulerInterval = 45 * time.Minute

// weatherCallSpacing paces requests within one sweep so a larger vendor
// footprint could never burst past the free tier's 3 req/sec cap.
const weatherCallSpacing = 500 * time.Millisecond

// StartWeatherScheduler blocks until ctx is cancelled; call with `go`.
func (s *Server) StartWeatherScheduler(ctx context.Context) {
	s.runWeatherWarm(ctx)
	ticker := time.NewTicker(weatherSchedulerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runWeatherWarm(ctx)
		}
	}
}

func (s *Server) runWeatherWarm(ctx context.Context) {
	if s.TomorrowIOAPIKey == "" {
		return // unconfigured — same graceful no-op as handleWeather
	}
	cells, err := s.Store.DistinctVendorCells(ctx)
	if err != nil {
		log.Printf("weather scheduler: list vendor cells: %v", err)
		return
	}

	warmed, failed := 0, 0
	for i, c := range cells {
		if i > 0 {
			time.Sleep(weatherCallSpacing)
		}
		key := weather.CellKey(c.Lat, c.Lng)
		snap, err := weather.Fetch(ctx, s.TomorrowIOAPIKey, c.Lat, c.Lng)
		if err != nil {
			log.Printf("weather scheduler: fetch cell=%s: %v", key, err)
			failed++
			continue
		}
		s.WeatherCache.Set(key, snap)
		if raw, err := json.Marshal(snap); err != nil {
			log.Printf("weather scheduler: marshal cell=%s: %v", key, err)
		} else if err := s.Store.SetWeatherCache(ctx, key, raw, time.Now().Add(weather.TTL)); err != nil {
			log.Printf("weather scheduler: persist cell=%s: %v", key, err)
		}
		warmed++
	}
	log.Printf("weather scheduler: warmed %d/%d location cells", warmed, warmed+failed)
}
