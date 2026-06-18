package embedder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/embedder"
)

func TestEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		emb := make([]float64, 768)
		emb[0] = 0.123
		json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": emb,
		})
	}))
	defer srv.Close()

	e := embedder.NewOllama(srv.URL, "nomic-embed-text")
	vec, err := e.Embed(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 768 {
		t.Fatalf("expected 768 dims, got %d", len(vec))
	}
	if vec[0] == 0 {
		t.Error("expected non-zero first element")
	}
}

func TestEmbedBatch(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		emb := make([]float64, 768)
		emb[0] = float64(callCount)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": emb,
		})
	}))
	defer srv.Close()

	e := embedder.NewOllama(srv.URL, "nomic-embed-text")
	vecs, err := e.EmbedBatch(context.Background(), []string{"text1", "text2", "text3"})
	if err != nil {
		t.Fatalf("EmbedBatch: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
}
