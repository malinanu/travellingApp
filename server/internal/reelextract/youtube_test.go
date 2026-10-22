package reelextract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spotto/server/internal/gemini"
)

func TestYouTubeText(t *testing.T) {
	m := ytdlpMeta{Title: "Top 10 Places in Phuket", Description: "Our favourite spots"}
	m.Chapters = append(m.Chapters,
		struct {
			StartTime float64 `json:"start_time"`
			Title     string  `json:"title"`
		}{0, "Intro"},
		struct {
			StartTime float64 `json:"start_time"`
			Title     string  `json:"title"`
		}{95, "Big Buddha"},
	)
	got := youtubeText(m)
	for _, want := range []string{"Top 10 Places in Phuket", "Our favourite spots", "Chapters:", "0:00 Intro", "1:35 Big Buddha"} {
		if !strings.Contains(got, want) {
			t.Errorf("youtubeText missing %q in:\n%s", want, got)
		}
	}
}

func TestYouTubeWatchURL(t *testing.T) {
	cases := map[[2]string]string{
		{"https://www.youtube.com/shorts/_T2xmk-u5pw", ""}:         "https://www.youtube.com/watch?v=_T2xmk-u5pw",
		{"https://www.youtube.com/watch?v=IpjDQmY7myo", ""}:        "https://www.youtube.com/watch?v=IpjDQmY7myo",
		{"https://www.youtube.com/shorts/whatever", "Tw_IDlFJRWY"}: "https://www.youtube.com/watch?v=Tw_IDlFJRWY",
	}
	for in, want := range cases {
		if got := youtubeWatchURL(in[0], in[1]); got != want {
			t.Errorf("youtubeWatchURL(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestYouTubeOEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"title":"7 Days in Phuket","author_name":"Trip Stamp","thumbnail_url":"https://i.ytimg.com/vi/x/hq.jpg"}`))
	}))
	defer srv.Close()
	old := youtubeOEmbedBase
	youtubeOEmbedBase = srv.URL
	defer func() { youtubeOEmbedBase = old }()
	m, err := youtubeOEmbed(context.Background(), "https://www.youtube.com/watch?v=x")
	if err != nil || m.Title != "7 Days in Phuket" || m.author() != "Trip Stamp" || m.Duration != 0 {
		t.Errorf("oembed = %+v, %v", m, err)
	}
}

func TestDropImpossibleTimes(t *testing.T) {
	at := func(s int) *int { return &s }
	places := []gemini.VideoPlace{{Name: "A", AtSec: at(120)}, {Name: "B", AtSec: at(856)}, {Name: "C"}}
	dropImpossibleTimes(places, 569)
	if places[0].AtSec == nil || *places[0].AtSec != 120 || places[1].AtSec != nil || places[2].AtSec != nil {
		t.Errorf("times = %v %v %v", places[0].AtSec, places[1].AtSec, places[2].AtSec)
	}
	dropImpossibleTimes(places[:1], 0) // unknown length: keep
	if places[0].AtSec == nil {
		t.Error("kept time dropped when the length is unknown")
	}
}
