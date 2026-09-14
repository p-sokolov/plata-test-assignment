package quotes

import (
	"plata-test-assignment/internal/models"
	"plata-test-assignment/internal/repository/postgres/sqlc/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repo struct {
	db      *pgxpool.Pool
	queries *storage.Queries
}

func New(db *pgxpool.Pool) *repo {
	return &repo{
		db:      db,
		queries: storage.New(db),
	}
}

func toModelQuoteUpdate(q storage.QuoteUpdate) *models.QuoteUpdate {
	var rate *float64
	if value, err := q.Rate.Float64Value(); err == nil && value.Valid {
		rate = &value.Float64
	}

	var errorMessage *string
	if q.ErrorMessage.Valid {
		errorMessage = &q.ErrorMessage.String
	}

	return &models.QuoteUpdate{
		ID:            q.ID,
		CurrencyPair:  q.CurrencyPair,
		Status:        q.Status,
		Rate:          rate,
		ErrorMessage:  errorMessage,
		AttemptCount:  q.AttemptCount,
		NextAttemptAt: q.NextAttemptAt,
		LockedUntil:   q.LockedUntil,
		LeaseToken:    q.LeaseToken,
		CreatedAt:     q.CreatedAt,
		UpdatedAt:     q.UpdatedAt,
	}
}
