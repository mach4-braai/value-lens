package main

import (
	"context"
	"log"
	"os"

	"github.com/devanmcgeer/value-lens/internal/edgar"
	"github.com/devanmcgeer/value-lens/internal/embedder"
	"github.com/devanmcgeer/value-lens/internal/ingest"
	"github.com/devanmcgeer/value-lens/internal/store"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	userAgent := os.Getenv("EDGAR_USER_AGENT")
	if userAgent == "" {
		log.Fatal("EDGAR_USER_AGENT is required (e.g. 'YourName email@example.com')")
	}

	s, err := store.New(ctx, dsn)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer s.Close()

	emb := embedder.NewOllama(ollamaURL, "nomic-embed-text")
	edgarClient := edgar.NewClient("", userAgent)
	pipeline := ingest.NewPipeline(s, emb)

	// If ticker provided as arg, ingest only that company
	var tickers []string
	if len(os.Args) > 1 {
		tickers = os.Args[1:]
	}

	companies, err := s.ListCompanies(ctx)
	if err != nil {
		log.Fatalf("list companies: %v", err)
	}

	for _, company := range companies {
		if len(tickers) > 0 && !contains(tickers, company.Ticker) {
			continue
		}
		log.Printf("ingesting %s (%s)...", company.Ticker, company.Name)
		if err := pipeline.IngestCompany(ctx, edgarClient, company); err != nil {
			log.Printf("ERROR: %s: %v", company.Ticker, err)
		}
	}
	log.Println("done")
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
