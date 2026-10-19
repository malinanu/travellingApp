package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/googleplaces"
	"github.com/spotto/server/internal/reelextract"
	"github.com/spotto/server/internal/store/postgres"
)

func TestNameSimilarity(t *testing.T) {
	cases := []struct {
		a, b string
		min  float64
		max  float64
	}{
		{"Kata Noodle House", "Kata Noodle House Phuket", 1, 1},
		{"Big Buddha", "The Big Buddha Phuket", 1, 1},
		{"Café del Mar", "Cafe Del Mar Phuket", 0.99, 1},
		{"Jungle Beach Cafe", "Jungle Beach", 1, 1},
		{"Kata Noodle House", "Patong Seafood", 0, 0},
		{"Sunset Bar", "Sunrise Bar", 0, 0}, // "bar" alone says nothing
	}
	for _, c := range cases {
		got := nameSimilarity(c.a, c.b)
		if got < c.min || got > c.max {
			t.Errorf("nameSimilarity(%q, %q) = %.2f, want %.2f..%.2f", c.a, c.b, got, c.min, c.max)
		}
	}
}

func TestPlaceQuery(t *testing.T) {
	ep := domain.ExtractedPlace{Name: "Kata Noodle House", Area: "Kata Beach", City: "Phuket", Country: "Thailand"}
	if got := placeQuery(ep, "Bangkok"); got != "Kata Noodle House, Kata Beach, Phuket, Thailand" {
		t.Errorf("placeQuery = %q", got)
	}
	if got := placeQuery(domain.ExtractedPlace{Name: "Big Buddha"}, "Phuket, Thailand"); got != "Big Buddha, Phuket, Thailand" {
		t.Errorf("placeQuery with region hint = %q", got)
	}
}

func TestExtractionPushText(t *testing.T) {
	job := domain.LinkExtraction{Status: domain.ExtractionDone, Author: "phuket.eats", Results: make([]domain.ExtractedPlace, 14)}
	if title, body := extractionPushText(job); title != "14 places found" || body == "" {
		t.Errorf("push text = %q / %q", title, body)
	}
	job = domain.LinkExtraction{Status: domain.ExtractionFailed, Error: reelextract.MsgCantOpen}
	if title, body := extractionPushText(job); title != "We couldn't read that link" || body != reelextract.MsgCantOpen {
		t.Errorf("failed push text = %q / %q", title, body)
	}
}

func connectTestStore(t *testing.T) *postgres.Store {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	store, err := postgres.Connect(ctx, "postgres://postgres:admin123@localhost:5432/spotto?sslmode=disable")
	if err != nil {
		t.Skipf("skipping: no local postgres available: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

// The resolver must store Google's place, serve repeats from the query
// cache without calling Google again, and treat an unrelated top result as
// no match.
func TestResolveExtractedPlace(t *testing.T) {
	store := connectTestStore(t)
	suffix := newID("t")
	var calls atomic.Int32
	google := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Goog-Api-Key") != "test-key" || r.Header.Get("X-Goog-FieldMask") == "" {
			t.Errorf("missing key or field mask headers")
		}
		var req struct {
			TextQuery string `json:"textQuery"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		name := "Kata Noodle House " + suffix
		if req.TextQuery == "Mystery Spot "+suffix+", Phuket, Thailand" {
			name = "Patong Seafood Market"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"places": []any{map[string]any{
			"id":               "gid_" + suffix + "_" + req.TextQuery,
			"displayName":      map[string]string{"text": name},
			"location":         map[string]float64{"latitude": 7.8206, "longitude": 98.2985},
			"formattedAddress": "Kata, Phuket 83100, Thailand",
			"rating":           4.6,
			"userRatingCount":  812,
			"primaryType":      "thai_restaurant",
			"types":            []string{"thai_restaurant", "restaurant", "food"},
			"photos":           []any{map[string]string{"name": "places/x/photos/y"}},
			"addressComponents": []any{
				map[string]any{"longText": "Phuket", "shortText": "Phuket", "types": []string{"locality"}},
				map[string]any{"longText": "Thailand", "shortText": "TH", "types": []string{"country"}},
			},
		}}})
	}))
	defer google.Close()

	s := &Server{Store: store, Places: &googleplaces.Client{APIKey: "test-key", BaseURL: google.URL, HTTP: google.Client()}}
	ctx := context.Background()

	ep := domain.ExtractedPlace{Name: "Kata Noodle House " + suffix, Kind: "eat", City: "Phuket", Country: "Thailand"}
	s.resolveExtractedPlace(ctx, &ep, "")
	if ep.GeoPlaceID == "" {
		t.Fatal("place was not resolved")
	}
	gp, err := store.GeoPlaceByID(ctx, ep.GeoPlaceID)
	if err != nil {
		t.Fatal(err)
	}
	if gp.Kind != "eat" || gp.CountryCode != "TH" || gp.City != "Phuket" || !gp.HasPhoto {
		t.Errorf("stored geo place = %+v", gp)
	}

	again := domain.ExtractedPlace{Name: ep.Name, Kind: "eat", City: "Phuket", Country: "Thailand"}
	s.resolveExtractedPlace(ctx, &again, "")
	if again.GeoPlaceID != ep.GeoPlaceID || calls.Load() != 1 {
		t.Errorf("repeat lookup: id %q (want %q), google calls %d (want 1)", again.GeoPlaceID, ep.GeoPlaceID, calls.Load())
	}

	odd := domain.ExtractedPlace{Name: "Mystery Spot " + suffix, City: "Phuket", Country: "Thailand"}
	s.resolveExtractedPlace(ctx, &odd, "")
	if odd.GeoPlaceID != "" {
		t.Errorf("unrelated Google result was accepted as %q", odd.GeoPlaceID)
	}
}

// A job moves pending → processing on claim, goes back to pending with a
// retry, and only the requeue of a finished job starts it over.
func TestExtractionQueue(t *testing.T) {
	store := connectTestStore(t)
	ctx := context.Background()
	url := "https://www.instagram.com/reel/" + newID("test") + "/"
	if err := store.EnsureLinkExtraction(ctx, url, "instagram"); err != nil {
		t.Fatal(err)
	}

	var claimed domain.LinkExtraction
	for i := 0; i < 50; i++ { // other pending rows may be ahead in a shared dev DB
		job, err := store.ClaimPendingExtraction(ctx)
		if err != nil {
			t.Fatalf("claim: %v", err)
		}
		if job.CanonicalURL == url {
			claimed = job
			break
		}
		_ = store.RetryExtractionLater(ctx, job.CanonicalURL, "", time.Hour)
	}
	if claimed.Status != domain.ExtractionProcessing || claimed.Attempts != 1 {
		t.Fatalf("claimed = %+v", claimed)
	}

	if err := store.RetryExtractionLater(ctx, url, reelextract.MsgCantOpen, time.Hour); err != nil {
		t.Fatal(err)
	}
	if e, _ := store.LinkExtraction(ctx, url); e.Status != domain.ExtractionPending || e.Error != reelextract.MsgCantOpen {
		t.Errorf("after retry later = %+v", e)
	}

	done := domain.LinkExtraction{CanonicalURL: url, Status: domain.ExtractionFailed, Error: reelextract.MsgCantOpen}
	if err := store.FinishExtraction(ctx, done); err != nil {
		t.Fatal(err)
	}
	if err := store.RequeueExtraction(ctx, url); err != nil {
		t.Fatal(err)
	}
	if e, _ := store.LinkExtraction(ctx, url); e.Status != domain.ExtractionPending || e.Error != "" || e.Attempts != 0 {
		t.Errorf("after requeue = %+v", e)
	}
}

// fakeReels stands in for the extraction service.
type fakeReels struct {
	res reelextract.Result
	err error
}

func (f fakeReels) Extract(context.Context, string, string) (reelextract.Result, error) {
	return f.res, f.err
}

// The worker turns the extractor's outcome into the job's status: places
// (done), caption-only places (partial), a retryable failure (back in the
// queue) and a final one (failed), and records what the job cost.
func TestProcessExtractionOutcomes(t *testing.T) {
	store := connectTestStore(t)
	ctx := context.Background()
	run := func(reels reelextract.Interface) domain.LinkExtraction {
		t.Helper()
		url := "https://www.instagram.com/reel/" + newID("w") + "/"
		if err := store.EnsureLinkExtraction(ctx, url, "instagram"); err != nil {
			t.Fatal(err)
		}
		job, _ := store.LinkExtraction(ctx, url)
		job.Attempts = 1
		(&Server{Store: store, Reels: reels}).processExtraction(ctx, job)
		out, err := store.LinkExtraction(ctx, url)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	two := []gemini.VideoPlace{{Name: "Big Buddha", Kind: "see"}, {Name: "Kata Noodle House", Kind: "eat"}}
	if e := run(fakeReels{res: reelextract.Result{Author: "itsemandty", Places: two, VideoBytes: 3 << 20, VideoSecs: 18}}); e.Status != domain.ExtractionDone || len(e.Results) != 2 || e.Author != "itsemandty" {
		t.Errorf("done: %+v", e)
	}
	if e := run(fakeReels{res: reelextract.Result{Places: two[:1], PartialReason: reelextract.MsgCaptionOnly}}); e.Status != domain.ExtractionPartial || e.Error != reelextract.MsgCaptionOnly {
		t.Errorf("partial: %+v", e)
	}
	if e := run(fakeReels{err: &reelextract.Error{Message: reelextract.MsgCantOpen, Retryable: true}}); e.Status != domain.ExtractionPending || e.Error != reelextract.MsgCantOpen {
		t.Errorf("retryable: %+v", e)
	}
	if e := run(fakeReels{err: &reelextract.Error{Message: reelextract.MsgNothingToRead}}); e.Status != domain.ExtractionFailed || e.Error != reelextract.MsgNothingToRead {
		t.Errorf("final: %+v", e)
	}
}
