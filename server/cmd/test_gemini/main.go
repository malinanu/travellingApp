// Command test_gemini is a live smoke test for the configured Gemini models,
// exercising the same internal/gemini client the server uses: a two-round
// tool call (which needs Gemini 3's thought_signature echoed back), a final
// tool-free round like guide.go's last round, and a batch embedding.
//
//	GEMINI_API_KEY=... go run ./cmd/test_gemini
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spotto/server/internal/gemini"
)

func main() {
	c := gemini.NewClient(os.Getenv("GEMINI_API_KEY"), os.Getenv("GEMINI_CHAT_MODEL"), os.Getenv("GEMINI_EMBED_MODEL"))
	ctx := context.Background()
	fmt.Printf("chat model %s, embed model %s\n", c.ChatModel, c.EmbedModel)

	tools := []gemini.Tool{{Type: "function", Function: gemini.ToolFunction{
		Name:        "search_places",
		Description: "Search for real places. Always use this before recommending one.",
		Parameters: map[string]any{"type": "object", "properties": map[string]any{
			"query": map[string]any{"type": "string"},
		}},
	}}}
	messages := []gemini.Message{
		{Role: "system", Content: "You are a travel assistant for Sri Lanka. Keep replies short."},
		{Role: "user", Content: "Recommend a quiet cafe in Galle."},
	}

	for round := range 3 {
		roundTools := tools
		if round == 2 {
			roundTools = nil
		}
		msg, usage, err := c.Complete(ctx, messages, roundTools)
		if err != nil {
			fail("chat round %d: %v", round+1, err)
		}
		fmt.Printf("chat round %d: %d tool calls, usage %+v\n", round+1, len(msg.ToolCalls), *usage)
		if len(msg.ToolCalls) == 0 {
			fmt.Printf("reply: %s\n", msg.Content)
			break
		}
		messages = append(messages, *msg)
		for _, tc := range msg.ToolCalls {
			messages = append(messages, gemini.Message{Role: "tool", ToolCallID: tc.ID,
				Content: `{"places":[{"id":"p1","name":"Pedlar's Inn Cafe","reasons":["quiet","fort views"]}]}`})
		}
	}

	vecs, usage, err := c.Embed(ctx, []string{"quiet cafe with wifi", "lively beach bar"})
	if err != nil {
		fail("embed: %v", err)
	}
	fmt.Printf("embed: %d vectors of %d dims, est usage %+v\n", len(vecs), len(vecs[0]), *usage)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL "+format+"\n", args...)
	os.Exit(1)
}
