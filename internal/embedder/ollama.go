package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Ollama struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewOllama(baseURL, model string) *Ollama {
	return &Ollama{
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func (o *Ollama) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(embeddingRequest{
		Model:  o.model,
		Prompt: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned %d", resp.StatusCode)
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	vec := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}

func (o *Ollama) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := o.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embed text %d: %w", i, err)
		}
		results[i] = vec
	}
	return results, nil
}
