package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Place extraction from a travel reel: Gemini watches the video (on-screen
// text and what's said) and reads the caption, and returns the real places
// it shows as structured JSON.

// VideoPlace is one place Gemini found.
type VideoPlace struct {
	Name       string  `json:"name"`
	Kind       string  `json:"kind"`
	Area       string  `json:"area"`
	City       string  `json:"city"`
	Country    string  `json:"country"`
	Evidence   string  `json:"evidence"`
	AtSec      *int    `json:"atSec"`
	Confidence float64 `json:"confidence"`
}

// VideoPlaces is Gemini's answer for one reel.
type VideoPlaces struct {
	RegionHint string       `json:"regionHint"`
	Places     []VideoPlace `json:"places"`
}

// MaxVideoPlaces caps how many places one reel can return.
const MaxVideoPlaces = 50

// inlineVideoLimit keeps a base64-encoded video under the 20 MB request cap;
// anything bigger goes through the Files API.
const inlineVideoLimit = 14 << 20

const videoPrompt = `You are given a short travel video (a reel) and its caption. List every real,
specific, named place the video recommends or shows: restaurants, cafes, street-food stalls,
bars, hotels, beaches, viewpoints, temples, markets, attractions, activities.

Use all three sources: text shown on screen, what is said, and the caption.
Captions often list many more places than the video shows ("50 best things to do in …"):
include every named place from the caption too, even if it never appears in the video.
- Only real places with a proper name. Skip generic things like "a beach" or "street food".
- Never invent a place. If a name is only partly readable, give your best reading and lower the confidence.
- area is the neighbourhood or beach (e.g. "Kata Beach"); city and country are where the place is.
- kind is one of eat, drink, stay, see, do, shop.
- evidence is where you found it: on_screen, speech or caption.
- atSec is the second in the video where the place first appears (omit for caption-only).
- confidence is 0 to 1.
- regionHint is the main city/region the reel is about, e.g. "Phuket, Thailand".
List places in the order the video shows them, then the caption-only ones, at most 50.`

// placesSchema is the structured-output schema for VideoPlaces.
var placesSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"regionHint": map[string]any{"type": "STRING"},
		"places": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"name":       map[string]any{"type": "STRING"},
					"kind":       map[string]any{"type": "STRING", "enum": []string{"eat", "drink", "stay", "see", "do", "shop"}},
					"area":       map[string]any{"type": "STRING"},
					"city":       map[string]any{"type": "STRING"},
					"country":    map[string]any{"type": "STRING"},
					"evidence":   map[string]any{"type": "STRING", "enum": []string{"on_screen", "speech", "caption"}},
					"atSec":      map[string]any{"type": "INTEGER"},
					"confidence": map[string]any{"type": "NUMBER"},
				},
				"required": []string{"name", "kind", "evidence", "confidence"},
			},
		},
	},
	"required": []string{"places"},
}

// Video is a reel to analyse: downloaded bytes, or a URL Gemini fetches
// itself (public YouTube videos).
type Video struct {
	Data     []byte
	MimeType string

	// FileURI, when set, is passed to Gemini instead of Data.
	FileURI string
	// LowResolution samples frames at lower detail: far fewer tokens for
	// long videos, while large on-screen text stays readable.
	LowResolution bool
}

// ExtractPlacesFromVideo asks model (ChatModel when "") which places a reel
// shows. video may be nil to work from the caption alone.
func (c *Client) ExtractPlacesFromVideo(ctx context.Context, model string, video *Video, caption string) (VideoPlaces, *Usage, error) {
	if model == "" {
		model = c.ChatModel
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()

	var parts []any
	if video != nil && video.FileURI != "" {
		parts = append(parts, fileURIPart(video.FileURI))
	} else if video != nil {
		if len(video.Data) <= inlineVideoLimit {
			parts = append(parts, map[string]any{"inline_data": map[string]string{
				"mime_type": video.MimeType,
				"data":      base64.StdEncoding.EncodeToString(video.Data),
			}})
		} else {
			f, err := c.uploadFile(ctx, video.Data, video.MimeType)
			if err != nil {
				return VideoPlaces{}, nil, err
			}
			defer c.deleteFile(context.WithoutCancel(ctx), f.Name)
			parts = append(parts, map[string]any{"file_data": map[string]string{
				"mime_type": video.MimeType,
				"file_uri":  f.URI,
			}})
		}
	}
	text := videoPrompt
	if video == nil {
		text += "\n\nThe video itself is not available; use only the caption."
	}
	if strings.TrimSpace(caption) != "" {
		text += "\n\nCaption:\n" + caption
	}
	parts = append(parts, map[string]any{"text": text})

	body, err := json.Marshal(map[string]any{
		"contents":         []any{map[string]any{"role": "user", "parts": parts}},
		"generationConfig": videoGenerationConfig(video),
	})
	if err != nil {
		return VideoPlaces{}, nil, err
	}
	res, err := post(ctx, fmt.Sprintf(generateEndpointFmt, model), body, c.apiKey)
	if err != nil {
		return VideoPlaces{}, nil, fmt.Errorf("gemini: request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return VideoPlaces{}, nil, upstreamError("video", res)
	}
	text, usage, err := decodeGenerateText(res.Body)
	if err != nil {
		return VideoPlaces{}, usage, err
	}
	places, err := ParseVideoPlaces(text)
	return places, usage, err
}

// fileURIPart points Gemini at a video it fetches itself (e.g. YouTube).
func fileURIPart(uri string) map[string]any {
	return map[string]any{"file_data": map[string]string{"file_uri": uri}}
}

func videoGenerationConfig(video *Video) map[string]any {
	c := map[string]any{
		"responseMimeType": "application/json",
		"responseSchema":   placesSchema,
		"temperature":      0.2,
	}
	if video != nil && video.LowResolution {
		c["mediaResolution"] = "MEDIA_RESOLUTION_LOW"
	}
	return c
}

// decodeGenerateText reads a generateContent response's text and usage.
func decodeGenerateText(r io.Reader) (string, *Usage, error) {
	var raw struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return "", nil, fmt.Errorf("gemini: decode response: %w", err)
	}
	usage := &Usage{
		PromptTokens:     raw.UsageMetadata.PromptTokenCount,
		CompletionTokens: raw.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      raw.UsageMetadata.TotalTokenCount,
	}
	if raw.PromptFeedback.BlockReason != "" {
		return "", usage, fmt.Errorf("gemini: blocked: %s", raw.PromptFeedback.BlockReason)
	}
	if len(raw.Candidates) == 0 {
		return "", usage, errors.New("gemini: no candidates in response")
	}
	var sb strings.Builder
	for _, p := range raw.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	return sb.String(), usage, nil
}

// ParseVideoPlaces parses Gemini's JSON answer, dropping nameless entries,
// normalising kinds and capping the count.
func ParseVideoPlaces(text string) (VideoPlaces, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	var out VideoPlaces
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &out); err != nil {
		return VideoPlaces{}, fmt.Errorf("gemini: parse places: %w", err)
	}
	kept := make([]VideoPlace, 0, len(out.Places))
	seen := map[string]bool{}
	for _, p := range out.Places {
		p.Name = strings.TrimSpace(p.Name)
		key := strings.ToLower(p.Name)
		if p.Name == "" || seen[key] {
			continue
		}
		seen[key] = true
		switch p.Kind {
		case "eat", "drink", "stay", "see", "do", "shop":
		default:
			p.Kind = "see"
		}
		switch p.Evidence {
		case "on_screen", "speech", "caption":
		default:
			p.Evidence = "caption"
		}
		if p.Confidence < 0 {
			p.Confidence = 0
		} else if p.Confidence > 1 {
			p.Confidence = 1
		}
		if p.AtSec != nil && *p.AtSec < 0 {
			p.AtSec = nil
		}
		kept = append(kept, p)
		if len(kept) == MaxVideoPlaces {
			break
		}
	}
	out.Places = kept
	return out, nil
}

type uploadedFile struct {
	Name  string `json:"name"`
	URI   string `json:"uri"`
	State string `json:"state"`
}

const filesBaseURL = "https://generativelanguage.googleapis.com"

// uploadFile sends a large video through the Files API (resumable upload)
// and waits until Gemini has finished processing it.
func (c *Client) uploadFile(ctx context.Context, data []byte, mimeType string) (uploadedFile, error) {
	start, _ := json.Marshal(map[string]any{"file": map[string]string{"display_name": "spotto-reel"}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, filesBaseURL+"/upload/v1beta/files", bytes.NewReader(start))
	if err != nil {
		return uploadedFile{}, err
	}
	c.apiKey(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Upload-Protocol", "resumable")
	req.Header.Set("X-Goog-Upload-Command", "start")
	req.Header.Set("X-Goog-Upload-Header-Content-Length", strconv.Itoa(len(data)))
	req.Header.Set("X-Goog-Upload-Header-Content-Type", mimeType)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return uploadedFile{}, fmt.Errorf("gemini: start upload: %w", err)
	}
	res.Body.Close()
	uploadURL := res.Header.Get("X-Goog-Upload-URL")
	if res.StatusCode != http.StatusOK || uploadURL == "" {
		return uploadedFile{}, fmt.Errorf("gemini: start upload: HTTP %d", res.StatusCode)
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return uploadedFile{}, err
	}
	req.Header.Set("X-Goog-Upload-Offset", "0")
	req.Header.Set("X-Goog-Upload-Command", "upload, finalize")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return uploadedFile{}, fmt.Errorf("gemini: upload: %w", err)
	}
	var up struct {
		File uploadedFile `json:"file"`
	}
	err = json.NewDecoder(res.Body).Decode(&up)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || err != nil {
		return uploadedFile{}, fmt.Errorf("gemini: upload: HTTP %d", res.StatusCode)
	}

	f := up.File
	for f.State == "PROCESSING" {
		select {
		case <-ctx.Done():
			return uploadedFile{}, ctx.Err()
		case <-time.After(3 * time.Second):
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, filesBaseURL+"/v1beta/"+f.Name, nil)
		c.apiKey(req)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return uploadedFile{}, fmt.Errorf("gemini: file status: %w", err)
		}
		err = json.NewDecoder(res.Body).Decode(&f)
		res.Body.Close()
		if err != nil {
			return uploadedFile{}, fmt.Errorf("gemini: file status: %w", err)
		}
	}
	if f.State != "ACTIVE" {
		return uploadedFile{}, fmt.Errorf("gemini: uploaded video not usable (state %s)", f.State)
	}
	return f, nil
}

func (c *Client) deleteFile(ctx context.Context, name string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, filesBaseURL+"/v1beta/"+name, nil)
	if err != nil {
		return
	}
	c.apiKey(req)
	if res, err := http.DefaultClient.Do(req); err == nil {
		res.Body.Close()
	}
}
