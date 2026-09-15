-- +goose Up
-- +goose StatementBegin
CREATE TABLE idempotency_keys (
    -- Client key for secure request retries
    key TEXT PRIMARY KEY,
    -- SHA-256 of the request body; detects key reuse with another request
    request_hash CHAR(64) NOT NULL CHECK (request_hash ~ '^[a-f0-9]{64}$'),
    update_id UUID NOT NULL REFERENCES quote_updates(id) ON DELETE RESTRICT,
    -- The HTTP status returned when repeating the same request
    response_code SMALLINT NOT NULL CHECK (response_code BETWEEN 100 AND 599),
    -- After this time, the recording can be deleted
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);

-- Quickly finds expired keys for cleanup
CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_idempotency_keys_expires_at;
DROP TABLE IF EXISTS idempotency_keys;
-- +goose StatementEnd
