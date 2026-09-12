-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE quote_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    currency_pair VARCHAR(7) NOT NULL CHECK (currency_pair ~ '^[A-Z]{3}/[A-Z]{3}$'),
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'PROCESSING', 'SUCCESS', 'FAILED')),
    rate NUMERIC(18, 8),
    error_message TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (status = 'SUCCESS' AND rate IS NOT NULL AND error_message IS NULL AND locked_until IS NULL)
        OR (status = 'FAILED' AND rate IS NULL AND error_message IS NOT NULL AND locked_until IS NULL)
        OR (status = 'PENDING' AND rate IS NULL AND locked_until IS NULL)
        OR (status = 'PROCESSING' AND rate IS NULL AND locked_until IS NOT NULL)
    )
);

-- index for /quotes/latest (fast search latest successfully updated quotes)
CREATE INDEX idx_quote_updates_pair_latest
ON quote_updates (currency_pair, updated_at DESC)
WHERE status = 'SUCCESS';

-- Only one active update per pair may call the external provider at a time.
CREATE UNIQUE INDEX uniq_quote_updates_active_pair
ON quote_updates (currency_pair) 
WHERE status IN ('PENDING', 'PROCESSING');

-- Supports efficient work claiming by horizontally scaled workers.
CREATE INDEX idx_quote_updates_pending_claim
ON quote_updates (next_attempt_at, created_at)
WHERE status = 'PENDING';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_quote_updates_pending_claim;
DROP INDEX IF EXISTS uniq_quote_updates_active_pair;
DROP INDEX IF EXISTS idx_quote_updates_pair_latest;
DROP TABLE IF EXISTS quote_updates;
-- +goose StatementEnd
