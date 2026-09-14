package quotes

import (
	"context"

	"plata-test-assignment/internal/errorz"
	"plata-test-assignment/internal/models"

	"github.com/google/uuid"
)

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*models.QuoteUpdate, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetLatest(ctx context.Context, pair string) (*models.QuoteUpdate, error) {
	pair = normalizePair(pair)

	if isSupported(pair) {
		return s.repo.GetLatest(ctx, pair)
	}
	return nil, errorz.ErrUnsupportedCurrency

}
