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
		// check cached data
		if s.cache != nil {
			cached, err := s.cache.GetLatest(ctx, pair)
			if err == nil && cached != nil {
				return &models.QuoteUpdate{
					CurrencyPair: cached.CurrencyPair,
					Rate:         &cached.Rate,
					UpdatedAt:    cached.UpdatedAt,
				}, nil
			}
		}

		// go to database
		quote, err := s.repo.GetLatest(ctx, pair)
		if err != nil {
			return nil, err
		}

		// caching new data
		if s.cache != nil && quote.Rate != nil {
			_ = s.cache.SetLatest(ctx, models.LatestQuote{
				CurrencyPair: quote.CurrencyPair,
				Rate:         *quote.Rate,
				UpdatedAt:    quote.UpdatedAt,
			})
		}
		return quote, nil
	}

	return nil, errorz.ErrUnsupportedCurrency
}
