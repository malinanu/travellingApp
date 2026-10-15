package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const embedEndpointFmt = "https://generativelanguage.googleapis.com/v1beta/models/%s:batchEmbedContents"

type embedContentRequest struct {
	Model   string `json:"model"`
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
	OutputDimensionality int `json:"outputDimensionality,omitempty"`
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
// as inputs. The API reports no token counts for embeddings, so the
// returned Usage is a characters/4 estimate (Estimated=true).
func (c *Client) Embed(ctx context.Context, inputs []string) ([][]float64, *Usage, error) {
	if len(inputs) == 0 {
		return nil, nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var reqBody batchEmbedRequest
	chars := 0
	for _, text := range inputs {
		req := embedContentRequest{Model: "models/" + c.EmbedModel, OutputDimensionality: EmbeddingDimensions}
		req.Content.Parts = []struct {
			Text string `json:"text"`
		}{{Text: text}}
		reqBody.Requests = append(reqBody.Requests, req)
		chars += len(text)
	}
	usage := &Usage{PromptTokens: chars / 4, TotalTokens: chars / 4, Estimated: true}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, err
	}

	res, err := post(ctx, fmt.Sprintf(embedEndpointFmt, c.EmbedModel), body, c.apiKey)
	if err != nil {
		return nil, usage, fmt.Errorf("gemini embed request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, usage, upstreamError("embed", res)
	}

	var parsed batchEmbedResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, usage, fmt.Errorf("gemini decode embed response: %w", err)
	}
	if parsed.Error != nil {
		return nil, usage, fmt.Errorf("gemini embed error: %s", parsed.Error.Message)
	}
	if len(parsed.Embeddings) != len(inputs) {
		return nil, usage, fmt.Errorf("gemini embed expected %d embeddings, got %d", len(inputs), len(parsed.Embeddings))
	}

	out := make([][]float64, len(parsed.Embeddings))
	for i, e := range parsed.Embeddings {
		out[i] = e.Values
	}
	return out, usage, nil
}
