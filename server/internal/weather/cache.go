package weather

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// Cache is a small in-memory (no Redis — single-instance deployment, same
// philosophy as internal/http/ratelimit.go's loginLimiter) TTL cache keyed
// by a coarse ~55km (0.5°) grid cell rather than exact coordinates. Sri
// Lanka spans roughly 10-15 such cells, so even with every cell cold this
// stays well under Tomorrow.io's free-tier limits (25 req/hour, 500/day)
// at a 60-minute TTL.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	snapshot  *Snapshot
	expiresAt time.Time
}

// TTL is how long a cached snapshot is served before a fresh Tomorrow.io
// call is made for that cell. Weather doesn't change fast enough hour-to-
// hour to need anything shorter, and a shorter TTL risks the free-tier cap
// under concurrent use across many cells. Exported so the Postgres-backed
// persistence layer (internal/http/weather.go) can stamp the same expiry.
const TTL = 60 * time.Minute

func NewCache() *Cache {
	return &Cache{entries: make(map[string]cacheEntry)}
}

// CellKey rounds lat/lng to the nearest 0.5° so nearby requests share one
// cache entry instead of fragmenting by exact GPS coordinate.
func CellKey(lat, lng float64) string {
	round := func(v float64) float64 { return math.Round(v*2) / 2 }
	return fmt.Sprintf("%.1f,%.1f", round(lat), round(lng))
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
