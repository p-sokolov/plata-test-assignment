package main

import (
	"context"
	"errors"
	"fmt"
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
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// setting up the logger
	logger := setupLogger()

	// create context to catch system calls
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// init config
	cfg, err := config.New(ctx)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// create app instance
	a, err := app.New(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}
	shutdown := func() error {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := a.Stop(shutdownCtx); err != nil {
			return fmt.Errorf("failed to stop server gracefully: %w", err)
		}
		return nil
	}

	// start app with error channel
	errChan := make(chan error, 1)
	go func() {
		if err := a.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	// Wait for either the server to stop or an OS shutdown signal.
	select {
	case err := <-errChan:
		shutdownErr := shutdown()
		if err != nil {
			return errors.Join(
				fmt.Errorf("server stopped unexpectedly: %w", err),
				shutdownErr,
			)
		}
		if shutdownErr != nil {
			return shutdownErr
		}
		return errors.New("server stopped unexpectedly")
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		if err := shutdown(); err != nil {
			return err
		}
	}

	return nil
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
