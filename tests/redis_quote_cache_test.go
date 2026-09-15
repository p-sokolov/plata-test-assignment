package tests

import (
	"context"
	"testing"
	"time"

	"plata-test-assignment/internal/cache/quotes"
	"plata-test-assignment/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisQuoteCacheLifecycle(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := quotes.New(client, time.Minute)
	want := models.LatestQuote{
		CurrencyPair: "EUR/MXN",
		Rate:         21.42,
		UpdatedAt:    time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC),
	}

	if err := cache.SetLatest(context.Background(), want); err != nil {
		t.Fatalf("SetLatest() error = %v", err)
	}
	got, err := cache.GetLatest(context.Background(), "EUR/MXN")
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if got == nil || *got != want {
		t.Errorf("GetLatest() = %+v, want %+v", got, want)
	}

	if err := cache.DeleteLatest(context.Background(), "EUR/MXN"); err != nil {
		t.Fatalf("DeleteLatest() error = %v", err)
	}
	got, err = cache.GetLatest(context.Background(), "EUR/MXN")
	if err != nil {
		t.Fatalf("GetLatest() after delete error = %v", err)
	}
	if got != nil {
		t.Errorf("GetLatest() after delete = %+v, want nil", got)
	}
}
