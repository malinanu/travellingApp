// Package placedetails fetches live traveller-relevant details (phone,
// website, wheelchair accessibility, price level) from the Google Places
// Details API for imported spots, and caches them — these attributes were
// never captured by the original sri_lanka_spots.json scrape (see
// cmd/importsrilanka) and Google bills per Details call, so results are
// cached long-term rather than re-fetched on every request. See
// internal/weather for the same in-memory + Postgres two-tier cache
// pattern this package follows.
package placedetails

import (
	"sync"
	"time"
)

// Details is the subset of a Google Places Details response the app
// surfaces to travellers.
type Details struct {
	Phone                string `json:"phone"`
	Website              string `json:"website"`
	WheelchairAccessible *bool  `json:"wheelchairAccessible,omitempty"`
	PriceLevel           string `json:"priceLevel"` // $/$$/$$$/$$$$/"Free", "" if unknown
}

// TTL is how long a cached result is served before a fresh Google call is
// made for that place. Unlike weather, these attributes rarely change, so
// a long TTL both keeps paid API spend down and rarely shows stale data.
const TTL = 30 * 24 * time.Hour

// Cache is a small in-memory TTL cache keyed by Google place ID — same
// shape as internal/weather.Cache, just keyed and expired differently.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	details   *Details
	expiresAt time.Time
}

func NewCache() *Cache {
	return &Cache{entries: make(map[string]cacheEntry)}
}

func (c *Cache) Get(googlePlaceID string) (*Details, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[googlePlaceID]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.details, true
}

func (c *Cache) Set(googlePlaceID string, details *Details) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[googlePlaceID] = cacheEntry{details: details, expiresAt: time.Now().Add(TTL)}
}
