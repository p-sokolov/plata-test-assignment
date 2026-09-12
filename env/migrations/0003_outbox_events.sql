-- +goose Up
-- +goose StatementBegin
CREATE TABLE outbox_events (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id UUID NOT NULL REFERENCES quote_updates(id) ON DELETE RESTRICT,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Supports concurrent outbox publishers using FOR UPDATE SKIP LOCKED.
CREATE INDEX idx_outbox_events_pending_publish
ON outbox_events (next_attempt_at, id)
WHERE published_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_outbox_events_pending_publish;
DROP TABLE IF EXISTS outbox_events;
-- +goose StatementEnd
