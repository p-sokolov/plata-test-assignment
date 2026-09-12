-- +goose Up
-- +goose StatementBegin
CREATE TABLE idempotency_keys (
    key TEXT PRIMARY KEY,
    request_hash CHAR(64) NOT NULL CHECK (request_hash ~ '^[a-f0-9]{64}$'),
    update_id UUID NOT NULL REFERENCES quote_updates(id) ON DELETE RESTRICT,
    response_code SMALLINT NOT NULL CHECK (response_code BETWEEN 100 AND 599),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);

-- Allows a bounded cleanup job to delete expired idempotency records efficiently.
CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_idempotency_keys_expires_at;
DROP TABLE IF EXISTS idempotency_keys;
-- +goose StatementEnd
