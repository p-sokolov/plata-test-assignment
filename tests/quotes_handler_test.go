package tests

import (
	"context"
	"testing"
	"time"

	"plata-test-assignment/internal/errorz"
	"plata-test-assignment/internal/models"
	v1 "plata-test-assignment/internal/transport/http/v1"
	handlerquotes "plata-test-assignment/internal/transport/http/v1/quotes"

	"github.com/google/uuid"
)

type quoteHandlerServiceStub struct {
	refreshID  uuid.UUID
	refreshErr error
	latest     *models.QuoteUpdate
	latestErr  error
}

func (s *quoteHandlerServiceStub) Refresh(context.Context, models.RefreshInput) (uuid.UUID, error) {
	return s.refreshID, s.refreshErr
}

func (s *quoteHandlerServiceStub) GetByID(context.Context, uuid.UUID) (*models.QuoteUpdate, error) {
	return nil, nil
}

func (s *quoteHandlerServiceStub) GetLatest(context.Context, string) (*models.QuoteUpdate, error) {
	return s.latest, s.latestErr
}

func TestRefreshQuoteReturnsAcceptedTaskID(t *testing.T) {
	t.Parallel()

	wantID := uuid.New()
	handler := handlerquotes.New(&quoteHandlerServiceStub{refreshID: wantID})

	response, err := handler.RefreshQuote(context.Background(), v1.RefreshQuoteRequestObject{
		Params: v1.RefreshQuoteParams{IdempotencyKey: "test-idempotency-key"},
		Body:   &v1.RefreshQuoteRequest{CurrencyPair: "EUR/MXN"},
	})
	if err != nil {
		t.Fatalf("RefreshQuote() error = %v", err)
	}
	accepted, ok := response.(v1.RefreshQuote202JSONResponse)
	if !ok {
		t.Fatalf("response type = %T, want RefreshQuote202JSONResponse", response)
	}
	if accepted.UpdateId != wantID {
		t.Errorf("update_id = %s, want %s", accepted.UpdateId, wantID)
	}
}

func TestRefreshQuoteReturnsConflictForReusedKey(t *testing.T) {
	t.Parallel()

	handler := handlerquotes.New(&quoteHandlerServiceStub{refreshErr: errorz.ErrIdempotencyConflict})
	response, err := handler.RefreshQuote(context.Background(), v1.RefreshQuoteRequestObject{
		Params: v1.RefreshQuoteParams{IdempotencyKey: "test-idempotency-key"},
		Body:   &v1.RefreshQuoteRequest{CurrencyPair: "EUR/MXN"},
	})
	if err != nil {
		t.Fatalf("RefreshQuote() error = %v", err)
	}
	if _, ok := response.(v1.RefreshQuote409JSONResponse); !ok {
		t.Errorf("response type = %T, want RefreshQuote409JSONResponse", response)
	}
}

func TestGetLatestQuoteReturnsLatestSuccessfulRate(t *testing.T) {
	t.Parallel()

	rate := 21.42
	updatedAt := time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)
	handler := handlerquotes.New(&quoteHandlerServiceStub{latest: &models.QuoteUpdate{
		CurrencyPair: "EUR/MXN",
		Rate:         &rate,
		UpdatedAt:    updatedAt,
	}})

	response, err := handler.GetLatestQuote(context.Background(), v1.GetLatestQuoteRequestObject{
		Params: v1.GetLatestQuoteParams{Pair: "EUR/MXN"},
	})
	if err != nil {
		t.Fatalf("GetLatestQuote() error = %v", err)
	}
	latest, ok := response.(v1.GetLatestQuote200JSONResponse)
	if !ok {
		t.Fatalf("response type = %T, want GetLatestQuote200JSONResponse", response)
	}
	if latest.Rate != rate || latest.CurrencyPair != "EUR/MXN" || !latest.UpdatedAt.Equal(updatedAt) {
		t.Errorf("latest response = %+v, want rate and metadata from successful quote", latest)
	}
}
