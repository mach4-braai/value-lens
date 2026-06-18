package ingest

import (
	"context"
	"fmt"
	"log"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/devanmcgeer/value-lens/internal/chunker"
	"github.com/devanmcgeer/value-lens/internal/edgar"
	"github.com/devanmcgeer/value-lens/internal/embedder"
	"github.com/devanmcgeer/value-lens/internal/parser"
	"github.com/devanmcgeer/value-lens/internal/store"
)

type Pipeline struct {
	store *store.Store
	emb   *embedder.Ollama
}

func NewPipeline(s *store.Store, emb *embedder.Ollama) *Pipeline {
	return &Pipeline{store: s, emb: emb}
}

func (p *Pipeline) IngestFiling(ctx context.Context, company store.Company, filing edgar.TenKFiling, htmlData []byte) error {
	// 1. Parse sections
	sections, err := parser.ParseTenK(htmlData)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	log.Printf("parsed %d sections from %s %s", len(sections), company.Ticker, filing.AccessionNumber)

	// 2. Insert filing record
	filingID, err := p.store.InsertFiling(ctx, store.Filing{
		CompanyID:       company.ID,
		FormType:        "10-K",
		FiledDate:       filing.FilingDate,
		AccessionNumber: filing.AccessionNumber,
		DocumentURL:     filing.DocumentURL(company.CIK),
	})
	if err != nil {
		return fmt.Errorf("insert filing: %w", err)
	}

	// 3. Chunk and embed each section
	chunkOpts := chunker.Options{MaxTokens: 512, Overlap: 64}
	totalChunks := 0
	for _, section := range sections {
		chunks := chunker.Chunk(section.Text, chunkOpts)
		for _, chunk := range chunks {
			vec, err := p.emb.Embed(ctx, chunk.Text)
			if err != nil {
				return fmt.Errorf("embed chunk: %w", err)
			}
			_, err = p.store.InsertChunk(ctx, store.Chunk{
				FilingID:   filingID,
				Section:    section.ID,
				ChunkIndex: chunk.Index,
				Content:    chunk.Text,
				TokenCount: chunk.TokenCount,
				Embedding:  pgvector.NewVector(vec),
			})
			if err != nil {
				return fmt.Errorf("insert chunk: %w", err)
			}
			totalChunks++
		}
	}
	log.Printf("ingested %d chunks for %s %s", totalChunks, company.Ticker, filing.AccessionNumber)
	return nil
}

func (p *Pipeline) IngestCompany(ctx context.Context, edgarClient *edgar.Client, company store.Company) error {
	subs, err := edgarClient.FetchSubmissions(ctx, company.CIK)
	if err != nil {
		return fmt.Errorf("fetch submissions for %s: %w", company.Ticker, err)
	}

	filings := subs.TenKFilings()
	log.Printf("found %d 10-K filings for %s", len(filings), company.Ticker)

	for _, filing := range filings {
		url := filing.DocumentURL(company.CIK)
		log.Printf("fetching %s", url)

		htmlData, err := edgarClient.FetchDocument(ctx, url)
		if err != nil {
			log.Printf("WARN: skip %s: %v", filing.AccessionNumber, err)
			continue
		}

		if err := p.IngestFiling(ctx, company, filing, htmlData); err != nil {
			log.Printf("WARN: ingest %s failed: %v", filing.AccessionNumber, err)
			continue
		}
	}
	return nil
}
