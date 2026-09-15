package quotes

import (
	"context"

	"plata-test-assignment/internal/errorz"
	"plata-test-assignment/internal/models"

	"github.com/google/uuid"
)

func (s *service) Refresh(ctx context.Context, input models.RefreshInput) (uuid.UUID, error) {
	input.CurrencyPair = normalizePair(input.CurrencyPair)

	if isSupported(input.CurrencyPair) {
		return s.repo.Refresh(ctx, input)
	}
	return uuid.Nil, errorz.ErrUnsupportedCurrency
}
