package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plata-test-assignment/internal/client/fx"
)

func TestFXClientGetRate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if got, want := r.URL.Query().Get("access_key"), "test-key"; got != want {
			t.Errorf("access_key = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("from"), "EUR"; got != want {
			t.Errorf("from = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("to"), "MXN"; got != want {
			t.Errorf("to = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":21.42}`))
	}))
	defer server.Close()

	client := fx.New(server.URL, "test-key", &http.Client{Timeout: time.Second})
	rate, err := client.GetRate(context.Background(), "EUR/MXN")
	if err != nil {
		t.Fatalf("GetRate() error = %v", err)
	}
	if rate != 21.42 {
		t.Errorf("GetRate() = %v, want 21.42", rate)
	}
}

func TestFXClientGetRateReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer server.Close()

	client := fx.New(server.URL, "test-key", &http.Client{Timeout: time.Second})
	_, err := client.GetRate(context.Background(), "EUR/MXN")
	if err == nil {
		t.Fatal("GetRate() error = nil, want API error")
	}
}

func TestFXClientGetRateHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := fx.New(server.URL, "test-key", &http.Client{Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetRate(ctx, "EUR/MXN")
	if err == nil {
		t.Fatal("GetRate() error = nil, want context cancellation error")
	}
}
