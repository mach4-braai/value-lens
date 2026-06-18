package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/devanmcgeer/value-lens/internal/embedder"
	"github.com/devanmcgeer/value-lens/internal/store"
)

type Engine struct {
	store  *store.Store
	emb    *embedder.Ollama
	claude *anthropic.Client
}

type RetrievedChunk struct {
	Content   string
	Section   string
	Ticker    string
	FiledDate string
	Distance  float64
}

type QueryResult struct {
	Answer  string
	Sources []RetrievedChunk
}

func NewEngine(s *store.Store, emb *embedder.Ollama, claude *anthropic.Client) *Engine {
	return &Engine{store: s, emb: emb, claude: claude}
}

func BuildPrompt(question string, chunks []RetrievedChunk) string {
	var sb strings.Builder
	sb.WriteString("You are a financial analyst assistant. Answer the question using ONLY the provided SEC filing excerpts. ")
	sb.WriteString("Cite the source (ticker, filing date, section) for each claim. ")
	sb.WriteString("If the excerpts don't contain enough information, say so.\n\n")
	sb.WriteString("## SEC Filing Excerpts\n\n")

	for i, c := range chunks {
		sb.WriteString(fmt.Sprintf("### Source %d [%s | %s | %s]\n", i+1, c.Ticker, c.FiledDate, c.Section))
		sb.WriteString(c.Content)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Question\n\n")
	sb.WriteString(question)
	return sb.String()
}

func (e *Engine) Query(ctx context.Context, question string, topK int) (*QueryResult, error) {
	// 1. Embed the question
	vec, err := e.emb.Embed(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// 2. Vector search
	results, err := e.store.SearchChunks(ctx, pgvector.NewVector(vec), topK)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	chunks := make([]RetrievedChunk, len(results))
	for i, r := range results {
		chunks[i] = RetrievedChunk{
			Content:   r.Content,
			Section:   r.Section,
			Ticker:    r.Ticker,
			FiledDate: r.FiledDate,
			Distance:  r.Distance,
		}
	}

	// 3. Build prompt and call Claude
	prompt := BuildPrompt(question, chunks)
	msg, err := e.claude.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_5,
		MaxTokens: 2048,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(prompt),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}

	answer := ""
	for _, block := range msg.Content {
		if block.Type == "text" {
			answer += block.Text
		}
	}

	return &QueryResult{Answer: answer, Sources: chunks}, nil
}
