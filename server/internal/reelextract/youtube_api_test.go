package reelextract

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spotto/server/internal/youtube"
)

func TestYouTubeIDAndIsShort(t *testing.T) {
	cases := []struct {
		link  string
		id    string
		short bool
	}{
		{"https://www.youtube.com/shorts/aBcDeFgHiJk", "aBcDeFgHiJk", true},
		{"https://www.youtube.com/watch?v=aBcDeFgHiJk", "aBcDeFgHiJk", false},
		{"https://www.youtube.com/", "", false},
	}
	for _, c := range cases {
		if got := youtubeID(c.link); got != c.id {
			t.Errorf("youtubeID(%s) = %q, want %q", c.link, got, c.id)
		}
		if got := isShort(c.link); got != c.short {
			t.Errorf("isShort(%s) = %v, want %v", c.link, got, c.short)
		}
	}
}

func TestYouTubeMetadataFromAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"snippet":{"title":"Kata eats","description":"📍 Kata Noodle House","channelTitle":"phuket.eats"},
			"contentDetails":{"duration":"PT50S","contentRating":{"ytRating":"ytAgeRestricted"}},"status":{"privacyStatus":"public"}}]}`))
	}))
	defer srv.Close()
	x := &Extractor{YouTube: &youtube.Client{APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client()}}
	m, err := x.youtubeMetadata(context.Background(), "https://www.youtube.com/shorts/aBcDeFgHiJk")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "aBcDeFgHiJk" || m.Description != "📍 Kata Noodle House" || m.Duration != 50 || m.Availability != "public" || m.AgeLimit != 18 {
		t.Errorf("meta = %+v", m)
	}
}

func TestYouTubeMetadataAllSourcesFail(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"Requests from referer <empty> are blocked.","details":[{"reason":"API_KEY_HTTP_REFERRER_BLOCKED"}]}}`, http.StatusForbidden)
	}))
	defer api.Close()
	oembed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer oembed.Close()
	old := youtubeOEmbedBase
	youtubeOEmbedBase = oembed.URL
	defer func() { youtubeOEmbedBase = old }()

	x := &Extractor{YouTube: &youtube.Client{APIKey: "k", BaseURL: api.URL, HTTP: api.Client()}}
	_, err := x.youtubeMetadata(context.Background(), "https://www.youtube.com/shorts/aBcDeFgHiJk")
	if err == nil || !strings.Contains(err.Error(), "API_KEY_HTTP_REFERRER_BLOCKED") || !strings.Contains(err.Error(), "oembed") {
		t.Errorf("err = %v, want both the API and oEmbed failures", err)
	}
}

func TestYouTubeMetadataNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()
	x := &Extractor{YouTube: &youtube.Client{APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client()}}
	if _, err := x.youtubeMetadata(context.Background(), "https://www.youtube.com/watch?v=gone"); !errors.Is(err, youtube.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
