-- name: CreateOrGetActiveQuoteUpdate :one
INSERT INTO quote_updates (currency_pair, status)
VALUES ($1, 'PENDING')
ON CONFLICT (currency_pair) WHERE status IN ('PENDING', 'PROCESSING')
DO UPDATE SET currency_pair = EXCLUDED.currency_pair
RETURNING quote_updates.*;

-- name: GetLatestByPair :one
SELECT quote_updates.*
FROM quote_updates
WHERE currency_pair = $1
  AND status = 'SUCCESS'
ORDER BY updated_at DESC
LIMIT 1;

-- name: GetByID :one
SELECT quote_updates.*
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
    lease_token = gen_random_uuid(),
    attempt_count = attempt_count + 1,
    updated_at = NOW()
WHERE id IN (SELECT id FROM candidate)
RETURNING quote_updates.*;

-- name: RequeueExpiredQuoteUpdates :execrows
UPDATE quote_updates
SET status = 'PENDING',
    locked_until = NULL,
    lease_token = NULL,
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
    lease_token = NULL,
    updated_at = NOW()
WHERE id = $2
  AND status = 'PROCESSING'
  AND locked_until > NOW()
  AND lease_token = $3
RETURNING quote_updates.*;

-- name: ScheduleQuoteUpdateRetry :one
UPDATE quote_updates
SET status = 'PENDING',
    error_message = $1,
    locked_until = NULL,
    lease_token = NULL,
    next_attempt_at = $2,
    updated_at = NOW()
WHERE id = $3
  AND status = 'PROCESSING'
  AND locked_until > NOW()
  AND lease_token = $4
RETURNING quote_updates.*;

-- name: MarkQuoteUpdateFailed :one
UPDATE quote_updates
SET status = 'FAILED',
    rate = NULL,
    error_message = $1,
    locked_until = NULL,
    lease_token = NULL,
    updated_at = NOW()
WHERE id = $2
  AND status = 'PROCESSING'
  AND locked_until > NOW()
  AND lease_token = $3
RETURNING quote_updates.*;

-- name: LockIdempotencyKey :exec
SELECT pg_advisory_xact_lock(hashtext($1));

-- name: GetIdempotencyKey :one
SELECT idempotency_keys.*
FROM idempotency_keys
WHERE key = $1
  AND expires_at > NOW();

-- name: UpsertIdempotencyKey :one
INSERT INTO idempotency_keys (key, request_hash, update_id, response_code, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (key) DO UPDATE
SET request_hash = EXCLUDED.request_hash,
    update_id = EXCLUDED.update_id,
    response_code = EXCLUDED.response_code,
    expires_at = EXCLUDED.expires_at,
    created_at = NOW()
RETURNING idempotency_keys.*;
