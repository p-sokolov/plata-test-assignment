package quotes

import (
	"context"
	"strings"

	"plata-test-assignment/internal/models"

	"github.com/google/uuid"
)

type repo interface {
	Refresh(ctx context.Context, input models.RefreshInput) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.QuoteUpdate, error)
	GetLatest(ctx context.Context, pair string) (*models.QuoteUpdate, error)
}

type service struct {
	repo repo
}

func New(repo repo) *service {
	return &service{repo: repo}
}

func isSupported(pair string) bool {
	if len(pair) != 7 {
		return false
	}

	supported := map[string]struct{}{
		"USD": {},
		"EUR": {},
		"MXN": {},
	}

	from := pair[:3]
	to := pair[4:]

	if _, ok := supported[from]; ok {
		if _, ok := supported[to]; ok {
			return true
		}
	}

	return false
}

func normalizePair(pair string) string {
	return strings.ToUpper(strings.TrimSpace(pair))
}
