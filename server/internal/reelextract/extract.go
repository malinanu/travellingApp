// Package reelextract turns a travel video link (Instagram, TikTok,
// YouTube, …) into the list of places it shows: yt-dlp fetches the post and
// its video, and Gemini watches it and reads the caption.
//
// It has no dependency on the HTTP API or the database, so it can run in
// its own service on its own server: the API's worker only needs something
// with an Extract method (the Interface below). See
// internal/http/inspiration_worker.go for the queue, place lookup and
// caching around it.
package reelextract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spotto/server/internal/gemini"
)

// Interface is what the API's worker needs; satisfied by *Extractor here,
// and later by an HTTP client for a separate extraction service.
type Interface interface {
	Extract(ctx context.Context, canonicalURL, platform string) (Result, error)
}

// Result of one extraction.
type Result struct {
	Caption      string
	Author       string
	ThumbnailURL string
	Model        string
	RegionHint   string
	Places       []gemini.VideoPlace

	// PartialReason is set when the video itself couldn't be watched and
	// places come from the caption only; it is shown to the traveller.
	PartialReason string

	// What the job cost, recorded so moving extraction to its own server
	// is a measured decision.
	VideoBytes int64
	VideoSecs  float64
}

// Error is a failure with a message fit to show the traveller, and whether
// trying again later might work.
type Error struct {
	Message   string
	Retryable bool
	Err       error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// Messages travellers see on a link card.
const (
	MsgNotConfigured       = "Reading reels isn't set up on this server yet."
	MsgCantOpen            = "We couldn't open this link. It may be private, deleted, or blocked from downloading."
	MsgNothingToRead       = "We couldn't watch this video and it has no caption to read."
	MsgCaptionOnly         = "We couldn't watch the video, so these places come from the caption only."
	MsgPhotoPost           = "This is a photo post, so these places come from its caption only."
	MsgPhotoPostUnreadable = "We can't read TikTok photo posts yet. Try a video post instead."
	MsgAnalysisFail        = "We couldn't analyse this video right now. Tap Retry to try again."
)

const (
	// Videos longer than this are read from the caption only: a long video
	// costs far more to analyse and rarely is a "N places to visit" reel.
	maxVideoSecs = 180
	maxVideoMB   = 45
)

// Extractor runs yt-dlp and Gemini in this process.
type Extractor struct {
	Gemini      *gemini.Client
	VideoModel  string // "" = Gemini.ChatModel
	CookiesFile string // optional cookies.txt for yt-dlp

	// OnAICall, when set, is told about every Gemini call (for usage
	// tracking): kind is "video" or "chat".
	OnAICall func(ctx context.Context, kind, model string, usage *gemini.Usage, start time.Time, err error)
}

func (x *Extractor) model() string {
	if x.VideoModel != "" {
		return x.VideoModel
	}
	return x.Gemini.ChatModel
}

// Extract fetches the post, watches the video when it can, and returns the
// places in it. Errors are *Error.
func (x *Extractor) Extract(ctx context.Context, link, platform string) (Result, error) {
	if x.Gemini == nil || x.Gemini.APIKey == "" {
		return Result{}, &Error{Message: MsgNotConfigured}
	}
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return Result{}, &Error{Message: MsgNotConfigured, Err: errors.New("yt-dlp not installed")}
	}

	if platform == "youtube" {
		return x.extractYouTube(ctx, link)
	}

	// TikTok photo carousels have no video; yt-dlp may still read their caption.
	photoPost := platform == "tiktok" && strings.Contains(link, "/photo/")
	meta, err := x.metadata(ctx, link)
	if err != nil {
		msg := MsgCantOpen
		if photoPost {
			msg = MsgPhotoPostUnreadable
		}
		return Result{}, &Error{Message: msg, Retryable: !photoPost, Err: err}
	}
	res := Result{Caption: meta.caption(), Author: meta.author(), ThumbnailURL: meta.Thumbnail, VideoSecs: meta.Duration}

	var video *gemini.Video
	var downloadErr error
	if !photoPost && meta.Duration <= maxVideoSecs {
		video, downloadErr = x.download(ctx, link)
		if video != nil {
			res.VideoBytes = int64(len(video.Data))
		}
	}
	if video == nil && strings.TrimSpace(res.Caption) == "" {
		return res, &Error{Message: MsgNothingToRead, Err: downloadErr}
	}

	if err := x.analyse(ctx, &res, video); err != nil {
		return res, err
	}
	if video == nil {
		res.PartialReason = MsgCaptionOnly
		if photoPost {
			res.PartialReason = MsgPhotoPost
		}
	}
	return res, nil
}

// analyse asks Gemini for the places and fills them into res.
func (x *Extractor) analyse(ctx context.Context, res *Result, video *gemini.Video) error {
	res.Model = x.model()
	start := time.Now()
	found, usage, err := x.Gemini.ExtractPlacesFromVideo(ctx, res.Model, video, res.Caption)
	if x.OnAICall != nil {
		kind := "video"
		if video == nil {
			kind = "chat"
		}
		x.OnAICall(ctx, kind, res.Model, usage, start, err)
	}
	if err != nil {
		return &Error{Message: MsgAnalysisFail, Retryable: true, Err: err}
	}
	res.RegionHint, res.Places = found.RegionHint, found.Places
	dropImpossibleTimes(res.Places, res.VideoSecs)
	return nil
}

// dropImpossibleTimes clears timestamps past the end of the video: at low
// frame detail Gemini sometimes misplaces them, and a card saying "at
// 14:16" in a 9-minute video is worse than none.
func dropImpossibleTimes(places []gemini.VideoPlace, videoSecs float64) {
	if videoSecs <= 0 {
		return
	}
	for i := range places {
		if places[i].AtSec != nil && float64(*places[i].AtSec) > videoSecs {
			places[i].AtSec = nil
		}
	}
}

// ---- yt-dlp ----

type ytdlpMeta struct {
	Description string  `json:"description"`
	Title       string  `json:"title"`
	Uploader    string  `json:"uploader"`
	UploaderID  string  `json:"uploader_id"`
	Channel     string  `json:"channel"`
	Duration    float64 `json:"duration"`
	Thumbnail   string  `json:"thumbnail"`

	// YouTube
	ID           string `json:"id"`
	Availability string `json:"availability"` // public | unlisted | private | needs_auth | …
	AgeLimit     int    `json:"age_limit"`
	Chapters     []struct {
		StartTime float64 `json:"start_time"`
		Title     string  `json:"title"`
	} `json:"chapters"`
}

func (m ytdlpMeta) caption() string {
	if m.Description != "" {
		return m.Description
	}
	return m.Title
}

func (m ytdlpMeta) author() string {
	switch {
	case m.Channel != "":
		return m.Channel
	case m.Uploader != "":
		return m.Uploader
	default:
		return m.UploaderID
	}
}

func (x *Extractor) ytdlpArgs(extra ...string) []string {
	args := []string{"--no-warnings", "--no-playlist", "--socket-timeout", "20"}
	if x.CookiesFile != "" {
		args = append(args, "--cookies", x.CookiesFile)
	}
	return append(args, extra...)
}

func (x *Extractor) metadata(ctx context.Context, link string) (ytdlpMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "yt-dlp", x.ytdlpArgs("--dump-json", "--skip-download", "--", link)...).Output()
	if err != nil {
		return ytdlpMeta{}, fmt.Errorf("yt-dlp metadata: %w%s", err, stderrOf(err))
	}
	var m ytdlpMeta
	if err := json.Unmarshal(out, &m); err != nil {
		return ytdlpMeta{}, fmt.Errorf("yt-dlp metadata: parse: %w", err)
	}
	return m, nil
}

// download fetches the video as a single file (no ffmpeg merge needed).
func (x *Extractor) download(ctx context.Context, link string) (*gemini.Video, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	dir, err := os.MkdirTemp("", "spotto-reel-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	limit := fmt.Sprintf("%dM", maxVideoMB)
	format := fmt.Sprintf("best[ext=mp4][filesize<%[1]s]/best[ext=mp4][filesize_approx<%[1]s]/best[ext=mp4]/best", limit)
	args := x.ytdlpArgs("-f", format, "--max-filesize", limit, "-o", filepath.Join(dir, "video.%(ext)s"), "--", link)
	if out, err := exec.CommandContext(ctx, "yt-dlp", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("yt-dlp download: %w: %s", err, lastLine(string(out)))
	}
	files, _ := filepath.Glob(filepath.Join(dir, "video.*"))
	if len(files) == 0 {
		return nil, errors.New("yt-dlp download: no file (over the size limit?)")
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		return nil, err
	}
	mime := "video/mp4"
	switch strings.ToLower(filepath.Ext(files[0])) {
	case ".webm":
		mime = "video/webm"
	case ".mov":
		mime = "video/quicktime"
	}
	return &gemini.Video{Data: data, MimeType: mime}, nil
}

func stderrOf(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(ee.Stderr) > 0 {
		return ": " + lastLine(string(ee.Stderr))
	}
	return ""
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
