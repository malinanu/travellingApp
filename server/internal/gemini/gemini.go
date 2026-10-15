// Package gemini extracts a Sri Lanka DRC Business Registration certificate
// number from an uploaded PDF/image using the configured Gemini chat model
// (Client.ChatModel) — a multimodal read,
// not a fixed OCR pipeline, so it needs no PDF rasterization step and holds
// up better against this certificate's decorative background than a
// regex-over-raw-OCR-text heuristic would. This is a pre-fill only: the
// vendor always sees/can edit the result before internal/eroc verifies it.
package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const generateEndpointFmt = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"

const prompt = `This is a Sri Lanka Registrar of Companies "Certificate of Incorporation".
Find the printed Company Number (format: 1-3 letters then 4-10 digits, e.g. PV00334782),
usually boxed near the top of the certificate. Respond with ONLY that number, no spaces,
uppercase, nothing else. If you cannot find one, respond with exactly: NONE`

var certNumberPattern = regexp.MustCompile(`^[A-Z]{1,3}[0-9]{4,10}$`)

// ExtractCertNumber sends the uploaded document (PDF or image, used as-is)
// to Gemini and returns its best-guess certificate number, or "" if none
// was found or the response didn't look like a valid number, plus the
// call's token usage when the API reported it. err is
// non-nil only for a pipeline failure (network, non-200, bad JSON) —
// callers apply their own bounded retry via a scheduler attempt counter.
func (c *Client) ExtractCertNumber(ctx context.Context, data []byte, contentType string) (string, *Usage, error) {
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
		return "", nil, err
	}

	res, err := post(ctx, fmt.Sprintf(generateEndpointFmt, c.ChatModel), body, c.apiKey)
	if err != nil {
		return "", nil, fmt.Errorf("gemini: request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		// A 429 is treated as a transient pipeline error by the scheduler.
		return "", nil, upstreamError("cert", res)
	}

	var raw struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return "", nil, fmt.Errorf("gemini: decode response: %w", err)
	}
	usage := &Usage{
		PromptTokens:     raw.UsageMetadata.PromptTokenCount,
		CompletionTokens: raw.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      raw.UsageMetadata.TotalTokenCount,
	}
	if len(raw.Candidates) == 0 {
		return "", usage, nil
	}
	var sb strings.Builder
	for _, part := range raw.Candidates[0].Content.Parts {
		sb.WriteString(part.Text)
	}

	text := strings.ToUpper(strings.TrimSpace(sb.String()))
	if text == "NONE" || !certNumberPattern.MatchString(text) {
		return "", usage, nil // a clean run that found nothing — not an error
	}
	return text, usage, nil
}
