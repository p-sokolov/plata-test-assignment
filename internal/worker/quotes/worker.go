package quotes

import (
	"context"
	"log/slog"
	"time"

	"plata-test-assignment/internal/models"

	"github.com/google/uuid"
)

type rateProvider interface {
	GetRate(ctx context.Context, pair string) (float64, error)
}

type repository interface {
	ClaimNext(ctx context.Context, lockedUntil time.Time) (*models.QuoteUpdate, error)
    RequeueExpired(ctx context.Context) (int64, error)
    MarkSucceeded(ctx context.Context, rate float64, id uuid.UUID) (bool, error)
    ScheduleRetry(ctx context.Context, errMsg string, nextAttemptAt time.Time, id uuid.UUID) (bool, error)
    MarkFailed(ctx context.Context, errMsg string, id uuid.UUID) (bool, error)
}

type Config struct {
    PollInterval time.Duration
    LeaseDuration time.Duration
    MaxAttempts int32
}

type Worker struct {
    repo     repository
    provider rateProvider
    cfg      *Config
    logger   *slog.Logger
}

func New(r repository, p rateProvider, c *Config, l *slog.Logger) *Worker {
	return &Worker{
		repo: r,
	    provider: p,
	    cfg: c,
	    logger: l,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	
	for {
		if err := w.processAvailable(ctx); err != nil {
			w.logger.Error("worker iteration failed", err)
		}
		
		select {
			case <-ctx.Done():
				return
			case <-ticker.C:
		}
	}
}

func (w *Worker) ProcessOnce(ctx context.Context) (bool, error) {
	// Take one of PENDING task
	nextTask, err := w.repo.ClaimNext(ctx, time.Now().Add(w.cfg.LeaseDuration))
	if err != nil {
		return false, err
	}
	if nextTask == nil {
		return false, nil
	}

	// Get currency rate from foreign api server
	rate, err := w.provider.GetRate(ctx, nextTask.CurrencyPair)
	if err != nil {
		if nextTask.AttemptCount >= w.cfg.MaxAttempts {
			// Worker sets task status to FAILED
			processed, err := w.repo.MarkFailed(ctx, err.Error(), nextTask.ID)
			return processed, err
		} else {
			// Worker sets task status to PENDING and specifies the time of the next attempt
			processed, err := w.repo.ScheduleRetry(ctx, err.Error(), time.Now().Add(w.cfg.PollInterval), nextTask.ID)
			return processed, err
		}
	}
	
	// Worker sets task status to SUCCESS
	processed, err := w.repo.MarkSucceeded(ctx, rate, nextTask.ID)
	if err != nil {
		return processed, err
	}

	return true, nil
}

func (w *Worker) processAvailable(ctx context.Context) error {
	var err error
	
	_, err = w.repo.RequeueExpired(ctx)
	if err != nil {
		return err
	}

	processed := true
	for processed {
		processed, err = w.ProcessOnce(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
