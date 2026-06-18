package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/devanmcgeer/value-lens/internal/api"
	"github.com/devanmcgeer/value-lens/internal/embedder"
	"github.com/devanmcgeer/value-lens/internal/rag"
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

	s, err := store.New(ctx, dsn)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer s.Close()

	emb := embedder.NewOllama(ollamaURL, "nomic-embed-text")
	claude := anthropic.NewClient()
	engine := rag.NewEngine(s, emb, &claude)
	handler := api.NewHandler(engine)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), handler.Router()); err != nil {
		log.Fatal(err)
	}
}
