package httpapi

// Where the Inspiration Board worker reads a link from. Each platform uses
// its official API when there is one, because yt-dlp gets blocked from
// server IPs ("sign in to confirm you're not a bot", Instagram's login
// redirect):
//   - YouTube: YouTube Data API v3 for the details, and Gemini watches the
//     public video by URL, so nothing is downloaded.
//   - TikTok and Instagram: no usable official API, so yt-dlp (with
//     YTDLP_COOKIES_FILE). TikTok's oEmbed no longer returns the caption,
//     author or thumbnail (checked Oct 2026), only an embed player.
// Official sources fall back to yt-dlp when they fail.

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os/exec"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/youtube"
)

// errNoYtdlp means the link needs yt-dlp and it isn't installed.
var errNoYtdlp = errors.New("yt-dlp not installed")

func hasYtdlp() bool {
	_, err := exec.LookPath("yt-dlp")
	return err == nil
}

// linkMetadata reads the caption, author, duration and thumbnail.
func (s *Server) linkMetadata(ctx context.Context, job domain.LinkExtraction) (ytdlpMeta, error) {
	switch job.Platform {
	case "youtube":
		if id := youtubeID(job.CanonicalURL); s.YouTube != nil && id != "" {
			v, err := s.YouTube.Video(ctx, id)
			if err == nil {
				return ytdlpMeta{
					Title: v.Title, Description: v.Description, Channel: v.Channel,
					Duration: v.Duration.Seconds(), Thumbnail: v.Thumbnail,
				}, nil
			}
			if errors.Is(err, youtube.ErrNotFound) {
				return ytdlpMeta{}, err
			}
			log.Printf("inspiration worker: youtube api %s: %v (trying yt-dlp)", job.CanonicalURL, err)
		}
	}
	if !hasYtdlp() {
		return ytdlpMeta{}, errNoYtdlp
	}
	return s.ytdlpMetadata(ctx, job.CanonicalURL)
}

// linkVideo gets the video for Gemini: a URL Gemini fetches itself for
// YouTube, otherwise a yt-dlp download. nil means caption only.
func (s *Server) linkVideo(ctx context.Context, job domain.LinkExtraction) *gemini.Video {
	if id := youtubeID(job.CanonicalURL); job.Platform == "youtube" && id != "" {
		return &gemini.Video{URI: "https://www.youtube.com/watch?v=" + id}
	}
	if !hasYtdlp() {
		return nil
	}
	video, err := s.ytdlpDownload(ctx, job.CanonicalURL)
	if err != nil {
		log.Printf("inspiration worker: download %s: %v", job.CanonicalURL, err)
		return nil
	}
	return video
}

// youtubeID pulls the video ID out of a canonicalLink YouTube URL.
func youtubeID(canonical string) string {
	u, err := url.Parse(canonical)
	if err != nil {
		return ""
	}
	if m := ytShortsRe.FindStringSubmatch(u.Path); m != nil {
		return m[1]
	}
	return u.Query().Get("v")
}
