package ingest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/devanmcgeer/value-lens/internal/edgar"
	"github.com/devanmcgeer/value-lens/internal/embedder"
	"github.com/devanmcgeer/value-lens/internal/ingest"
	"github.com/devanmcgeer/value-lens/internal/store"
)

func TestIngestFiling(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer s.Close()

	// Mock Ollama
	ollamaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		emb := make([]float64, 768)
		emb[0] = 0.5
		json.NewEncoder(w).Encode(map[string]interface{}{"embedding": emb})
	}))
	defer ollamaSrv.Close()
	emb := embedder.NewOllama(ollamaSrv.URL, "nomic-embed-text")

	// Read test 10-K HTML
	htmlData, err := os.ReadFile("../../testdata/tenk_sample.htm")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	company, err := s.GetCompanyByTicker(ctx, "AAPL")
	if err != nil {
		t.Fatalf("get company: %v", err)
	}

	filing := edgar.TenKFiling{
		AccessionNumber: "0000320193-24-TEST001",
		FilingDate:      "2024-11-01",
		DocumentName:    "test.htm",
	}

	p := ingest.NewPipeline(s, emb)
	err = p.IngestFiling(ctx, company, filing, htmlData)
	if err != nil {
		t.Fatalf("IngestFiling: %v", err)
	}

	// Verify chunks were stored
	fakeEmb := make([]float32, 768)
	fakeEmb[0] = 0.5
	results, err := s.SearchChunks(ctx, pgvector.NewVector(fakeEmb), 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected chunks to be stored")
	}
}
