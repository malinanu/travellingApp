// Package youtube reads a video's public metadata (description, channel,
// duration, thumbnail) through the YouTube Data API v3. The Inspiration
// Board uses it instead of yt-dlp, which YouTube blocks from server IPs.
package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://www.googleapis.com"

type Client struct {
	APIKey  string
	BaseURL string // tests point this at an httptest server
	HTTP    *http.Client
}

func New(apiKey string) *Client {
	return &Client{APIKey: apiKey, BaseURL: defaultBaseURL, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// Video is the metadata Spotto needs from one YouTube video.
type Video struct {
	Title       string
	Description string
	Channel     string
	Duration    time.Duration
	Thumbnail   string
	// PrivacyStatus is public, unlisted or private; Gemini only watches
	// public videos.
	PrivacyStatus string
	AgeRestricted bool
}

// ErrNotFound means the video doesn't exist or isn't public.
var ErrNotFound = errors.New("youtube: video not found")

// Video fetches one video's metadata by its 11-character ID.
func (c *Client) Video(ctx context.Context, id string) (Video, error) {
	q := url.Values{"part": {"snippet,contentDetails,status"}, "id": {id}, "key": {c.APIKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/youtube/v3/videos?"+q.Encode(), nil)
	if err != nil {
		return Video{}, err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		// url.Error repeats the request URL, which carries the API key.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return Video{}, fmt.Errorf("youtube: videos: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return Video{}, fmt.Errorf("youtube: videos: HTTP %d: %s", res.StatusCode, apiErrorText(msg))
	}
	var raw struct {
		Items []struct {
			Snippet struct {
				Title        string `json:"title"`
				Description  string `json:"description"`
				ChannelTitle string `json:"channelTitle"`
				Thumbnails   map[string]struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration      string `json:"duration"`
				ContentRating struct {
					YtRating string `json:"ytRating"`
				} `json:"contentRating"`
			} `json:"contentDetails"`
			Status struct {
				PrivacyStatus string `json:"privacyStatus"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return Video{}, fmt.Errorf("youtube: decode: %w", err)
	}
	if len(raw.Items) == 0 {
		return Video{}, ErrNotFound
	}
	it := raw.Items[0]
	v := Video{
		Title:       it.Snippet.Title,
		Description: it.Snippet.Description,
		Channel:     it.Snippet.ChannelTitle,
		Duration:    ParseDuration(it.ContentDetails.Duration),

		PrivacyStatus: it.Status.PrivacyStatus,
		AgeRestricted: it.ContentDetails.ContentRating.YtRating == "ytAgeRestricted",
	}
	for _, size := range []string{"maxres", "standard", "high", "medium", "default"} {
		if t, ok := it.Snippet.Thumbnails[size]; ok && t.URL != "" {
			v.Thumbnail = t.URL
			break
		}
	}
	return v, nil
}

// apiErrorText turns a Google API error body into one log line: the reason
// code (e.g. API_KEY_HTTP_REFERRER_BLOCKED) and message, instead of the
// whole indented JSON.
func apiErrorText(body []byte) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
			Details []struct {
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) != nil || e.Error.Message == "" {
		return strings.Join(strings.Fields(string(body)), " ")
	}
	for _, d := range e.Error.Details {
		if d.Reason != "" {
			return d.Reason + ": " + e.Error.Message
		}
	}
	return e.Error.Message
}

var durationRe = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

// ParseDuration reads the API's ISO-8601 duration ("PT1M5S"); 0 when it
// can't (live streams report "P0D").
func ParseDuration(s string) time.Duration {
	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	units := []time.Duration{24 * time.Hour, time.Hour, time.Minute, time.Second}
	var d time.Duration
	for i, u := range units {
		if n, err := strconv.Atoi(m[i+1]); err == nil {
			d += time.Duration(n) * u
		}
	}
	return d
}
