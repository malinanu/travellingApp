// Package openrouter calls OpenRouter's OpenAI-compatible chat-completions
// API (https://openrouter.ai/docs) with tool/function calling, for the
// Local Guide chat (see internal/http/guide.go). Uses a free model, so
// this feature carries zero marginal LLM cost.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"
)

const endpoint = "https://openrouter.ai/api/v1/chat/completions"
const embeddingsEndpoint = "https://openrouter.ai/api/v1/embeddings"

// EmbeddingModel is a free-tier multimodal embedding model — used for
// Local Guide's semantic place search (see internal/recommend,
// internal/http/embedding_worker.go), same zero-marginal-cost rationale as
// the chat model above.
const EmbeddingModel = "nvidia/llama-nemotron-embed-vl-1b-v2:free"

// model is hardcoded to a free-tier model — keeps this feature at zero
// marginal cost. Free models have tighter rate limits than paid ones (see
// ErrRateLimited) and can be slower to respond.
const model = "google/gemma-4-31b-it:free"

// ErrRateLimited distinguishes a 429 so callers can surface something
// friendlier than a generic failure — free-model rate limits are real and
// tight, unlike a paid tier.
var ErrRateLimited = errors.New("openrouter: rate limited (429)")

// Message is one turn in an OpenAI-style chat-completion request/response.
// Role is "system"/"user"/"assistant"/"tool". An assistant message either
// carries Content (a final answer) or ToolCalls (the model wants to call
// one or more tools before it can answer); a tool message carries the
// result of one prior ToolCall, identified by ToolCallID.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name string `json:"name"`
	// Arguments is a JSON-encoded string (not a nested object) per the
	// OpenAI tool-calling convention OpenRouter mirrors — callers must
	// json.Unmarshal it themselves.
	Arguments string `json:"arguments"`
}

// Tool describes one function the model may choose to call, as a JSON
// schema — see internal/http/guide.go for the actual tool definitions
// (search_places, get_place_details).
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // JSON schema object
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Complete sends the conversation so far (plus any available tools) to
// OpenRouter and returns the model's response message — which may itself
// carry ToolCalls instead of a final Content answer; the caller is
// responsible for executing those and calling Complete again with the
// extended message list (see guide.go's tool-calling loop).
func Complete(ctx context.Context, apiKey string, messages []Message, tools []Tool) (*Message, error) {
	// Bounded independently of the caller's context — a slow/hanging
	// OpenRouter response (free-tier inference can be slow) shouldn't hold
	// the request open indefinitely.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	body, err := json.Marshal(chatRequest{Model: model, Messages: messages, Tools: tools})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	// Optional per OpenRouter's docs — identifies this app on their
	// leaderboards, no effect on functionality if omitted.
	req.Header.Set("HTTP-Referer", "https://spotto.app")
	req.Header.Set("X-Title", "Spotto Local Guide")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter: request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter: unexpected status %d", res.StatusCode)
	}

	var parsed chatResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("openrouter: decode response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("openrouter: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openrouter: empty response")
	}
	return &parsed.Choices[0].Message, nil
}

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Embed returns one embedding vector per input string, in the same order
// as inputs — regardless of what order the API returns them in (the
// response's own Index field is authoritative for that).
func Embed(ctx context.Context, apiKey string, inputs []string) ([][]float64, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// Independent of the caller's context, same rationale as Complete.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	body, err := json.Marshal(embeddingsRequest{Model: EmbeddingModel, Input: inputs})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, embeddingsEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "https://spotto.app")
	req.Header.Set("X-Title", "Spotto Local Guide")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter: embed request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter: embed unexpected status %d", res.StatusCode)
	}

	var parsed embeddingsResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("openrouter: decode embed response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("openrouter: %s", parsed.Error.Message)
	}
	if len(parsed.Data) != len(inputs) {
		return nil, fmt.Errorf("openrouter: expected %d embeddings, got %d", len(inputs), len(parsed.Data))
	}

	sort.Slice(parsed.Data, func(i, j int) bool { return parsed.Data[i].Index < parsed.Data[j].Index })
	out := make([][]float64, len(parsed.Data))
	for i, d := range parsed.Data {
		out[i] = d.Embedding
	}
	return out, nil
}
