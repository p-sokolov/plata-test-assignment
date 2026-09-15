package quotes

import (
	"context"
	"errors"
	"fmt"
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
	if update.LeaseToken == nil {
		return nil, fmt.Errorf("claimed quote update %s without lease token", update.ID)
	}

	return toModelQuoteUpdate(update), nil
}

// RequeueExpired returns the number of updates whose lease expired and which are pending again.
func (r *repo) RequeueExpired(ctx context.Context) (int64, error) {
	q := repository.Queries(ctx, r.queries)
	return q.RequeueExpiredQuoteUpdates(ctx)
}

// MarkSucceeded completes an update only when leaseToken still belongs to this worker.
func (r *repo) MarkSucceeded(ctx context.Context, rate float64, id, leaseToken uuid.UUID) (bool, error) {
	q := repository.Queries(ctx, r.queries)

	var pgRate pgtype.Numeric
	rateText := strconv.FormatFloat(rate, 'f', -1, 64)
	if err := pgRate.Scan(rateText); err != nil {
		return false, err
	}

	params := storage.MarkQuoteUpdateSucceededParams{
		Rate:       pgRate,
		ID:         id,
		LeaseToken: &leaseToken,
	}

	_, err := q.MarkQuoteUpdateSucceeded(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// ScheduleRetry releases an update only when leaseToken still belongs to this worker.
func (r *repo) ScheduleRetry(ctx context.Context, errMsg string, nextAttemptAt time.Time, id, leaseToken uuid.UUID) (bool, error) {
	q := repository.Queries(ctx, r.queries)

	var pgErrMsg pgtype.Text
	if err := pgErrMsg.Scan(errMsg); err != nil {
		return false, err
	}

	params := storage.ScheduleQuoteUpdateRetryParams{
		ErrorMessage:  pgErrMsg,
		NextAttemptAt: nextAttemptAt,
		ID:            id,
		LeaseToken:    &leaseToken,
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

// MarkFailed finishes an update only when leaseToken still belongs to this worker.
func (r *repo) MarkFailed(ctx context.Context, errMsg string, id, leaseToken uuid.UUID) (bool, error) {
	q := repository.Queries(ctx, r.queries)

	var pgErrMsg pgtype.Text
	if err := pgErrMsg.Scan(errMsg); err != nil {
		return false, err
	}

	params := storage.MarkQuoteUpdateFailedParams{
		ErrorMessage: pgErrMsg,
		ID:           id,
		LeaseToken:   &leaseToken,
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
