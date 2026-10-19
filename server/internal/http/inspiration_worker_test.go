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
	"github.com/spotto/server/internal/googleplaces"
	"github.com/spotto/server/internal/store/postgres"
)

func TestCanonicalLink(t *testing.T) {
	cases := []struct {
		in, want, platform string
	}{
		{"https://www.instagram.com/reels/DbQSwgwMM_n/", "https://www.instagram.com/reel/DbQSwgwMM_n/", "instagram"},
		{"https://instagram.com/reel/DbQSwgwMM_n/?igsh=abc&utm_source=ig_web", "https://www.instagram.com/reel/DbQSwgwMM_n/", "instagram"},
		{"instagram.com/p/Cx1y2Z3/", "https://www.instagram.com/p/Cx1y2Z3/", "instagram"},
		{"https://www.instagram.com/some.user/reel/Abc_12-3/", "https://www.instagram.com/reel/Abc_12-3/", "instagram"},
		{"https://www.tiktok.com/@phuket.eats/video/7312345678901234567?is_from_webapp=1", "https://www.tiktok.com/@phuket.eats/video/7312345678901234567", "tiktok"},
		{"https://vm.tiktok.com/ZMabc123/", "https://vm.tiktok.com/ZMabc123/", "tiktok"},
		{"https://youtube.com/shorts/aBcDeFgHiJk?si=x", "https://www.youtube.com/shorts/aBcDeFgHiJk", "youtube"},
		{"https://youtu.be/aBcDeFgHiJk", "https://www.youtube.com/watch?v=aBcDeFgHiJk", "youtube"},
	}
	for _, c := range cases {
		got, platform, err := canonicalLink(c.in)
		if err != nil || got != c.want || platform != c.platform {
			t.Errorf("canonicalLink(%q) = %q, %q, %v; want %q, %q", c.in, got, platform, err, c.want, c.platform)
		}
	}
	for _, bad := range []string{
		"", "https://instagram.com/", "https://instagram.com.evil.example/reel/abc/",
		"ftp://instagram.com/reel/abc/", "https://example.com/reel/abc/", "https://www.instagram.com/explore/",
	} {
		if got, _, err := canonicalLink(bad); err == nil {
			t.Errorf("canonicalLink(%q) = %q, want an error", bad, got)
		}
	}
}

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
	job = domain.LinkExtraction{Status: domain.ExtractionFailed, Error: msgCantOpen}
	if title, body := extractionPushText(job); title != "We couldn't read that link" || body != msgCantOpen {
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

	if err := store.RetryExtractionLater(ctx, url, msgCantOpen, time.Hour); err != nil {
		t.Fatal(err)
	}
	if e, _ := store.LinkExtraction(ctx, url); e.Status != domain.ExtractionPending || e.Error != msgCantOpen {
		t.Errorf("after retry later = %+v", e)
	}

	done := domain.LinkExtraction{CanonicalURL: url, Status: domain.ExtractionFailed, Error: msgCantOpen}
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
