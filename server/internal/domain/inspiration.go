package domain

import "time"

// GeoPlace is a real-world place found through Google Places (anywhere in
// the world, unlike the Sri Lanka-only Place catalog), cached for reuse.
type GeoPlace struct {
	ID            string    `json:"id"`
	GooglePlaceID string    `json:"googlePlaceId"`
	Name          string    `json:"name"`
	Lat           float64   `json:"lat"`
	Lng           float64   `json:"lng"`
	Address       string    `json:"address"`
	City          string    `json:"city"`
	CountryCode   string    `json:"countryCode"`
	Kind          string    `json:"kind"` // eat | drink | stay | see | do | shop
	Rating        *float64  `json:"rating,omitempty"`
	RatingCount   int       `json:"ratingCount"`
	PhotoName     string    `json:"-"`
	HasPhoto      bool      `json:"hasPhoto"`
	MapsURL       string    `json:"mapsUrl"`
	Types         []string  `json:"types"`
	FetchedAt     time.Time `json:"fetchedAt"`
}

// Link extraction statuses.
const (
	ExtractionPending    = "pending"
	ExtractionProcessing = "processing"
	ExtractionDone       = "done"
	ExtractionPartial    = "partial" // video couldn't be read; places come from the caption only
	ExtractionFailed     = "failed"
)

// ExtractedPlace is one place found in a reel. GeoPlaceID is set once it was
// found on Google; PlaceID when it is also a Spotto catalog place.
type ExtractedPlace struct {
	Name       string  `json:"name"`
	Kind       string  `json:"kind"`
	Area       string  `json:"area,omitempty"`
	City       string  `json:"city,omitempty"`
	Country    string  `json:"country,omitempty"`
	Evidence   string  `json:"evidence,omitempty"` // caption | speech | on_screen
	AtSec      *int    `json:"atSec,omitempty"`
	Confidence float64 `json:"confidence"`
	GeoPlaceID string  `json:"geoPlaceId,omitempty"`
	PlaceID    string  `json:"placeId,omitempty"`
}

// LinkExtraction is the shared result for one canonical link.
type LinkExtraction struct {
	CanonicalURL string           `json:"canonicalUrl"`
	Platform     string           `json:"platform"`
	Status       string           `json:"status"`
	Error        string           `json:"error,omitempty"`
	Caption      string           `json:"caption,omitempty"`
	Author       string           `json:"author,omitempty"`
	ThumbnailURL string           `json:"thumbnailUrl,omitempty"`
	Results      []ExtractedPlace `json:"results"`
	Model        string           `json:"-"`
	Attempts     int              `json:"-"`
	// What the job cost: downloaded video size and length, and wall time.
	VideoBytes   int64     `json:"-"`
	VideoSecs    float64   `json:"-"`
	ProcessingMs int       `json:"-"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// InspirationLink is a link a traveller saved to their board.
type InspirationLink struct {
	ID           string    `json:"id"`
	UserID       string    `json:"-"`
	URL          string    `json:"url"`
	CanonicalURL string    `json:"canonicalUrl"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"createdAt"`
}

// SavedPlace is a "save for later" entry: either a catalog place or a geo place.
type SavedPlace struct {
	ID           string    `json:"id"`
	UserID       string    `json:"-"`
	PlaceID      string    `json:"placeId,omitempty"`
	GeoPlaceID   string    `json:"geoPlaceId,omitempty"`
	SourceLinkID string    `json:"sourceLinkId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
