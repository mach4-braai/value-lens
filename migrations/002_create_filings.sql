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
