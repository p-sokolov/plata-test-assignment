package tests

import (
	"context"
	"testing"
	"time"

	"plata-test-assignment/internal/errorz"
	"plata-test-assignment/internal/models"
	servicequotes "plata-test-assignment/internal/service/quotes"

	"github.com/google/uuid"
)

type quoteServiceRepoStub struct {
	refreshInput models.RefreshInput
	refreshID    uuid.UUID
	refreshErr   error
	latestPair   string
	latestQuote  *models.QuoteUpdate
	latestErr    error
}

type latestCacheStub struct {
	quote *models.LatestQuote
	err   error
}

func (s *latestCacheStub) GetLatest(context.Context, string) (*models.LatestQuote, error) {
	return s.quote, s.err
}

func (s *latestCacheStub) SetLatest(context.Context, models.LatestQuote) error { return nil }

func (s *quoteServiceRepoStub) Refresh(_ context.Context, input models.RefreshInput) (uuid.UUID, error) {
	s.refreshInput = input
	return s.refreshID, s.refreshErr
}

func (s *quoteServiceRepoStub) GetByID(context.Context, uuid.UUID) (*models.QuoteUpdate, error) {
	return nil, nil
}

func (s *quoteServiceRepoStub) GetLatest(_ context.Context, pair string) (*models.QuoteUpdate, error) {
	s.latestPair = pair
	return s.latestQuote, s.latestErr
}

func TestQuoteServiceRefreshNormalizesPair(t *testing.T) {
	t.Parallel()

	wantID := uuid.New()
	repo := &quoteServiceRepoStub{refreshID: wantID}
	service := servicequotes.New(repo, nil)

	gotID, err := service.Refresh(context.Background(), models.RefreshInput{
		CurrencyPair:   " eur/mxn ",
		IdempotencyKey: "key",
	})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if gotID != wantID {
		t.Errorf("Refresh() id = %s, want %s", gotID, wantID)
	}
	if repo.refreshInput.CurrencyPair != "EUR/MXN" {
		t.Errorf("repository pair = %q, want EUR/MXN", repo.refreshInput.CurrencyPair)
	}
}

func TestQuoteServiceRejectsUnsupportedPair(t *testing.T) {
	t.Parallel()

	repo := &quoteServiceRepoStub{}
	service := servicequotes.New(repo, nil)

	_, err := service.Refresh(context.Background(), models.RefreshInput{CurrencyPair: "GBP/JPY"})
	if err != errorz.ErrUnsupportedCurrency {
		t.Fatalf("Refresh() error = %v, want %v", err, errorz.ErrUnsupportedCurrency)
	}
	if repo.refreshInput.CurrencyPair != "" {
		t.Error("repository must not be called for an unsupported pair")
	}
}

func TestQuoteServiceGetLatestNormalizesPair(t *testing.T) {
	t.Parallel()

	repo := &quoteServiceRepoStub{latestQuote: &models.QuoteUpdate{CurrencyPair: "USD/EUR"}}
	service := servicequotes.New(repo, nil)

	_, err := service.GetLatest(context.Background(), " usd/eur ")
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if repo.latestPair != "USD/EUR" {
		t.Errorf("repository pair = %q, want USD/EUR", repo.latestPair)
	}
}

func TestQuoteServiceGetLatestUsesCache(t *testing.T) {
	t.Parallel()

	rate := 21.42
	updatedAt := time.Now().UTC()
	repo := &quoteServiceRepoStub{}
	cache := &latestCacheStub{quote: &models.LatestQuote{CurrencyPair: "EUR/MXN", Rate: rate, UpdatedAt: updatedAt}}
	service := servicequotes.New(repo, cache)

	quote, err := service.GetLatest(context.Background(), "EUR/MXN")
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if quote.Rate == nil || *quote.Rate != rate || !quote.UpdatedAt.Equal(updatedAt) {
		t.Errorf("GetLatest() = %+v, want cached quote", quote)
	}
	if repo.latestPair != "" {
		t.Error("repository must not be called on cache hit")
	}
}
