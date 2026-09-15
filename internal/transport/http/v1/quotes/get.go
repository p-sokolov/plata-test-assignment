package quotes

import (
	"context"
	"errors"

	"plata-test-assignment/internal/errorz"
	"plata-test-assignment/internal/models"
	v1 "plata-test-assignment/internal/transport/http/v1"
)

func (h *Handler) GetQuoteUpdateByID(
	ctx context.Context,
	request v1.GetQuoteUpdateByIDRequestObject,
) (v1.GetQuoteUpdateByIDResponseObject, error) {
	quote, err := h.svc.GetByID(ctx, request.Id)
	if err != nil {
		if errors.Is(err, errorz.ErrQuoteNotFound) {
			return v1.GetQuoteUpdateByID404JSONResponse{Error: err.Error()}, nil
		}
		return nil, err
	}

	return v1.GetQuoteUpdateByID200JSONResponse(toV1QuoteUpdate(quote)), nil
}

func (h *Handler) GetLatestQuote(
	ctx context.Context,
	request v1.GetLatestQuoteRequestObject,
) (v1.GetLatestQuoteResponseObject, error) {
	quote, err := h.svc.GetLatest(ctx, request.Params.Pair)
	if err != nil {
		switch {
		case errors.Is(err, errorz.ErrUnsupportedCurrency):
			return v1.GetLatestQuote400JSONResponse{Error: err.Error()}, nil
		case errors.Is(err, errorz.ErrQuoteNotFound):
			return v1.GetLatestQuote404JSONResponse{Error: err.Error()}, nil
		default:
			return nil, err
		}
	}

	if quote.Rate == nil {
		return nil, errorz.ErrInternalServerError
	}

	return v1.GetLatestQuote200JSONResponse{
		CurrencyPair: quote.CurrencyPair,
		Rate:         *quote.Rate,
		UpdatedAt:    quote.UpdatedAt,
	}, nil
}

func toV1QuoteUpdate(quote *models.QuoteUpdate) v1.UpdateQuoteDetailsResponse {
	return v1.UpdateQuoteDetailsResponse{
		Id:           quote.ID,
		CurrencyPair: quote.CurrencyPair,
		Status:       v1.UpdateQuoteDetailsResponseStatus(quote.Status),
		Rate:         quote.Rate,
		UpdatedAt:    &quote.UpdatedAt,
		ErrorMessage: quote.ErrorMessage,
		CreatedAt:    quote.CreatedAt,
	}
}
