package quotes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"plata-test-assignment/internal/errorz"
	"plata-test-assignment/internal/models"
	"plata-test-assignment/internal/repository"
	"plata-test-assignment/internal/repository/postgres/sqlc/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const idempotencyTTL = 24 * time.Hour

func (r *repo) Refresh(ctx context.Context, input models.RefreshInput) (uuid.UUID, error) {
	var updateID uuid.UUID
	transactor := repository.NewTransactor(r.db)
	
	refreshTx := func(ctx context.Context) error {
		q := repository.Queries(ctx, r.queries)
		if err := q.LockIdempotencyKey(ctx, input.IdempotencyKey); err != nil {
			return err
		}

		requestHash := hashRequest(input.CurrencyPair)
		existing, err := q.GetIdempotencyKey(ctx, input.IdempotencyKey)
		
		switch {
			case err == nil:
				if existing.RequestHash != requestHash {
					return errorz.ErrIdempotencyConflict
				}
				updateID = existing.UpdateID
				return nil
			case !errors.Is(err, pgx.ErrNoRows):
				return err
		}

		quoteUpdate, err := q.CreateOrGetActiveQuoteUpdate(ctx, input.CurrencyPair)
		if err != nil {
			return err
		}

		key, err := q.UpsertIdempotencyKey(ctx, storage.UpsertIdempotencyKeyParams{
			Key:          input.IdempotencyKey,
			RequestHash:  requestHash,
			UpdateID:     quoteUpdate.ID,
			ResponseCode: 202,
			ExpiresAt:    time.Now().UTC().Add(idempotencyTTL),
		})
		if err != nil {
			return err
		}

		updateID = key.UpdateID
		return nil
	}
	
	err := transactor.RunInTx(ctx, refreshTx)
	if err != nil {
		return uuid.Nil, err
	}

	return updateID, nil
}

func hashRequest(currencyPair string) string {
	sum := sha256.Sum256([]byte(currencyPair))
	return hex.EncodeToString(sum[:])
}
