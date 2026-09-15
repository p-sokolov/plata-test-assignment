package quotes

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
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
	MarkSucceeded(ctx context.Context, rate float64, id, leaseToken uuid.UUID) (bool, error)
	ScheduleRetry(ctx context.Context, errMsg string, nextAttemptAt time.Time, id, leaseToken uuid.UUID) (bool, error)
	MarkFailed(ctx context.Context, errMsg string, id, leaseToken uuid.UUID) (bool, error)
}

// interface for redis integration
type latestCache interface {
	DeleteLatest(ctx context.Context, pair string) error
}

type Config struct {
	PollInterval  time.Duration
	LeaseDuration time.Duration
	MaxAttempts   int32

	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

type Worker struct {
	repo     repository
	provider rateProvider
	cache    latestCache
	cfg      Config
	logger   *slog.Logger
}

func New(r repository, p rateProvider, cache latestCache, cfg Config, l *slog.Logger) (*Worker, error) {
	if cfg.PollInterval <= 0 || cfg.LeaseDuration <= 0 {
		return nil, fmt.Errorf("worker poll interval and lease duration must be positive")
	}
	if cfg.MaxAttempts < 1 {
		return nil, fmt.Errorf("worker max attempts must be at least one")
	}
	if cfg.RetryBaseDelay <= 0 || cfg.RetryMaxDelay < cfg.RetryBaseDelay {
		return nil, fmt.Errorf("worker retry delays are invalid")
	}

	return &Worker{repo: r, provider: p, cache: cache, cfg: cfg, logger: l}, nil
}

// Run processes queued updates until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		if err := w.processAvailable(ctx); err != nil && ctx.Err() == nil {
			w.logger.Error("worker iteration failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) processAvailable(ctx context.Context) error {
	if _, err := w.repo.RequeueExpired(ctx); err != nil {
		return err
	}

	for {
		processed, err := w.ProcessOnce(ctx)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}
	}
}

// ProcessOnce claims and handles one update. processed reports whether it claimed work.
func (w *Worker) ProcessOnce(ctx context.Context) (processed bool, err error) {
	nextTask, err := w.repo.ClaimNext(ctx, time.Now().Add(w.cfg.LeaseDuration))
	if err != nil {
		return false, err
	}
	if nextTask == nil {
		return false, nil
	}
	if nextTask.LeaseToken == nil {
		return true, fmt.Errorf("claimed quote update %s has no lease token", nextTask.ID)
	}

	// Get currency rate from foreign api server
	rate, err := w.provider.GetRate(ctx, nextTask.CurrencyPair)
	if err != nil {
		if nextTask.AttemptCount >= w.cfg.MaxAttempts {
			// Worker sets task status to FAILED
			applied, markErr := w.repo.MarkFailed(ctx, err.Error(), nextTask.ID, *nextTask.LeaseToken)
			return w.finish(nextTask, applied, markErr)
		}

		nextAttemptAt := time.Now().Add(w.retryDelay(nextTask.AttemptCount))
		// Worker sets task status to PENDING and specifies the time of the next attempt
		applied, retryErr := w.repo.ScheduleRetry(ctx, err.Error(), nextAttemptAt, nextTask.ID, *nextTask.LeaseToken)
		return w.finish(nextTask, applied, retryErr)
	}

	// Worker sets task status to SUCCESS
	applied, markErr := w.repo.MarkSucceeded(ctx, rate, nextTask.ID, *nextTask.LeaseToken)
	if markErr == nil && applied && w.cache != nil {
		if err := w.cache.DeleteLatest(ctx, nextTask.CurrencyPair); err != nil {
			w.logger.Warn("failed to invalidate latest quote cache", "pair", nextTask.CurrencyPair, "error", err)
		}
	}
	return w.finish(nextTask, applied, markErr)
}

// finish keeps draining the queue after a task was claimed, even if its lease was lost.
func (w *Worker) finish(task *models.QuoteUpdate, applied bool, err error) (bool, error) {
	if err != nil {
		// If the task had already been picked up but `MarkSucceeded` returned `applied=false`
		// due to a lost lease, `ProcessOnce` should still return `true, nil`.
		// Otherwise, `processAvailable` would stop, even though there might be other tasks in the queue
		return true, err
	}
	if !applied {
		w.logger.Warn("quote update lease was lost before state transition", "update_id", task.ID)
	}
	return true, nil
}

func (w *Worker) retryDelay(attempt int32) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	// Calculate the exponential delay based on the attempt number: BaseDelay * 2^(AttemptCount)
	// Use a bitwise shift (1 << count-1) to raise 2 to the power
	shift := uint(attempt - 1)
	if shift >= 63 {
		return time.Duration(rand.Int63n(int64(w.cfg.RetryMaxDelay)))
	}
	backoff := w.cfg.RetryBaseDelay * time.Duration(1<<shift)
	if backoff <= 0 || backoff > w.cfg.RetryMaxDelay {
		backoff = w.cfg.RetryMaxDelay
	}

	return time.Duration(rand.Int63n(int64(backoff)))
}
