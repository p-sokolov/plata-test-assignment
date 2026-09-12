-- name: CreateOrGetActiveQuoteUpdate :one
INSERT INTO quote_updates (currency_pair, status)
VALUES ($1, 'PENDING')
ON CONFLICT (currency_pair) WHERE status IN ('PENDING', 'PROCESSING')
DO UPDATE SET currency_pair = EXCLUDED.currency_pair
RETURNING id, currency_pair, status, rate, error_message, attempt_count, next_attempt_at, locked_until, created_at, updated_at;

-- name: GetLatestByPair :one
SELECT id, currency_pair, status, rate, error_message, attempt_count, next_attempt_at, locked_until, created_at, updated_at
FROM quote_updates
WHERE currency_pair = $1
  AND status = 'SUCCESS'
ORDER BY updated_at DESC
LIMIT 1;

-- name: GetByID :one
SELECT id, currency_pair, status, rate, error_message, attempt_count, next_attempt_at, locked_until, created_at, updated_at
FROM quote_updates
WHERE id = $1;

-- name: ClaimNextQuoteUpdate :one
WITH candidate AS (
    SELECT id
    FROM quote_updates
    WHERE status = 'PENDING'
      AND next_attempt_at <= NOW()
    ORDER BY next_attempt_at, created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE quote_updates
SET status = 'PROCESSING',
    locked_until = $1,
    attempt_count = attempt_count + 1,
    updated_at = NOW()
WHERE id IN (SELECT id FROM candidate)
RETURNING id, currency_pair, status, rate, error_message, attempt_count, next_attempt_at, locked_until, created_at, updated_at;

-- name: RequeueExpiredQuoteUpdates :execrows
UPDATE quote_updates
SET status = 'PENDING',
    locked_until = NULL,
    next_attempt_at = NOW(),
    updated_at = NOW()
WHERE status = 'PROCESSING'
  AND locked_until < NOW();

-- name: MarkQuoteUpdateSucceeded :one
UPDATE quote_updates
SET status = 'SUCCESS',
    rate = $1,
    error_message = NULL,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = $2
  AND status = 'PROCESSING'
  AND locked_until > NOW()
RETURNING id, currency_pair, status, rate, error_message, attempt_count, next_attempt_at, locked_until, created_at, updated_at;

-- name: ScheduleQuoteUpdateRetry :one
UPDATE quote_updates
SET status = 'PENDING',
    error_message = $1,
    locked_until = NULL,
    next_attempt_at = $2,
    updated_at = NOW()
WHERE id = $3
  AND status = 'PROCESSING'
  AND locked_until > NOW()
RETURNING id, currency_pair, status, rate, error_message, attempt_count, next_attempt_at, locked_until, created_at, updated_at;

-- name: MarkQuoteUpdateFailed :one
UPDATE quote_updates
SET status = 'FAILED',
    rate = NULL,
    error_message = $1,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = $2
  AND status = 'PROCESSING'
  AND locked_until > NOW()
RETURNING id, currency_pair, status, rate, error_message, attempt_count, next_attempt_at, locked_until, created_at, updated_at;
