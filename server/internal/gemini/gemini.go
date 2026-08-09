// Package gemini extracts a Sri Lanka DRC Business Registration certificate
// number from an uploaded PDF/image using Gemini 2.5 Flash's free tier
// (https://ai.google.dev/gemini-api/docs/rate-limits) — a multimodal read,
// not a fixed OCR pipeline, so it needs no PDF rasterization step and holds
// up better against this certificate's decorative background than a
// regex-over-raw-OCR-text heuristic would. This is a pre-fill only: the
// vendor always sees/can edit the result before internal/eroc verifies it.
package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const endpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"

const prompt = `This is a Sri Lanka Registrar of Companies "Certificate of Incorporation".
Find the printed Company Number (format: 1-3 letters then 4-10 digits, e.g. PV00334782),
usually boxed near the top of the certificate. Respond with ONLY that number, no spaces,
uppercase, nothing else. If you cannot find one, respond with exactly: NONE`

var certNumberPattern = regexp.MustCompile(`^[A-Z]{1,3}[0-9]{4,10}$`)

// ExtractCertNumber sends the uploaded document (PDF or image, used as-is)
// to Gemini and returns its best-guess certificate number, or "" if none
// was found or the response didn't look like a valid number. err is
// non-nil only for a pipeline failure (network, non-200, bad JSON) —
// callers apply their own bounded retry via a scheduler attempt counter.
func ExtractCertNumber(ctx context.Context, apiKey string, data []byte, contentType string) (string, error) {
	// Bounded independently of the caller's context — a slow/hanging
	// Gemini response shouldn't hold the scheduler's sweep goroutine open
	// indefinitely.
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	reqBody := map[string]any{
		"contents": []any{map[string]any{
			"parts": []any{
				map[string]any{"inline_data": map[string]string{
					"mime_type": contentType,
					"data":      base64.StdEncoding.EncodeToString(data),
				}},
				map[string]any{"text": prompt},
			},
		}},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?key="+url.QueryEscape(apiKey), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		// Treated as a transient pipeline error by the scheduler — free
		// tier is 10 requests/minute, so a burst can legitimately 429.
		return "", fmt.Errorf("gemini: rate limited (429)")
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini: unexpected status %d", res.StatusCode)
	}

	var raw struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}
	if len(raw.Candidates) == 0 || len(raw.Candidates[0].Content.Parts) == 0 {
		return "", nil
	}

	text := strings.ToUpper(strings.TrimSpace(raw.Candidates[0].Content.Parts[0].Text))
	if text == "NONE" || !certNumberPattern.MatchString(text) {
		return "", nil // a clean run that found nothing — not an error
	}
	return text, nil
}
