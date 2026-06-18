package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

type Store struct {
	pool *pgxpool.Pool
}

type Company struct {
	ID     int
	Ticker string
	Name   string
	CIK    string
}

type Filing struct {
	ID              int
	CompanyID       int
	FormType        string
	FiledDate       string
	AccessionNumber string
	DocumentURL     string
}

type Chunk struct {
	ID         int
	FilingID   int
	Section    string
	ChunkIndex int
	Content    string
	TokenCount int
	Embedding  pgvector.Vector
}

type ChunkResult struct {
	Content    string
	Section    string
	Ticker     string
	FiledDate  string
	Distance   float64
}

func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) GetCompanyByTicker(ctx context.Context, ticker string) (Company, error) {
	var c Company
	err := s.pool.QueryRow(ctx,
		"SELECT id, ticker, name, cik FROM companies WHERE ticker = $1", ticker,
	).Scan(&c.ID, &c.Ticker, &c.Name, &c.CIK)
	return c, err
}

func (s *Store) ListCompanies(ctx context.Context) ([]Company, error) {
	rows, err := s.pool.Query(ctx, "SELECT id, ticker, name, cik FROM companies")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Ticker, &c.Name, &c.CIK); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}
	return companies, rows.Err()
}

func (s *Store) InsertFiling(ctx context.Context, f Filing) (int, error) {
	var id int
	err := s.pool.QueryRow(ctx,
		`INSERT INTO filings (company_id, form_type, filed_date, accession_number, document_url)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (accession_number) DO UPDATE SET document_url = EXCLUDED.document_url
		 RETURNING id`,
		f.CompanyID, f.FormType, f.FiledDate, f.AccessionNumber, f.DocumentURL,
	).Scan(&id)
	return id, err
}

func (s *Store) InsertChunk(ctx context.Context, c Chunk) (int, error) {
	var id int
	err := s.pool.QueryRow(ctx,
		`INSERT INTO chunks (filing_id, section, chunk_index, content, token_count, embedding)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		c.FilingID, c.Section, c.ChunkIndex, c.Content, c.TokenCount, c.Embedding,
	).Scan(&id)
	return id, err
}

func (s *Store) SearchChunks(ctx context.Context, queryEmbedding pgvector.Vector, limit int) ([]ChunkResult, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT c.content, c.section, co.ticker, f.filed_date::text,
		        c.embedding <=> $1 AS distance
		 FROM chunks c
		 JOIN filings f ON f.id = c.filing_id
		 JOIN companies co ON co.id = f.company_id
		 ORDER BY c.embedding <=> $1
		 LIMIT $2`,
		queryEmbedding, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		if err := rows.Scan(&r.Content, &r.Section, &r.Ticker, &r.FiledDate, &r.Distance); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
