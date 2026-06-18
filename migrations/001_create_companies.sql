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
