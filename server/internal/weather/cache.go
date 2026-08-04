package weather

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// Cache is a small in-memory (no Redis — single-instance deployment, same
// philosophy as internal/http/ratelimit.go's loginLimiter) TTL cache keyed
// by a coarse ~55km (0.5°) grid cell rather than exact coordinates. The
// vendor dataset spans roughly 25 such cells (every imported spot becomes
// its own vendor with real coordinates — see importsrilanka), and the
// background scheduler (internal/http/weather_scheduler.go) re-warms every
// cell roughly once per TTL window. At 25 cells, the TTL has to be long
// enough that cells/TTL stays comfortably under Tomorrow.io's free-tier
// cap (25 req/hour, 500/day) with headroom left for live traveller
// requests in the same hour — a 60-minute TTL leaves none at all.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	snapshot  *Snapshot
	expiresAt time.Time
}

// TTL is how long a cached snapshot is served before a fresh Tomorrow.io
// call is made for that cell. Weather doesn't change fast enough over a
// few hours to need anything shorter, and a shorter TTL risks the
// free-tier cap under concurrent use across ~25 cells (see Cache's doc
// comment). Exported so the Postgres-backed persistence layer
// (internal/http/weather.go) can stamp the same expiry.
const TTL = 3 * time.Hour

func NewCache() *Cache {
	return &Cache{entries: make(map[string]cacheEntry)}
}

// CellKey rounds lat/lng to the nearest 0.2° (~20km) so all users in the same region
// share the exact same cached weather snapshot instead of fragmenting by exact GPS coordinate.
func CellKey(lat, lng float64) string {
	round := func(v float64) float64 { return math.Round(v*5) / 5 }
	return fmt.Sprintf("%.2f,%.2f", round(lat), round(lng))
}

func (c *Cache) Get(key string) (*Snapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.snapshot, true
}

func (c *Cache) Set(key string, snapshot *Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{snapshot: snapshot, expiresAt: time.Now().Add(TTL)}
}
