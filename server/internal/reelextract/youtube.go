package reelextract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spotto/server/internal/gemini"
)

// YouTube is read without downloading: Gemini fetches public YouTube
// videos itself from the link, so even a 20-minute vlog never passes
// through this server. yt-dlp is only asked for the title, description
// and chapters (vlogs often name each place in a chapter).

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
	meta, err := x.metadata(ctx, link)
	if err != nil {
		// YouTube often blocks yt-dlp on server IPs. Its public oEmbed still
		// gives the title and channel, and Gemini can watch the video anyway.
		oe, oerr := youtubeOEmbed(ctx, link)
		if oerr != nil {
			return Result{}, &Error{Message: MsgCantOpen, Retryable: true, Err: err}
		}
		meta = oe
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
