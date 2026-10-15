package gemini

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Default model ids, overridable via GEMINI_CHAT_MODEL / GEMINI_EMBED_MODEL
// (see internal/config). Google retires model ids on its own schedule —
// gemini-1.5-flash and text-embedding-004 both started 404ing, which is what
// silently broke Local Guide — so these live in config rather than code.
//
// The chat default is a "lite" model on purpose: gemini-3.8-flash thinks
// before every answer (10s+ per round, several rounds per guide message)
// and its free tier allows only 20 requests a day. The lite model answers
// a tool-calling round in about a second and handles the guide's tools fine.
const (
	DefaultChatModel  = "gemini-3.5-flash-lite"
	DefaultEmbedModel = "gemini-embedding-001"
)

// EmbeddingDimensions is requested explicitly (gemini-embedding-001 defaults
// to 3072) so each place's JSONB vector stays small.
const EmbeddingDimensions = 768

// Client carries the API key and model ids for every Gemini call the server
// makes. It holds no usage state — callers in internal/http time and record
// each call themselves (see ai_usage.go).
type Client struct {
	APIKey     string
	ChatModel  string
	EmbedModel string
}

func NewClient(apiKey, chatModel, embedModel string) *Client {
	if chatModel == "" {
		chatModel = DefaultChatModel
	}
	if embedModel == "" {
		embedModel = DefaultEmbedModel
	}
	return &Client{APIKey: apiKey, ChatModel: chatModel, EmbedModel: embedModel}
}

// ErrRateLimited distinguishes a 429 so callers can surface something
// friendlier than a generic failure.
var ErrRateLimited = errors.New("gemini: rate limited (429)")

// Usage is token accounting for one call. Embedding calls get no token
// counts back from the API, so Estimated marks a characters/4 estimate.
type Usage struct {
	PromptTokens     int  `json:"prompt_tokens"`
	CompletionTokens int  `json:"completion_tokens"`
	TotalTokens      int  `json:"total_tokens"`
	Estimated        bool `json:"-"`
}

// ErrDailyQuota marks a 429 caused by a per-day quota (the free tier's
// daily request cap) — retrying is pointless until Google's Pacific-midnight
// reset. It is always wrapped together with ErrRateLimited.
var ErrDailyQuota = errors.New("daily quota used up")

var (
	quotaRe   = regexp.MustCompile(`Quota exceeded for metric: ([^,]+), limit: (\d+), model: ([\w.\-]+)`)
	retryRe   = regexp.MustCompile(`Please retry in ([\dhm]*\d+)(?:\.\d+)?s`)
	messageRe = regexp.MustCompile(`"message":\s*"((?:[^"\\]|\\.)*)"`)
	quotaIDRe = regexp.MustCompile(`"quotaId":\s*"([^"]+)"`)
)

// upstreamError turns a non-200 response into an error that carries the
// status and Google's own reason (e.g. "models/x is not found"), so the
// admin dashboard's error list shows the real cause instead of a bare status
// code. A 429's body opens with a long boilerplate paragraph, so the quota
// metric, limit, model and retry delay are pulled out and put first rather
// than truncated away. A 429 wraps ErrRateLimited (and ErrDailyQuota when
// the exhausted quota is a daily one).
func upstreamError(kind string, res *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	body := strings.Join(strings.Fields(string(b)), " ")

	var detail string
	if m := quotaRe.FindStringSubmatch(body); m != nil {
		detail = fmt.Sprintf("quota %s limit %s model %s", m[1], m[2], m[3])
		if r := retryRe.FindStringSubmatch(body); r != nil {
			detail += "; retry in " + r[1] + "s"
		}
	} else if m := messageRe.FindStringSubmatch(body); m != nil {
		detail = m[1]
	} else {
		detail = body
	}
	if len(detail) > 300 {
		detail = detail[:300] + "…"
	}

	if res.StatusCode == http.StatusTooManyRequests {
		if isDailyQuota(body) {
			return fmt.Errorf("%w (%w): %s: %s", ErrRateLimited, ErrDailyQuota, kind, detail)
		}
		return fmt.Errorf("%w: %s: %s", ErrRateLimited, kind, detail)
	}
	return fmt.Errorf("gemini %s %d: %s", kind, res.StatusCode, detail)
}

// isDailyQuota reports whether a 429 was caused by a per-day quota. The
// metric name alone can't tell (the free tier's per-minute and per-day caps
// share generate_content_free_tier_requests), so this reads the violation's
// quotaId, e.g. GenerateRequestsPerDayPerProjectPerModel-FreeTier.
func isDailyQuota(body string) bool {
	for _, m := range quotaIDRe.FindAllStringSubmatch(body, -1) {
		if strings.Contains(m[1], "PerDay") {
			return true
		}
	}
	return false
}

// retryBackoff is the wait before each retry of a 500/503 — Gemini answers
// "model is currently experiencing high demand" with a 503 that usually
// clears within a second or two, so a short retry turns most of those into
// a successful reply instead of a traveller-facing error.
var retryBackoff = []time.Duration{1 * time.Second, 2500 * time.Millisecond, 5 * time.Second}

// post sends body to url, retrying transient 500/503s per retryBackoff.
// The caller owns closing the returned response's body.
func post(ctx context.Context, url string, body []byte, setAuth func(*http.Request)) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		setAuth(req)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		transient := res.StatusCode == http.StatusServiceUnavailable || res.StatusCode == http.StatusInternalServerError
		if !transient || attempt >= len(retryBackoff) {
			return res, nil
		}
		res.Body.Close()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryBackoff[attempt]):
		}
	}
}

func (c *Client) bearer(req *http.Request) { req.Header.Set("Authorization", "Bearer "+c.APIKey) }
func (c *Client) apiKey(req *http.Request) { req.Header.Set("x-goog-api-key", c.APIKey) }
