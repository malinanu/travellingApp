package googleplaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchTextNoResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.SearchText(context.Background(), "Nowhere"); !errors.Is(err, ErrNoResult) {
		t.Errorf("err = %v, want ErrNoResult", err)
	}
}

func TestSearchTextHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"API key not valid"}}`, http.StatusForbidden)
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.SearchText(context.Background(), "Big Buddha"); err == nil || errors.Is(err, ErrNoResult) {
		t.Errorf("err = %v, want an HTTP error", err)
	}
}

func TestKind(t *testing.T) {
	cases := map[string]string{
		"thai_restaurant":   "eat",
		"cafe":              "eat",
		"night_club":        "drink",
		"resort_hotel":      "stay",
		"buddhist_temple":   "see",
		"beach":             "see",
		"clothing_store":    "shop",
		"diving_center":     "do",
		"accounting_office": "",
	}
	for primary, want := range cases {
		if got := Kind(primary, nil); got != want {
			t.Errorf("Kind(%q) = %q, want %q", primary, got, want)
		}
	}
	if got := Kind("", []string{"point_of_interest", "bar"}); got != "drink" {
		t.Errorf("Kind from types = %q", got)
	}
}

func TestSearchTextMany(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mask := r.Header.Get("X-Goog-FieldMask"); !strings.Contains(mask, "places.editorialSummary") {
			t.Errorf("field mask %q lacks editorialSummary", mask)
		}
		var body struct {
			PageSize int `json:"pageSize"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.PageSize != 8 {
			t.Errorf("pageSize = %d, want 8", body.PageSize)
		}
		_, _ = w.Write([]byte(`{"places":[
			{"id":"a","displayName":{"text":"Sigiriya Rock"},"location":{"latitude":7.957,"longitude":80.76},
			 "rating":4.7,"userRatingCount":30000,"primaryType":"historical_landmark","editorialSummary":{"text":"Ancient rock fortress."},
			 "photos":[{"name":"places/a/photos/p1"}]},
			{"id":"b","displayName":{"text":"Pidurangala Rock"},"location":{"latitude":7.966,"longitude":80.76}}]}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := c.SearchTextMany(context.Background(), "top sights in Sigiriya", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Sigiriya Rock" || got[0].Summary != "Ancient rock fortress." ||
		got[0].PhotoName != "places/a/photos/p1" || got[1].Name != "Pidurangala Rock" {
		t.Errorf("got %+v", got)
	}
}
