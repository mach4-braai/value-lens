package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/store"
	pgvector "github.com/pgvector/pgvector-go"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	return dsn
}

func TestNew(t *testing.T) {
	dsn := testDSN(t)
	s, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
}

func TestInsertAndSearchChunks(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()
	s, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	company, err := s.GetCompanyByTicker(ctx, "AAPL")
	if err != nil {
		t.Fatalf("GetCompanyByTicker: %v", err)
	}

	filing := store.Filing{
		CompanyID:       company.ID,
		FormType:        "10-K",
		FiledDate:       "2024-01-01",
		AccessionNumber: "0000320193-24-000001",
		DocumentURL:     "https://example.com/filing.htm",
	}
	filingID, err := s.InsertFiling(ctx, filing)
	if err != nil {
		t.Fatalf("InsertFiling: %v", err)
	}

	emb := make([]float32, 768)
	emb[0] = 1.0
	chunk := store.Chunk{
		FilingID:   filingID,
		Section:    "item_1a",
		ChunkIndex: 0,
		Content:    "Apple faces significant competition in all areas of its business.",
		TokenCount: 10,
		Embedding:  pgvector.NewVector(emb),
	}
	_, err = s.InsertChunk(ctx, chunk)
	if err != nil {
		t.Fatalf("InsertChunk: %v", err)
	}

	results, err := s.SearchChunks(ctx, pgvector.NewVector(emb), 5)
	if err != nil {
		t.Fatalf("SearchChunks: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Ticker != "AAPL" {
		t.Errorf("expected ticker AAPL, got %s", results[0].Ticker)
	}
}
