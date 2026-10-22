package youtube

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"PT59S":      59 * time.Second,
		"PT1M5S":     65 * time.Second,
		"PT2H":       2 * time.Hour,
		"P1DT1H1M1S": 25*time.Hour + time.Minute + time.Second,
		"P0D":        0,
		"":           0,
		"garbage":    0,
	}
	for in, want := range cases {
		if got := ParseDuration(in); got != want {
			t.Errorf("ParseDuration(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestVideo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "aBcDeFgHiJk" || r.URL.Query().Get("key") != "k" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"items":[{"snippet":{"title":"Phuket eats","description":"1. Kata Noodle House","channelTitle":"phuket.eats",
			"thumbnails":{"default":{"url":"d.jpg"},"high":{"url":"h.jpg"}}},"contentDetails":{"duration":"PT45S"}}]}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client()}
	v, err := c.Video(context.Background(), "aBcDeFgHiJk")
	if err != nil {
		t.Fatal(err)
	}
	if v.Description != "1. Kata Noodle House" || v.Channel != "phuket.eats" || v.Duration != 45*time.Second || v.Thumbnail != "h.jpg" {
		t.Errorf("video = %+v", v)
	}
}

func TestVideoNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.Video(context.Background(), "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestVideoHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"YouTube Data API v3 has not been used"}}`, http.StatusForbidden)
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Video(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("err = %v, want HTTP 403", err)
	}
}
