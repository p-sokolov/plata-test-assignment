package quotes

import (
	"context"
	"errors"

	"plata-test-assignment/internal/errorz"
	"plata-test-assignment/internal/models"
	v1 "plata-test-assignment/internal/transport/http/v1"
)

func (h *Handler) RefreshQuote(
	ctx context.Context,
	request v1.RefreshQuoteRequestObject,
) (v1.RefreshQuoteResponseObject, error) {
	if request.Body == nil {
		return v1.RefreshQuote400JSONResponse{Error: "request body is required"}, nil
	}

	updateID, err := h.svc.Refresh(ctx, models.RefreshInput{
		CurrencyPair:   request.Body.CurrencyPair,
		IdempotencyKey: request.Params.IdempotencyKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, errorz.ErrUnsupportedCurrency):
			return v1.RefreshQuote400JSONResponse{Error: err.Error()}, nil
		case errors.Is(err, errorz.ErrIdempotencyConflict):
			return v1.RefreshQuote409JSONResponse{Error: err.Error()}, nil
		default:
			return nil, err
		}
	}

	return v1.RefreshQuote202JSONResponse{UpdateId: updateID}, nil
}
