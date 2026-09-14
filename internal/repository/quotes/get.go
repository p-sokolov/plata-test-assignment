package quotes

import (
	"context"
	"errors"

	"plata-test-assignment/internal/errorz"
	"plata-test-assignment/internal/models"
	"plata-test-assignment/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *repo) GetByID(ctx context.Context, id uuid.UUID) (*models.QuoteUpdate, error) {
	q := repository.Queries(ctx, r.queries)

	quote, err := q.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorz.ErrQuoteNotFound
		}
		return nil, err
	}

	return toModelQuoteUpdate(quote), nil
}

func (r *repo) GetLatest(ctx context.Context, pair string) (*models.QuoteUpdate, error) {
	q := repository.Queries(ctx, r.queries)

	quote, err := q.GetLatestByPair(ctx, pair)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorz.ErrQuoteNotFound
		}
		return nil, err
	}

	return toModelQuoteUpdate(quote), nil
}
