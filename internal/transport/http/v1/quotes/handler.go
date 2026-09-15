package quotes

import (
	"context"

	"plata-test-assignment/internal/models"

	"github.com/google/uuid"
)

type service interface {
	Refresh(ctx context.Context, input models.RefreshInput) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.QuoteUpdate, error)
	GetLatest(ctx context.Context, pair string) (*models.QuoteUpdate, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}
