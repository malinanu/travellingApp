package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

const embeddingsEndpoint = "https://generativelanguage.googleapis.com/v1beta/openai/embeddings"

// EmbeddingModel is the semantic-search vector embedding model.
// Google Gemini text-embedding-004 provides high quality embeddings.
const EmbeddingModel = "text-embedding-004"

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

	// Independent of the caller's context.
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
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
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

	var parsed embeddingsResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("gemini decode embed response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini embed error: %s", parsed.Error.Message)
	}
	if len(parsed.Data) != len(inputs) {
		return nil, fmt.Errorf("gemini embed expected %d embeddings, got %d", len(inputs), len(parsed.Data))
	}

	sort.Slice(parsed.Data, func(i, j int) bool { return parsed.Data[i].Index < parsed.Data[j].Index })
	out := make([][]float64, len(parsed.Data))
	for i, d := range parsed.Data {
		out[i] = d.Embedding
	}
	return out, nil
}
