package tests

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"plata-test-assignment/internal/models"
	workerquotes "plata-test-assignment/internal/worker/quotes"

	"github.com/google/uuid"
)

type workerRepoStub struct {
	tasks []*models.QuoteUpdate

	succeeded bool
	retried   bool
	failed    bool
	lostLease bool

	successID    uuid.UUID
	successToken uuid.UUID
	retryAt      time.Time
}

func (s *workerRepoStub) ClaimNext(context.Context, time.Time) (*models.QuoteUpdate, error) {
	if len(s.tasks) == 0 {
		return nil, nil
	}
	task := s.tasks[0]
	s.tasks = s.tasks[1:]
	return task, nil
}

func (s *workerRepoStub) RequeueExpired(context.Context) (int64, error) { return 0, nil }

func (s *workerRepoStub) MarkSucceeded(_ context.Context, _ float64, id, token uuid.UUID) (bool, error) {
	s.succeeded = true
	s.successID = id
	s.successToken = token
	return !s.lostLease, nil
}

func (s *workerRepoStub) ScheduleRetry(_ context.Context, _ string, next time.Time, _, _ uuid.UUID) (bool, error) {
	s.retried = true
	s.retryAt = next
	return true, nil
}

func (s *workerRepoStub) MarkFailed(context.Context, string, uuid.UUID, uuid.UUID) (bool, error) {
	s.failed = true
	return true, nil
}

type rateProviderStub struct {
	rate float64
	err  error
}

func (s rateProviderStub) GetRate(context.Context, string) (float64, error) { return s.rate, s.err }

func newWorkerForTest(t *testing.T, repo *workerRepoStub, provider rateProviderStub, maxAttempts int32) *workerquotes.Worker {
	t.Helper()
	worker, err := workerquotes.New(repo, provider, nil, workerquotes.Config{
		PollInterval:   time.Second,
		LeaseDuration:  30 * time.Second,
		MaxAttempts:    maxAttempts,
		RetryBaseDelay: time.Second,
		RetryMaxDelay:  time.Minute,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("worker.New() error = %v", err)
	}
	return worker
}

func taskForAttempt(attempt int32) *models.QuoteUpdate {
	token := uuid.New()
	return &models.QuoteUpdate{
		ID:           uuid.New(),
		CurrencyPair: "EUR/MXN",
		AttemptCount: attempt,
		LeaseToken:   &token,
	}
}

func TestWorkerProcessOnceMarksSuccessfulQuote(t *testing.T) {
	t.Parallel()

	task := taskForAttempt(1)
	repo := &workerRepoStub{tasks: []*models.QuoteUpdate{task}}
	worker := newWorkerForTest(t, repo, rateProviderStub{rate: 21.42}, 3)

	processed, err := worker.ProcessOnce(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOnce() = (%v, %v), want (true, nil)", processed, err)
	}
	if !repo.succeeded || repo.successID != task.ID || repo.successToken != *task.LeaseToken {
		t.Error("worker must complete the claimed task with its lease token")
	}
}

func TestWorkerProcessOnceSchedulesRetryForTransientFailure(t *testing.T) {
	t.Parallel()

	repo := &workerRepoStub{tasks: []*models.QuoteUpdate{taskForAttempt(1)}}
	worker := newWorkerForTest(t, repo, rateProviderStub{err: errors.New("upstream unavailable")}, 3)
	before := time.Now()

	processed, err := worker.ProcessOnce(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOnce() = (%v, %v), want (true, nil)", processed, err)
	}
	if !repo.retried || repo.failed {
		t.Error("transient failure must schedule retry, not mark task as failed")
	}
	if repo.retryAt.Before(before) || repo.retryAt.After(before.Add(time.Second)) {
		t.Errorf("retry time = %s, want full-jitter delay in [0, 1s]", repo.retryAt)
	}
}

func TestWorkerProcessOnceFailsAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	repo := &workerRepoStub{tasks: []*models.QuoteUpdate{taskForAttempt(3)}}
	worker := newWorkerForTest(t, repo, rateProviderStub{err: errors.New("upstream unavailable")}, 3)

	processed, err := worker.ProcessOnce(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOnce() = (%v, %v), want (true, nil)", processed, err)
	}
	if !repo.failed || repo.retried {
		t.Error("last allowed attempt must mark task as failed without another retry")
	}
}

func TestWorkerProcessOnceReportsNoWork(t *testing.T) {
	t.Parallel()

	worker := newWorkerForTest(t, &workerRepoStub{}, rateProviderStub{}, 3)
	processed, err := worker.ProcessOnce(context.Background())
	if err != nil || processed {
		t.Fatalf("ProcessOnce() = (%v, %v), want (false, nil)", processed, err)
	}
}

func TestWorkerProcessOnceContinuesAfterLostLease(t *testing.T) {
	t.Parallel()

	repo := &workerRepoStub{tasks: []*models.QuoteUpdate{taskForAttempt(1)}, lostLease: true}
	worker := newWorkerForTest(t, repo, rateProviderStub{rate: 21.42}, 3)

	processed, err := worker.ProcessOnce(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOnce() = (%v, %v), want (true, nil)", processed, err)
	}
	if !repo.succeeded {
		t.Error("worker must attempt the state transition before detecting the lost lease")
	}
}
