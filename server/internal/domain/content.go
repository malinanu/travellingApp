package domain

import "time"

// Travel content written in the Command Center by admins and publishers and
// read by travellers in the app and on the website (migration 049).

const (
	ContentAudio     = "audio"     // audio guide or audiobook
	ContentBlog      = "blog"      // blog post
	ContentGuide     = "guide"     // long-form travel guide with linked places
	ContentItinerary = "itinerary" // day-by-day plan travellers can copy into a trip
	ContentTip       = "tip"       // short tip or news item
)

// ContentTypes lists every content type, in the order the clients show them.
var ContentTypes = []string{ContentAudio, ContentBlog, ContentGuide, ContentItinerary, ContentTip}

// A post moves draft → in_review (a publisher submits) → published, or back
// to rejected with a note. Admins can publish without a review; unpublishing
// archives it. Publishers can edit only draft, rejected and archived posts.
const (
	ContentDraft     = "draft"
	ContentInReview  = "in_review"
	ContentPublished = "published"
	ContentRejected  = "rejected"
	ContentArchived  = "archived"
)

// ContentChapter is a point in an audio file listeners can jump to.
type ContentChapter struct {
	Title    string `json:"title"`
	StartSec int    `json:"startSec"`
}

// ItineraryStop is one stop of an itinerary day: a Spotto place (PlaceID
// set) or a named point, optionally at a time of day.
type ItineraryStop struct {
	PlaceID string   `json:"placeId,omitempty"`
	Name    string   `json:"name"`
	Lat     *float64 `json:"lat,omitempty"`
	Lng     *float64 `json:"lng,omitempty"`
	Time    string   `json:"time,omitempty"` // "09:30"
	Note    string   `json:"note,omitempty"`
}

type ItineraryDay struct {
	Title string          `json:"title"`
	Stops []ItineraryStop `json:"stops"`
}

// ContentPlace is a linked place as shown with a published post.
type ContentPlace struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Cover    string  `json:"cover,omitempty"`
}

type ContentItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	// Body is Markdown. List responses leave it out.
	Body string `json:"body,omitempty"`
	// CoverURL and AudioURL are signed "/media/raw/…" paths, relative to
	// the API, or empty.
	CoverURL     string           `json:"coverUrl"`
	AudioURL     string           `json:"audioUrl"`
	AudioSeconds int              `json:"audioSeconds"`
	Chapters     []ContentChapter `json:"chapters"`
	PlaceIDs     []string         `json:"placeIds"`
	Days         []ItineraryDay   `json:"days"`
	Tags         []string         `json:"tags"`
	RegionID     *string          `json:"regionId"`
	RegionName   string           `json:"regionName"`
	// RegionLat/RegionLng are the town's centre, for "near you" sorting.
	RegionLat   *float64   `json:"regionLat,omitempty"`
	RegionLng   *float64   `json:"regionLng,omitempty"`
	Language    string     `json:"language"`
	AuthorID    *string    `json:"authorId"`
	AuthorName  string     `json:"authorName"`
	Status      string     `json:"status"`
	ReviewNote  string     `json:"reviewNote"`
	ReviewedBy  *string    `json:"reviewedBy"`
	SubmittedAt *time.Time `json:"submittedAt"`
	PublishedAt *time.Time `json:"publishedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	Views       int        `json:"views"`
	Plays       int        `json:"plays"`
	// Places is filled on a published post's detail: its linked places and
	// the Spotto places its itinerary stops at.
	Places []ContentPlace `json:"places,omitempty"`
}
