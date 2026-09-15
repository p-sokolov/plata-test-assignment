package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plata-test-assignment/internal/app"
	"plata-test-assignment/internal/config"
)

func main() {
	// setting up the logger
	logger := setupLogger()

	// create context to catch system calls
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// init config
	cfg, err := config.New(ctx)
	if err != nil {
		logger.Error("failed to read config", slog.Any("error", err))
		os.Exit(1)
	}

	// create app instance
	a, err := app.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to create app", slog.Any("error", err))
		os.Exit(1)
	}

	// start app with error channel
	errChan := make(chan error, 1)
	go func() {
		if err := a.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	// wait until receiving stop signal from app or client
	select {
	case err := <-errChan:
		if err != nil {
			logger.Error("server stopped unexpectedly", slog.Any("error", err))
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.Stop(shutdownCtx); err != nil {
			logger.Error("failed to stop server gracefully", slog.Any("error", err))
			os.Exit(1)
		}
	}
}

func setupLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
