package googleplaces

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
