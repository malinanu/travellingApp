package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const embeddingsEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:batchEmbedContents"

// EmbeddingModel is the semantic-search vector embedding model.
const EmbeddingModel = "models/text-embedding-004"

type embedContentRequest struct {
	Model   string `json:"model"`
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
}

type batchEmbedRequest struct {
	Requests []embedContentRequest `json:"requests"`
}

type batchEmbedResponse struct {
	Embeddings []struct {
		Values []float64 `json:"values"`
	} `json:"embeddings"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Embed returns one embedding vector per input string, in the same order
// as inputs.
func Embed(ctx context.Context, apiKey string, inputs []string) ([][]float64, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var reqBody batchEmbedRequest
	for _, text := range inputs {
		req := embedContentRequest{Model: EmbeddingModel}
		req.Content.Parts = []struct{ Text string `json:"text"` }{{Text: text}}
		reqBody.Requests = append(reqBody.Requests, req)
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, embeddingsEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("x-goog-api-key", apiKey)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini embed request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini embed unexpected status %d", res.StatusCode)
	}

	var parsed batchEmbedResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("gemini decode embed response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini embed error: %s", parsed.Error.Message)
	}
	if len(parsed.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("gemini embed expected %d embeddings, got %d", len(inputs), len(parsed.Embeddings))
	}

	out := make([][]float64, len(parsed.Embeddings))
	for i, e := range parsed.Embeddings {
		out[i] = e.Values
	}
	return out, nil
}
