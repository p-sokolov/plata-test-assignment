-- +goose Up
-- +goose StatementBegin
CREATE TABLE outbox_events (
    id BIGSERIAL PRIMARY KEY,
    quote_id UUID NOT NULL REFERENCES quote_updates(id) ON DELETE RESTRICT,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- NULL, until the event is delivered
    published_at TIMESTAMPTZ,
    -- Number of event delivery attempts
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    -- Do not repeat the delivery before this time
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Rapid selection of undelivered events across multiple publishers
CREATE INDEX idx_quotes_outbox_pending_publish
ON outbox_events (next_attempt_at, id)
WHERE published_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_quotes_outbox_pending_publish;
DROP TABLE IF EXISTS outbox_events;
-- +goose StatementEnd
