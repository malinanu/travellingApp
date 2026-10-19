package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

func TestInspirationAndSavedPlacesAPI(t *testing.T) {
	store := mustConnect(t)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000", InspirationDailyLimit: 2}
	srv := httptest.NewServer(httpapi.New(store, auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg).Router(cfg))
	defer srv.Close()

	stamp := time.Now().UnixNano()
	email := fmt.Sprintf("insp-%d@example.com", stamp)
	reg, _ := json.Marshal(map[string]string{"name": "Board Tester", "email": email, "password": "Original-Pass-123"})
	res, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(reg))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	token := loginAs(t, srv, email, "Original-Pass-123")

	call := func(method, path string, body any) (int, map[string]any) {
		t.Helper()
		var rd *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rd = bytes.NewReader(b)
		} else {
			rd = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, srv.URL+path, rd)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		out := map[string]any{}
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	reel := fmt.Sprintf("https://www.instagram.com/reels/T%d/?igsh=xyz", stamp)
	st, link := call(http.MethodPost, "/me/inspirations", map[string]string{"url": reel, "note": "Phuket trip"})
	if st != http.StatusAccepted || link["status"] != "pending" || link["platform"] != "instagram" {
		t.Fatalf("add link: %d %v", st, link)
	}
	if want := fmt.Sprintf("https://www.instagram.com/reel/T%d/", stamp); link["canonicalUrl"] != want {
		t.Errorf("canonicalUrl = %v, want %s", link["canonicalUrl"], want)
	}
	if st, again := call(http.MethodPost, "/me/inspirations", map[string]string{"url": reel}); st != http.StatusOK || again["id"] != link["id"] {
		t.Errorf("re-adding the same reel: %d %v", st, again)
	}
	if st, _ := call(http.MethodPost, "/me/inspirations", map[string]string{"url": "https://example.com/blog"}); st != http.StatusBadRequest {
		t.Errorf("unsupported link: status %d, want 400", st)
	}
	call(http.MethodPost, "/me/inspirations", map[string]string{"url": fmt.Sprintf("https://www.instagram.com/p/U%d/", stamp)})
	if st, _ := call(http.MethodPost, "/me/inspirations", map[string]string{"url": fmt.Sprintf("https://www.instagram.com/p/V%d/", stamp)}); st != http.StatusTooManyRequests {
		t.Errorf("third new link with a limit of 2: status %d, want 429", st)
	}
	if st, list := call(http.MethodGet, "/me/inspirations", nil); st != http.StatusOK || len(list["links"].([]any)) != 2 {
		t.Errorf("list links: %d %v", st, list)
	}

	gpID, err := store.UpsertGeoPlace(context.Background(), domain.GeoPlace{
		ID: fmt.Sprintf("gp_test%d", stamp), GooglePlaceID: fmt.Sprintf("gid_test%d", stamp),
		Name: "Big Buddha Phuket", Lat: 7.8276, Lng: 98.3129, Kind: "see", CountryCode: "TH", City: "Phuket",
	})
	if err != nil {
		t.Fatal(err)
	}
	st, saved := call(http.MethodPost, "/me/saved-places", map[string]string{"geoPlaceId": gpID, "sourceLinkId": link["id"].(string)})
	if st != http.StatusCreated || saved["geoPlace"] == nil || saved["sourceLinkId"] != link["id"] {
		t.Fatalf("save place: %d %v", st, saved)
	}
	if st, again := call(http.MethodPost, "/me/saved-places", map[string]string{"geoPlaceId": gpID}); st != http.StatusCreated || again["id"] != saved["id"] {
		t.Errorf("saving twice should return the same entry: %d %v", st, again)
	}
	if st, _ := call(http.MethodPost, "/me/saved-places", map[string]string{"geoPlaceId": gpID, "placeId": "x"}); st != http.StatusBadRequest {
		t.Errorf("both ids: status %d, want 400", st)
	}
	if st, list := call(http.MethodGet, "/me/saved-places", nil); st != http.StatusOK || len(list["places"].([]any)) != 1 {
		t.Errorf("list saved: %d %v", st, list)
	}
	if st, _ := call(http.MethodDelete, "/me/saved-places/"+saved["id"].(string), nil); st != http.StatusNoContent {
		t.Errorf("delete saved: status %d", st)
	}
	if st, _ := call(http.MethodDelete, "/me/inspirations/"+link["id"].(string), nil); st != http.StatusNoContent {
		t.Errorf("delete link: status %d", st)
	}
}
