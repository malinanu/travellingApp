package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const chatEndpoint = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"

// ChatModel is a free-tier chat model.
const ChatModel = "gemini-2.5-flash"

// ErrRateLimited distinguishes a 429 so callers can surface something
// friendlier than a generic failure — free-model rate limits are real and
// tight, unlike a paid tier.
var ErrRateLimited = errors.New("gemini: rate limited (429)")

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
	// OpenAI tool-calling convention — callers must json.Unmarshal it themselves.
	Arguments string `json:"arguments"`
}

// Tool describes one function the model may choose to call, as a JSON
// schema — see internal/http/guide.go for the actual tool definitions.
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
// Gemini's OpenAI-compatible endpoint and returns the model's response message
// — which may itself carry ToolCalls instead of a final Content answer; the
// caller is responsible for executing those and calling Complete again with the
// extended message list (see guide.go's tool-calling loop).
func Complete(ctx context.Context, apiKey string, messages []Message, tools []Tool) (*Message, error) {
	// Bounded independently of the caller's context.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	body, err := json.Marshal(chatRequest{Model: ChatModel, Messages: messages, Tools: tools})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("x-goog-api-key", apiKey)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini chat request: %w", err)
	}
	defer res.Body.Close()
	
	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini chat: unexpected status %d", res.StatusCode)
	}

	var parsed chatResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("gemini decode chat response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini chat error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("gemini chat: empty response")
	}
	return &parsed.Choices[0].Message, nil
}
