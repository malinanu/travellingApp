package reelextract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/youtube"
)

// YouTube is read without downloading: Gemini fetches public YouTube
// videos itself from the link, so even a 20-minute vlog never passes
// through this server. The title, description and chapters (vlogs often
// name each place in a chapter) come from the YouTube Data API, then
// yt-dlp, then oEmbed: YouTube blocks yt-dlp from server IPs, and oEmbed
// has no description or duration.

const (
	// Longest YouTube video Gemini watches; longer ones are read from their
	// description and chapters.
	maxYouTubeSecs = 20 * 60
	// Above this, frames are sampled at low detail to keep the token cost
	// of long videos down.
	lowResAfterSecs = maxVideoSecs
)

const (
	MsgYouTubeTooLong   = "This video is over 20 minutes, so places come from its description and chapters only."
	MsgYouTubeNotPublic = "This video isn't public, so places come from its description and chapters only."
)

func (x *Extractor) extractYouTube(ctx context.Context, link string) (Result, error) {
	meta, err := x.youtubeMetadata(ctx, link)
	if errors.Is(err, youtube.ErrNotFound) {
		return Result{}, &Error{Message: MsgCantOpen, Err: err}
	}
	if err != nil {
		if !isShort(link) {
			return Result{}, &Error{Message: MsgCantOpen, Retryable: true, Err: err}
		}
		// Gemini can still watch a Short by URL; only the caption, author
		// and thumbnail are lost. Shorts are short, so no length check.
		log.Printf("reelextract: no details for %s (%v); analysing the video alone", link, err)
		meta = ytdlpMeta{}
	}
	res := Result{Caption: youtubeText(meta), Author: meta.author(), ThumbnailURL: meta.Thumbnail, VideoSecs: meta.Duration}

	var video *gemini.Video
	switch {
	case meta.Availability != "" && meta.Availability != "public":
		res.PartialReason = MsgYouTubeNotPublic
	case meta.AgeLimit >= 18:
		res.PartialReason = MsgYouTubeNotPublic
	case meta.Duration > maxYouTubeSecs:
		res.PartialReason = MsgYouTubeTooLong
	default:
		video = &gemini.Video{
			FileURI: youtubeWatchURL(link, meta.ID),
			// Unknown length (oEmbed fallback) is treated as long.
			LowResolution: meta.Duration == 0 || meta.Duration > lowResAfterSecs,
		}
	}
	if video == nil && strings.TrimSpace(res.Caption) == "" {
		return res, &Error{Message: MsgNothingToRead}
	}
	if err := x.analyse(ctx, &res, video); err != nil {
		return res, err
	}
	return res, nil
}

// youtubeText is what Gemini reads besides the video: title, description
// and chapter list.
func youtubeText(m ytdlpMeta) string {
	var b strings.Builder
	if m.Title != "" {
		b.WriteString(m.Title + "\n\n")
	}
	if m.Description != "" {
		b.WriteString(m.Description + "\n\n")
	}
	if len(m.Chapters) > 0 {
		b.WriteString("Chapters:\n")
		for _, c := range m.Chapters {
			s := int(c.StartTime)
			fmt.Fprintf(&b, "%d:%02d %s\n", s/60, s%60, c.Title)
		}
	}
	return strings.TrimSpace(b.String())
}

// youtubeWatchURL is the standard watch link Gemini accepts (Shorts too).
func youtubeWatchURL(link, id string) string {
	if id == "" {
		if u, err := url.Parse(link); err == nil {
			id = u.Query().Get("v")
			if id == "" {
				parts := strings.Split(strings.Trim(u.Path, "/"), "/")
				id = parts[len(parts)-1]
			}
		}
	}
	return "https://www.youtube.com/watch?v=" + url.QueryEscape(id)
}

// youtubeOEmbedBase is YouTube's public oEmbed endpoint (tests point it elsewhere).
var youtubeOEmbedBase = "https://www.youtube.com/oembed"

func youtubeOEmbed(ctx context.Context, link string) (ytdlpMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, youtubeOEmbedBase+"?format=json&url="+url.QueryEscape(link), nil)
	if err != nil {
		return ytdlpMeta{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ytdlpMeta{}, fmt.Errorf("youtube oembed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return ytdlpMeta{}, fmt.Errorf("youtube oembed: HTTP %d", res.StatusCode)
	}
	var o struct {
		Title        string `json:"title"`
		AuthorName   string `json:"author_name"`
		ThumbnailURL string `json:"thumbnail_url"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&o); err != nil {
		return ytdlpMeta{}, fmt.Errorf("youtube oembed: decode: %w", err)
	}
	return ytdlpMeta{Title: o.Title, Channel: o.AuthorName, Thumbnail: o.ThumbnailURL}, nil
}

// youtubeMetadata tries the official API, then yt-dlp, then oEmbed.
func (x *Extractor) youtubeMetadata(ctx context.Context, link string) (ytdlpMeta, error) {
	var errs []error
	if id := youtubeID(link); x.YouTube != nil && id != "" {
		v, err := x.YouTube.Video(ctx, id)
		if err == nil {
			m := ytdlpMeta{
				ID: id, Title: v.Title, Description: v.Description, Channel: v.Channel,
				Duration: v.Duration.Seconds(), Thumbnail: v.Thumbnail, Availability: v.PrivacyStatus,
			}
			if v.AgeRestricted {
				m.AgeLimit = 18
			}
			return m, nil
		}
		if errors.Is(err, youtube.ErrNotFound) {
			return ytdlpMeta{}, err
		}
		errs = append(errs, err)
	}
	if _, err := exec.LookPath("yt-dlp"); err == nil {
		m, err := x.metadata(ctx, link)
		if err == nil {
			return m, nil
		}
		errs = append(errs, err)
	}
	m, err := youtubeOEmbed(ctx, link)
	if err == nil {
		return m, nil
	}
	return ytdlpMeta{}, errors.Join(append(errs, err)...)
}

// youtubeID is the video ID in a canonical YouTube link ("" if none).
func youtubeID(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	if v := u.Query().Get("v"); v != "" {
		return v
	}
	if parts := strings.Split(strings.Trim(u.Path, "/"), "/"); len(parts) == 2 && parts[0] == "shorts" {
		return parts[1]
	}
	return ""
}

// isShort reports whether link is a YouTube Shorts link.
func isShort(link string) bool {
	u, err := url.Parse(link)
	return err == nil && strings.HasPrefix(u.Path, "/shorts/")
}
