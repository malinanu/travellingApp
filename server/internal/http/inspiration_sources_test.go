package httpapi

import (
	"context"
	"testing"

	"github.com/spotto/server/internal/domain"
)

func TestYoutubeID(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/shorts/aBcDeFgHiJk":  "aBcDeFgHiJk",
		"https://www.youtube.com/watch?v=aBcDeFgHiJk": "aBcDeFgHiJk",
		"https://www.instagram.com/reel/DbQSwgwMM_n/": "",
	}
	for in, want := range cases {
		if got := youtubeID(in); got != want {
			t.Errorf("youtubeID(%q) = %q, want %q", in, got, want)
		}
	}
	// Every canonical YouTube link must yield an ID.
	for _, raw := range []string{"https://youtube.com/shorts/aBcDeFgHiJk?si=x", "https://youtu.be/aBcDeFgHiJk", "https://m.youtube.com/watch?v=aBcDeFgHiJk&t=3"} {
		canonical, _, err := canonicalLink(raw)
		if err != nil || youtubeID(canonical) != "aBcDeFgHiJk" {
			t.Errorf("%s -> %q (%v) -> %q", raw, canonical, err, youtubeID(canonical))
		}
	}
}

func TestLinkVideoYouTubeUsesURL(t *testing.T) {
	s := &Server{}
	v := s.linkVideo(context.Background(), domain.LinkExtraction{
		CanonicalURL: "https://www.youtube.com/shorts/aBcDeFgHiJk", Platform: "youtube",
	})
	if v == nil || v.URI != "https://www.youtube.com/watch?v=aBcDeFgHiJk" || v.Data != nil {
		t.Errorf("video = %+v", v)
	}
}
