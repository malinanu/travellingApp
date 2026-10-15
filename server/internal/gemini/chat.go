package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const chatEndpoint = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"

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
	// ExtraContent carries Gemini 3's thought_signature
	// ({"google":{"thought_signature":"..."}}). It must be echoed back
	// verbatim with the assistant turn on the next request, or Gemini
	// rejects the follow-up with a 400 — so it's kept opaque, never parsed.
	ExtraContent json.RawMessage `json:"extra_content,omitempty"`
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
	Usage *Usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Complete sends the conversation so far (plus any available tools) to
// Gemini's OpenAI-compatible endpoint and returns the model's response message
// — which may itself carry ToolCalls instead of a final Content answer; the
// caller is responsible for executing those and calling Complete again with the
// extended message list (see guide.go's tool-calling loop).
func (c *Client) Complete(ctx context.Context, messages []Message, tools []Tool) (*Message, *Usage, error) {
	// Bounded independently of the caller's context.
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	body, err := json.Marshal(chatRequest{Model: c.ChatModel, Messages: messages, Tools: tools})
	if err != nil {
		return nil, nil, err
	}

	res, err := post(ctx, chatEndpoint, body, c.bearer)
	if err != nil {
		return nil, nil, fmt.Errorf("gemini chat request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, nil, upstreamError("chat", res)
	}

	var parsed chatResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, nil, fmt.Errorf("gemini decode chat response: %w", err)
	}
	if parsed.Error != nil {
		return nil, parsed.Usage, fmt.Errorf("gemini chat error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, parsed.Usage, fmt.Errorf("gemini chat: empty response")
	}
	return &parsed.Choices[0].Message, parsed.Usage, nil
}
