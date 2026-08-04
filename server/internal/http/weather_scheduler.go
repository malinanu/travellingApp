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
//
// Every sweep goes through getOrFetchWeather (weather.go), the same
// cache-then-fetch path handleWeather uses, so a cell whose cache is still
// fresh is skipped rather than re-fetched — the sweep only spends real
// Tomorrow.io calls on cells that actually need it. This matters because
// the dataset spans ~25 distinct cells today (every imported spot becomes
// its own vendor with real coordinates — see importsrilanka), not the
// ~12-15 this scheduler originally assumed: an unconditional sweep of 25
// cells alone would exhaust the entire 25-req/hour free tier in one pass,
// leaving nothing for live traveller requests. See weather.TTL for the
// other half of the fix — the TTL controls the steady-state call rate
// once cells are warm.

import (
	"context"
	"log"
	"time"
)

// weatherSchedulerInterval is kept comfortably under weather.TTL so a cell
// is always checked well before it goes stale. Since getOrFetchWeather
// skips cells whose cache is still fresh, running this more often than
// strictly needed is cheap — it just means most cells are no-ops most of
// the time, cells only cost a real call once per TTL window.
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
	cells, err := s.Store.DistinctVendorCells(ctx)
	if err != nil {
		log.Printf("weather scheduler: list vendor cells: %v", err)
		return
	}

	fetched, skipped, failed := 0, 0, 0
	for i, c := range cells {
		if i > 0 {
			time.Sleep(weatherCallSpacing)
		}
		_, fromCache, err := s.getOrFetchWeather(ctx, c.Lat, c.Lng)
		if err != nil {
			failed++
			continue
		}
		if fromCache {
			skipped++
		} else {
			fetched++
		}
	}
	log.Printf("weather scheduler: %d/%d cells already warm, %d fetched, %d failed",
		skipped, len(cells), fetched, failed)
}
