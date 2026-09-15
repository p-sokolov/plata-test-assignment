package quotes

import (
	"context"
	"errors"
	"strconv"
	"time"

	"plata-test-assignment/internal/models"
	"plata-test-assignment/internal/repository"
	"plata-test-assignment/internal/repository/postgres/sqlc/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ClaimNext atomically reserves one due update until lockedUntil.
// It returns nil, nil when the queue has no due work.
func (r *repo) ClaimNext(ctx context.Context, lockedUntil time.Time) (*models.QuoteUpdate, error) {
	q := repository.Queries(ctx, r.queries)

	update, err := q.ClaimNextQuoteUpdate(ctx, &lockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return toModelQuoteUpdate(update), nil
}

// RequeueExpired returns the number of updates whose lease expired and which are pending again.
func (r *repo) RequeueExpired(ctx context.Context) (int64, error) {
	q := repository.Queries(ctx, r.queries)
	return q.RequeueExpiredQuoteUpdates(ctx)
}

// 
func (r *repo) MarkSucceeded(ctx context.Context, rate float64, id uuid.UUID) (bool, error) {
	q := repository.Queries(ctx, r.queries)

	var pgRate pgtype.Numeric
	rateText := strconv.FormatFloat(rate, 'f', -1, 64)
	if err := pgRate.Scan(rateText); err != nil {
    	return false, err
	}

	params := storage.MarkQuoteUpdateSucceededParams{
		Rate: pgRate,
		ID:   id,
	}

	_, err := q.MarkQuoteUpdateSucceeded(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// ScheduleRetry releases an active update and postpones its next processing attempt.
func (r *repo) ScheduleRetry(ctx context.Context, errMsg string, nextAttemptAt time.Time, id uuid.UUID) (bool, error) {
	q := repository.Queries(ctx, r.queries)

	var pgErrMsg pgtype.Text
	if err := pgErrMsg.Scan(errMsg); err != nil {
		return false, err
	}

	params := storage.ScheduleQuoteUpdateRetryParams{
		ErrorMessage:  pgErrMsg,
		NextAttemptAt: nextAttemptAt,
		ID:            id,
	}

	_, err := q.ScheduleQuoteUpdateRetry(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// MarkFailed finishes an active update when it must not be retried again.
func (r *repo) MarkFailed(ctx context.Context, errMsg string, id uuid.UUID) (bool, error) {
	q := repository.Queries(ctx, r.queries)

	var pgErrMsg pgtype.Text
	if err := pgErrMsg.Scan(errMsg); err != nil {
		return false, err
	}

	params := storage.MarkQuoteUpdateFailedParams{
		ErrorMessage: pgErrMsg,
		ID:           id,
	}

	_, err := q.MarkQuoteUpdateFailed(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// LockIdempotencyKey serializes concurrent requests that use the same idempotency key.
func (r *repo) LockIdempotencyKey(ctx context.Context, hash string) error {
	q := repository.Queries(ctx, r.queries)

	err := q.LockIdempotencyKey(ctx, hash)
	if err != nil {
		return err
	}

	return nil
}

// GetIdempotencyKey returns nil, nil when the key is absent or expired.
func (r *repo) GetIdempotencyKey(ctx context.Context, key string) (*models.IdempotencyKey, error) {
	q := repository.Queries(ctx, r.queries)

	iKey, err := q.GetIdempotencyKey(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return toModelIdempotencyKey(iKey), nil
}

// UpsertIdempotencyKey saves the response associated with an idempotency key.
func (r *repo) UpsertIdempotencyKey(ctx context.Context, input *models.IdempotencyKey) (*models.IdempotencyKey, error) {
	q := repository.Queries(ctx, r.queries)

	params := storage.UpsertIdempotencyKeyParams{
		Key:          input.Key,
		RequestHash:  input.RequestHash,
		UpdateID:     input.UpdateID,
		ResponseCode: input.ResponseCode,
		ExpiresAt:    input.ExpiresAt,
	}

	update, err := q.UpsertIdempotencyKey(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return toModelIdempotencyKey(update), nil
}
