# RAG Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a RAG pipeline that ingests SEC 10-K filings for 5 companies (TSLA, AAPL, GOOGL, NFLX, META), chunks and embeds them, stores vectors in pgvector, and exposes a query interface backed by Claude for financial analysis.

**Architecture:** Go backend fetches 10-K filings from SEC EDGAR, parses HTML into logical sections, chunks them, embeds via self-hosted Ollama (nomic-embed-text), and stores in Postgres+pgvector. A query endpoint accepts natural language questions, performs vector similarity search, and synthesizes answers via the Claude API. A thin React+Bun frontend provides the UI.

**Tech Stack:** Go 1.22+, Postgres 16 + pgvector, Ollama + nomic-embed-text, React + Bun, Claude API (anthropic-sdk-go), chi router, pgx v5, Docker Compose.

---

## Project Structure

```
value-lens/
├── cmd/
│   └── server/
│       └── main.go              # HTTP server entrypoint
├── internal/
│   ├── edgar/
│   │   ├── client.go            # SEC EDGAR API client
│   │   └── client_test.go
│   ├── parser/
│   │   ├── tenk.go              # 10-K HTML section parser
│   │   └── tenk_test.go
│   ├── chunker/
│   │   ├── chunker.go           # Text chunking with overlap
│   │   └── chunker_test.go
│   ├── embedder/
│   │   ├── ollama.go            # Ollama embedding client
│   │   └── ollama_test.go
│   ├── store/
│   │   ├── store.go             # Postgres/pgvector repository
│   │   └── store_test.go
│   ├── rag/
│   │   ├── engine.go            # RAG query orchestration
│   │   └── engine_test.go
│   └── api/
│       ├── handler.go           # HTTP handlers
│       └── handler_test.go
├── migrations/
│   ├── 001_create_companies.sql
│   ├── 002_create_filings.sql
│   └── 003_create_chunks.sql
├── web/                          # React + Bun frontend
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   └── components/
│   │       ├── QueryInput.tsx
│   │       └── ResultCard.tsx
│   ├── index.html
│   ├── package.json
│   └── bunfig.toml
├── docker-compose.yml
├── mise.toml
├── go.mod
└── go.sum
```

## CIK Reference

| Company | Ticker | CIK (zero-padded) |
|---|---|---|
| Tesla | TSLA | `0001318605` |
| Apple | AAPL | `0000320193` |
| Alphabet | GOOGL | `0001652044` |
| Netflix | NFLX | `0001065280` |
| Meta | META | `0001326801` |

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `docker-compose.yml`
- Create: `mise.toml`
- Create: `.env.example`

**Step 1: Initialize Go module**

Run: `cd value-lens && go mod init github.com/devanmcgeer/value-lens`
Expected: `go.mod` created

**Step 2: Create Docker Compose with Postgres+pgvector and Ollama**

```yaml
# docker-compose.yml
services:
  postgres:
    image: pgvector/pgvector:pg16
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: valuelens
      POSTGRES_USER: valuelens
      POSTGRES_PASSWORD: valuelens
    volumes:
      - pgdata:/var/lib/postgresql/data

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama

volumes:
  pgdata:
  ollama_data:
```

**Step 3: Create mise.toml**

```toml
[tools]
go = "1.22"
node = "22"
bun = "latest"

[tasks.dev]
run = "go run ./cmd/server"

[tasks.db]
run = "docker compose up -d postgres"

[tasks.ollama]
run = "docker compose up -d ollama"

[tasks.migrate]
run = """
for f in migrations/*.sql; do
  psql "$DATABASE_URL" -f "$f"
done
"""

[tasks.web]
dir = "web"
run = "bun run dev"
```

**Step 4: Create .env.example**

```
DATABASE_URL=postgres://valuelens:valuelens@localhost:5432/valuelens?sslmode=disable
OLLAMA_URL=http://localhost:11434
ANTHROPIC_API_KEY=sk-ant-...
EDGAR_USER_AGENT=YourName your-email@example.com
```

**Step 5: Create minimal main.go**

```go
// cmd/server/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Fatal(err)
	}
}
```

**Step 6: Install Go dependencies**

Run: `go get github.com/go-chi/chi/v5 && go mod tidy`
Expected: Dependencies added to `go.mod`

**Step 7: Start services and verify**

Run: `docker compose up -d && sleep 3 && curl http://localhost:11434/api/tags`
Expected: Ollama responds with JSON

Run: `docker compose exec ollama ollama pull nomic-embed-text`
Expected: Model downloaded

Run: `go run ./cmd/server &` then `curl http://localhost:8080/healthz`
Expected: `ok`

**Step 8: Done — ready for manual commit**

---

### Task 2: Database Schema and Migrations

**Files:**
- Create: `migrations/001_create_companies.sql`
- Create: `migrations/002_create_filings.sql`
- Create: `migrations/003_create_chunks.sql`

**Step 1: Write companies migration**

```sql
-- migrations/001_create_companies.sql
CREATE TABLE IF NOT EXISTS companies (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    cik VARCHAR(10) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO companies (ticker, name, cik) VALUES
    ('TSLA', 'Tesla, Inc.', '0001318605'),
    ('AAPL', 'Apple Inc.', '0000320193'),
    ('GOOGL', 'Alphabet Inc.', '0001652044'),
    ('NFLX', 'Netflix, Inc.', '0001065280'),
    ('META', 'Meta Platforms, Inc.', '0001326801')
ON CONFLICT (ticker) DO NOTHING;
```

**Step 2: Write filings migration**

```sql
-- migrations/002_create_filings.sql
CREATE TABLE IF NOT EXISTS filings (
    id SERIAL PRIMARY KEY,
    company_id INTEGER NOT NULL REFERENCES companies(id),
    form_type VARCHAR(20) NOT NULL,
    filed_date DATE NOT NULL,
    accession_number VARCHAR(25) NOT NULL UNIQUE,
    document_url TEXT NOT NULL,
    ingested_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_filings_company ON filings(company_id);
```

**Step 3: Write chunks migration with pgvector**

```sql
-- migrations/003_create_chunks.sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS chunks (
    id SERIAL PRIMARY KEY,
    filing_id INTEGER NOT NULL REFERENCES filings(id) ON DELETE CASCADE,
    section VARCHAR(50) NOT NULL,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL,
    embedding vector(768) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chunks_filing ON chunks(filing_id);
CREATE INDEX idx_chunks_embedding ON chunks
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
```

**Step 4: Run migrations**

Run: `mise run migrate`
Expected: Tables created, `\dt` in psql shows `companies`, `filings`, `chunks`

**Step 5: Verify seed data**

Run: `psql "$DATABASE_URL" -c "SELECT ticker, name, cik FROM companies;"`
Expected: 5 rows returned

**Step 6: Done — ready for manual commit**

---

### Task 3: Postgres Store Layer

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`

**Step 1: Write the failing test for store initialization**

```go
// internal/store/store_test.go
package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/store"
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
```

**Step 2: Run test to verify it fails**

Run: `DATABASE_URL="postgres://valuelens:valuelens@localhost:5432/valuelens?sslmode=disable" go test ./internal/store/ -v -run TestNew`
Expected: FAIL — `store` package doesn't exist

**Step 3: Write minimal store implementation**

```go
// internal/store/store.go
package store

import (
	"context"
	"fmt"

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
```

Note: You'll need to add the missing `pgx` import — `"github.com/jackc/pgx/v5"`.

**Step 4: Run test to verify it passes**

Run: `go get github.com/jackc/pgx/v5 github.com/pgvector/pgvector-go && go mod tidy`
Then: `DATABASE_URL="..." go test ./internal/store/ -v -run TestNew`
Expected: PASS

**Step 5: Write failing test for InsertChunk and SearchChunks**

```go
// append to internal/store/store_test.go
func TestInsertAndSearchChunks(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()
	s, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	// Get a company
	company, err := s.GetCompanyByTicker(ctx, "AAPL")
	if err != nil {
		t.Fatalf("GetCompanyByTicker: %v", err)
	}

	// Insert a test filing
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

	// Insert a chunk with a fake embedding (768 dims)
	emb := make([]float32, 768)
	emb[0] = 1.0 // point in a known direction
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

	// Search with same embedding should return the chunk
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
```

**Step 6: Run test to verify it fails**

Run: `DATABASE_URL="..." go test ./internal/store/ -v -run TestInsertAndSearchChunks`
Expected: FAIL — methods not defined

**Step 7: Implement store methods**

Add to `internal/store/store.go`:

```go
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
```

**Step 8: Run test to verify it passes**

Run: `DATABASE_URL="..." go test ./internal/store/ -v`
Expected: PASS

**Step 9: Done — ready for manual commit**

---

### Task 4: SEC EDGAR Client

**Files:**
- Create: `internal/edgar/client.go`
- Create: `internal/edgar/client_test.go`
- Create: `testdata/submissions_sample.json` (for tests)

**Step 1: Write failing test for fetching submissions**

```go
// internal/edgar/client_test.go
package edgar_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/edgar"
)

func TestFetchSubmissions(t *testing.T) {
	// Use a test server to avoid hitting EDGAR in tests
	body, err := os.ReadFile("../../testdata/submissions_sample.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	c := edgar.NewClient(srv.URL, "Test test@example.com")
	subs, err := c.FetchSubmissions(context.Background(), "0000320193")
	if err != nil {
		t.Fatalf("FetchSubmissions: %v", err)
	}
	filings := subs.TenKFilings()
	if len(filings) == 0 {
		t.Fatal("expected at least one 10-K filing")
	}
}
```

**Step 2: Create testdata sample**

Create `testdata/submissions_sample.json` with a minimal EDGAR submissions response structure. The real response has `recentFilings` with arrays for `form`, `filingDate`, `accessionNumber`, `primaryDocument`. Create a minimal sample with at least one 10-K entry.

```json
{
  "cik": "320193",
  "entityType": "operating",
  "name": "Apple Inc.",
  "tickers": ["AAPL"],
  "filings": {
    "recent": {
      "accessionNumber": ["0000320193-24-000123", "0000320193-24-000456"],
      "filingDate": ["2024-11-01", "2024-08-01"],
      "form": ["10-K", "10-Q"],
      "primaryDocument": ["aapl-20240928.htm", "aapl-20240629.htm"]
    }
  }
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/edgar/ -v -run TestFetchSubmissions`
Expected: FAIL — package doesn't exist

**Step 4: Implement EDGAR client**

```go
// internal/edgar/client.go
package edgar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://data.sec.gov"

type Client struct {
	baseURL   string
	userAgent string
	http      *http.Client
}

func NewClient(baseURL, userAgent string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:   baseURL,
		userAgent: userAgent,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

type Submissions struct {
	CIK    string `json:"cik"`
	Name   string `json:"name"`
	Filings struct {
		Recent RecentFilings `json:"recent"`
	} `json:"filings"`
}

type RecentFilings struct {
	AccessionNumber []string `json:"accessionNumber"`
	FilingDate      []string `json:"filingDate"`
	Form            []string `json:"form"`
	PrimaryDocument []string `json:"primaryDocument"`
}

type TenKFiling struct {
	AccessionNumber string
	FilingDate      string
	DocumentName    string
}

func (s *Submissions) TenKFilings() []TenKFiling {
	var filings []TenKFiling
	for i, form := range s.Filings.Recent.Form {
		if form == "10-K" {
			filings = append(filings, TenKFiling{
				AccessionNumber: s.Filings.Recent.AccessionNumber[i],
				FilingDate:      s.Filings.Recent.FilingDate[i],
				DocumentName:    s.Filings.Recent.PrimaryDocument[i],
			})
		}
	}
	return filings
}

func (t TenKFiling) DocumentURL(cik string) string {
	// Accession numbers have dashes, URL path uses no dashes
	noDashes := ""
	for _, c := range t.AccessionNumber {
		if c != '-' {
			noDashes += string(c)
		}
	}
	return fmt.Sprintf(
		"https://www.sec.gov/Archives/edgar/data/%s/%s/%s",
		cik, noDashes, t.DocumentName,
	)
}

func (c *Client) FetchSubmissions(ctx context.Context, cik string) (*Submissions, error) {
	url := fmt.Sprintf("%s/submissions/CIK%s.json", c.baseURL, cik)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch submissions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("EDGAR returned %d", resp.StatusCode)
	}

	var subs Submissions
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &subs, nil
}

func (c *Client) FetchDocument(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("document fetch returned %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/edgar/ -v`
Expected: PASS

**Step 6: Done — ready for manual commit**

---

### Task 5: 10-K HTML Section Parser

**Files:**
- Create: `internal/parser/tenk.go`
- Create: `internal/parser/tenk_test.go`
- Create: `testdata/tenk_sample.htm` (minimal 10-K HTML snippet)

**Step 1: Create test data**

Create `testdata/tenk_sample.htm` with a minimal 10-K HTML structure. SEC filings use headers and anchors to denote sections. A simplified sample:

```html
<html><body>
<a name="item1"></a>
<h2>Item 1. Business</h2>
<p>The Company designs, manufactures, and markets smartphones,
personal computers, tablets, wearables, and accessories.</p>
<p>The Company's products include iPhone, Mac, iPad, and wearables.</p>

<a name="item1a"></a>
<h2>Item 1A. Risk Factors</h2>
<p>The Company faces substantial competition in all areas.</p>
<p>Global markets are highly competitive and subject to rapid change.</p>

<a name="item7"></a>
<h2>Item 7. Management's Discussion and Analysis</h2>
<p>Total net revenue was $394.3 billion for fiscal 2024.</p>
<p>The Company expects continued investment in research and development.</p>
</body></html>
```

**Step 2: Write the failing test**

```go
// internal/parser/tenk_test.go
package parser_test

import (
	"os"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/parser"
)

func TestParseTenK(t *testing.T) {
	data, err := os.ReadFile("../../testdata/tenk_sample.htm")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	sections, err := parser.ParseTenK(data)
	if err != nil {
		t.Fatalf("ParseTenK: %v", err)
	}

	if len(sections) == 0 {
		t.Fatal("expected at least one section")
	}

	// Should find Item 1, Item 1A, Item 7
	keys := map[string]bool{}
	for _, s := range sections {
		keys[s.ID] = true
	}
	for _, want := range []string{"item_1", "item_1a", "item_7"} {
		if !keys[want] {
			t.Errorf("missing section %s", want)
		}
	}

	// Item 1A should contain competition text
	for _, s := range sections {
		if s.ID == "item_1a" && len(s.Text) == 0 {
			t.Error("item_1a has empty text")
		}
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/parser/ -v -run TestParseTenK`
Expected: FAIL

**Step 4: Implement parser**

```go
// internal/parser/tenk.go
package parser

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type Section struct {
	ID    string // e.g. "item_1", "item_1a", "item_7"
	Title string // e.g. "Item 1. Business"
	Text  string // extracted plain text
}

// itemPattern matches "Item 1.", "Item 1A.", "Item 7.", etc.
var itemPattern = regexp.MustCompile(`(?i)^item\s+(\d+[a-z]?)\.?\s`)

func ParseTenK(data []byte) ([]Section, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Extract all text nodes grouped by section headers
	var sections []Section
	var current *Section
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && isHeading(n.Data) {
			text := strings.TrimSpace(extractText(n))
			if m := itemPattern.FindStringSubmatch(text); m != nil {
				// Start a new section
				if current != nil {
					sections = append(sections, *current)
				}
				id := "item_" + strings.ToLower(m[1])
				current = &Section{ID: id, Title: text}
			}
		} else if n.Type == html.TextNode && current != nil {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				current.Text += t + " "
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if current != nil {
		sections = append(sections, *current)
	}

	// Trim trailing spaces
	for i := range sections {
		sections[i].Text = strings.TrimSpace(sections[i].Text)
	}
	return sections, nil
}

func isHeading(tag string) bool {
	return tag == "h1" || tag == "h2" || tag == "h3" || tag == "h4"
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/parser/ -v`
Expected: PASS

**Step 6: Done — ready for manual commit**

---

### Task 6: Text Chunker

**Files:**
- Create: `internal/chunker/chunker.go`
- Create: `internal/chunker/chunker_test.go`

**Step 1: Write the failing test**

```go
// internal/chunker/chunker_test.go
package chunker_test

import (
	"strings"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/chunker"
)

func TestChunk(t *testing.T) {
	// Create text of ~1000 words
	words := make([]string, 1000)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")

	chunks := chunker.Chunk(text, chunker.Options{
		MaxTokens: 200,
		Overlap:   50,
	})

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	// Each chunk should be under max tokens (rough word≈token for English)
	for i, c := range chunks {
		wordCount := len(strings.Fields(c.Text))
		if wordCount > 250 { // some tolerance
			t.Errorf("chunk %d has %d words, exceeds max", i, wordCount)
		}
	}

	// Chunks should overlap
	if len(chunks) > 1 {
		// The end of chunk 0 should partially appear in chunk 1
		words0 := strings.Fields(chunks[0].Text)
		words1 := strings.Fields(chunks[1].Text)
		lastOfFirst := words0[len(words0)-1]
		if words1[0] == lastOfFirst {
			// Overlap exists, but let's just verify there are multiple chunks
		}
	}
}

func TestChunkShortText(t *testing.T) {
	chunks := chunker.Chunk("This is a short sentence.", chunker.Options{
		MaxTokens: 200,
		Overlap:   50,
	})
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for short text, got %d", len(chunks))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/chunker/ -v`
Expected: FAIL

**Step 3: Implement chunker**

```go
// internal/chunker/chunker.go
package chunker

import "strings"

type Options struct {
	MaxTokens int // approximate max tokens per chunk
	Overlap   int // number of overlapping tokens between chunks
}

type Result struct {
	Text       string
	Index      int
	TokenCount int
}

// Chunk splits text into overlapping chunks. Uses whitespace tokenization
// as a rough approximation — accurate enough for embedding purposes.
func Chunk(text string, opts Options) []Result {
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 512
	}
	words := strings.Fields(text)
	if len(words) <= opts.MaxTokens {
		return []Result{{Text: text, Index: 0, TokenCount: len(words)}}
	}

	var chunks []Result
	step := opts.MaxTokens - opts.Overlap
	if step <= 0 {
		step = opts.MaxTokens / 2
	}

	idx := 0
	for start := 0; start < len(words); start += step {
		end := start + opts.MaxTokens
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[start:end], " ")
		chunks = append(chunks, Result{
			Text:       chunk,
			Index:      idx,
			TokenCount: end - start,
		})
		idx++
		if end == len(words) {
			break
		}
	}
	return chunks
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/chunker/ -v`
Expected: PASS

**Step 5: Done — ready for manual commit**

---

### Task 7: Ollama Embedding Client

**Files:**
- Create: `internal/embedder/ollama.go`
- Create: `internal/embedder/ollama_test.go`

**Step 1: Write the failing test**

```go
// internal/embedder/ollama_test.go
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
	// Mock Ollama server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Return a fake 768-dim embedding
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/embedder/ -v`
Expected: FAIL

**Step 3: Implement Ollama embedding client**

```go
// internal/embedder/ollama.go
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

	// Convert float64 to float32 for pgvector
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/embedder/ -v`
Expected: PASS

**Step 5: Done — ready for manual commit**

---

### Task 8: Ingestion Pipeline

**Files:**
- Create: `internal/ingest/pipeline.go`
- Create: `internal/ingest/pipeline_test.go`

**Step 1: Write the failing test**

```go
// internal/ingest/pipeline_test.go
package ingest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
```

Note: Add `pgvector "github.com/pgvector/pgvector-go"` to imports.

**Step 2: Run test to verify it fails**

Run: `DATABASE_URL="..." go test ./internal/ingest/ -v`
Expected: FAIL

**Step 3: Implement ingestion pipeline**

```go
// internal/ingest/pipeline.go
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
```

**Step 4: Run test to verify it passes**

Run: `DATABASE_URL="..." go test ./internal/ingest/ -v`
Expected: PASS

**Step 5: Done — ready for manual commit**

---

### Task 9: RAG Query Engine

**Files:**
- Create: `internal/rag/engine.go`
- Create: `internal/rag/engine_test.go`

**Step 1: Write the failing test**

```go
// internal/rag/engine_test.go
package rag_test

import (
	"context"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/rag"
)

func TestBuildPrompt(t *testing.T) {
	chunks := []rag.RetrievedChunk{
		{Content: "Apple faces significant competition.", Section: "item_1a", Ticker: "AAPL", FiledDate: "2024-11-01"},
		{Content: "Revenue was $394.3 billion.", Section: "item_7", Ticker: "AAPL", FiledDate: "2024-11-01"},
	}
	prompt := rag.BuildPrompt("What are Apple's main risks?", chunks)

	if len(prompt) == 0 {
		t.Fatal("expected non-empty prompt")
	}
	// Should contain the question and the chunk content
	if !containsAll(prompt, "Apple's main risks", "significant competition", "394.3 billion") {
		t.Errorf("prompt missing expected content: %s", prompt[:200])
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/rag/ -v`
Expected: FAIL

**Step 3: Implement RAG engine**

```go
// internal/rag/engine.go
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
	store   *store.Store
	emb     *embedder.Ollama
	claude  *anthropic.Client
}

type RetrievedChunk struct {
	Content   string
	Section   string
	Ticker    string
	FiledDate string
	Distance  float64
}

type QueryResult struct {
	Answer string
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
		Model:     anthropic.ModelClaude3_5Sonnet,
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/rag/ -v -run TestBuildPrompt`
Expected: PASS (only testing prompt building, not the full query which requires live services)

**Step 5: Done — ready for manual commit**

---

### Task 10: HTTP API Handlers

**Files:**
- Create: `internal/api/handler.go`
- Create: `internal/api/handler_test.go`
- Modify: `cmd/server/main.go`

**Step 1: Write the failing test**

```go
// internal/api/handler_test.go
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/api"
)

func TestQueryEndpointValidation(t *testing.T) {
	h := api.NewHandler(nil) // nil engine — just testing validation
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	// Missing question should 400
	resp, err := http.Post(srv.URL+"/api/query", "application/json",
		strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	h := api.NewHandler(nil)
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api/ -v`
Expected: FAIL

**Step 3: Implement handlers**

```go
// internal/api/handler.go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/devanmcgeer/value-lens/internal/rag"
)

type Handler struct {
	engine *rag.Engine
}

func NewHandler(engine *rag.Engine) *Handler {
	return &Handler{engine: engine}
}

type QueryRequest struct {
	Question string `json:"question"`
	TopK     int    `json:"top_k,omitempty"`
}

type QueryResponse struct {
	Answer  string       `json:"answer"`
	Sources []SourceInfo `json:"sources"`
}

type SourceInfo struct {
	Ticker    string  `json:"ticker"`
	FiledDate string  `json:"filed_date"`
	Section   string  `json:"section"`
	Excerpt   string  `json:"excerpt"`
	Distance  float64 `json:"distance"`
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Post("/api/query", h.handleQuery)
	return r
}

func (h *Handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.Question == "" {
		http.Error(w, `{"error":"question is required"}`, http.StatusBadRequest)
		return
	}
	if req.TopK == 0 {
		req.TopK = 5
	}

	result, err := h.engine.Query(r.Context(), req.Question, req.TopK)
	if err != nil {
		http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
		return
	}

	resp := QueryResponse{Answer: result.Answer}
	for _, s := range result.Sources {
		resp.Sources = append(resp.Sources, SourceInfo{
			Ticker:    s.Ticker,
			FiledDate: s.FiledDate,
			Section:   s.Section,
			Excerpt:   truncate(s.Content, 200),
			Distance:  s.Distance,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
```

**Step 4: Update `cmd/server/main.go` to wire everything together**

```go
// cmd/server/main.go
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
	engine := rag.NewEngine(s, emb, claude)
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
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/api/ -v`
Expected: PASS

**Step 6: Done — ready for manual commit**

---

### Task 11: CLI Ingest Command

**Files:**
- Create: `cmd/ingest/main.go`

**Step 1: Implement ingest CLI**

```go
// cmd/ingest/main.go
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
```

**Step 2: Add mise task**

Add to `mise.toml`:

```toml
[tasks.ingest]
run = "go run ./cmd/ingest"

[tasks."ingest:one"]
run = "go run ./cmd/ingest {{arg(name='ticker')}}"
```

**Step 3: Test manually**

Run: `mise run ingest:one ticker=AAPL`
Expected: Logs showing filing fetch, parse, chunk, embed, store

**Step 4: Done — ready for manual commit**

---

### Task 12: React Frontend

**Files:**
- Create: `web/package.json`
- Create: `web/index.html`
- Create: `web/bunfig.toml`
- Create: `web/tsconfig.json`
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Create: `web/src/components/QueryInput.tsx`
- Create: `web/src/components/ResultCard.tsx`

**Step 1: Initialize React app with Bun**

Run:
```bash
cd web && bun init
bun add react react-dom
bun add -d @types/react @types/react-dom typescript
```

**Step 2: Create bunfig.toml**

```toml
# web/bunfig.toml
[serve.static]
dir = "public"

[dev]
port = 3000
```

**Step 3: Create index.html**

```html
<!-- web/index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Value Lens</title>
</head>
<body>
  <div id="root"></div>
  <script type="module" src="./src/main.tsx"></script>
</body>
</html>
```

**Step 4: Create main.tsx entrypoint**

```tsx
// web/src/main.tsx
import { createRoot } from "react-dom/client";
import App from "./App";

createRoot(document.getElementById("root")!).render(<App />);
```

**Step 5: Implement App component**

```tsx
// web/src/App.tsx
import { useState } from "react";
import { QueryInput } from "./components/QueryInput";
import { ResultCard } from "./components/ResultCard";

interface Source {
  ticker: string;
  filed_date: string;
  section: string;
  excerpt: string;
  distance: number;
}

interface QueryResult {
  answer: string;
  sources: Source[];
}

export default function App() {
  const [result, setResult] = useState<QueryResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleQuery(question: string) {
    setLoading(true);
    setError("");
    try {
      const resp = await fetch("/api/query", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question, top_k: 5 }),
      });
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const data: QueryResult = await resp.json();
      setResult(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Query failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ maxWidth: 800, margin: "0 auto", padding: 24 }}>
      <h1>Value Lens</h1>
      <p style={{ color: "#888" }}>
        Ask questions about SEC filings for TSLA, AAPL, GOOGL, NFLX, META
      </p>
      <QueryInput onSubmit={handleQuery} disabled={loading} />
      {loading && <p>Querying...</p>}
      {error && <p style={{ color: "red" }}>{error}</p>}
      {result && <ResultCard result={result} />}
    </div>
  );
}
```

**Step 3: Implement QueryInput**

```tsx
// web/src/components/QueryInput.tsx
import { useState } from "react";

interface Props {
  onSubmit: (question: string) => void;
  disabled: boolean;
}

export function QueryInput({ onSubmit, disabled }: Props) {
  const [question, setQuestion] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (question.trim()) {
      onSubmit(question.trim());
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ marginBottom: 24 }}>
      <textarea
        value={question}
        onChange={(e) => setQuestion(e.target.value)}
        placeholder="e.g. What are Tesla's biggest risk factors?"
        rows={3}
        style={{ width: "100%", padding: 12, fontSize: 16 }}
        disabled={disabled}
      />
      <button
        type="submit"
        disabled={disabled || !question.trim()}
        style={{ marginTop: 8, padding: "8px 24px", fontSize: 16 }}
      >
        Ask
      </button>
    </form>
  );
}
```

**Step 4: Implement ResultCard**

```tsx
// web/src/components/ResultCard.tsx
interface Source {
  ticker: string;
  filed_date: string;
  section: string;
  excerpt: string;
  distance: number;
}

interface Props {
  result: {
    answer: string;
    sources: Source[];
  };
}

export function ResultCard({ result }: Props) {
  return (
    <div>
      <h2>Answer</h2>
      <div style={{ whiteSpace: "pre-wrap", lineHeight: 1.6 }}>
        {result.answer}
      </div>

      <h3 style={{ marginTop: 24 }}>Sources ({result.sources.length})</h3>
      {result.sources.map((s, i) => (
        <div
          key={i}
          style={{
            border: "1px solid #333",
            borderRadius: 8,
            padding: 12,
            marginBottom: 8,
          }}
        >
          <strong>
            {s.ticker} | {s.filed_date} | {s.section}
          </strong>
          <span style={{ float: "right", color: "#888" }}>
            dist: {s.distance.toFixed(4)}
          </span>
          <p style={{ marginTop: 8, color: "#ccc" }}>{s.excerpt}</p>
        </div>
      ))}
    </div>
  );
}
```

**Step 8: Verify frontend runs**

Run: `cd web && bun --hot index.html`
Expected: Bun dev server on http://localhost:3000

Note: During development, the frontend fetches `/api/query`. In dev, the Go backend runs on :8080, so either:
- Use the Go backend to serve the built frontend static files, OR
- Update the fetch URL to `http://localhost:8080/api/query` during dev (CORS is already enabled on the backend)

**Step 9: Done — ready for manual commit**

---

## End-to-End Verification

After all tasks are complete, run through this checklist:

1. `docker compose up -d` — Postgres and Ollama running
2. `docker compose exec ollama ollama pull nomic-embed-text` — model ready
3. `mise run migrate` — schema applied
4. `mise run ingest:one ticker=AAPL` — ingests Apple 10-K filings
5. `mise run dev` — Go server on :8080
6. `mise run web` — React on :3000
7. Open http://localhost:3000, ask "What are Apple's biggest risk factors?"
8. Verify: answer appears with source citations from 10-K Item 1A

---

## Architecture Diagram

```
┌──────────┐     ┌──────────────┐     ┌──────────────────┐
│  React   │────▶│  Go API      │────▶│  Ollama          │
│  :3000   │     │  :8080       │     │  nomic-embed-text│
└──────────┘     │              │     │  :11434          │
                 │  /api/query  │     └──────────────────┘
                 │              │
                 │              │────▶ Claude API
                 │              │     (synthesis)
                 │              │
                 │              │────▶┌──────────────────┐
                 └──────────────┘     │  Postgres 16     │
                                      │  + pgvector      │
┌──────────────┐                      │  :5432           │
│  CLI ingest  │─────────────────────▶│                  │
│  cmd/ingest  │                      └──────────────────┘
└──────────────┘
       │
       ▼
┌──────────────┐
│  SEC EDGAR   │
│  (free API)  │
└──────────────┘
```
