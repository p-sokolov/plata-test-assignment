package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"plata-test-assignment/internal/client/fx"
	"plata-test-assignment/internal/errorz"
	custommiddleware "plata-test-assignment/internal/transport/http/middleware"
	v1 "plata-test-assignment/internal/transport/http/v1"
	worker "plata-test-assignment/internal/worker/quotes"
	swaggerui "plata-test-assignment/pkg/swagger-ui"

	"plata-test-assignment/env"
	"plata-test-assignment/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	repository "plata-test-assignment/internal/repository/quotes"
	service "plata-test-assignment/internal/service/quotes"
	handler "plata-test-assignment/internal/transport/http/v1/quotes"
)

type App struct {
	cfg    *config.Config
	l      *slog.Logger
	e      *echo.Echo
	dbPool *pgxpool.Pool
	w      *worker.Worker
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates and initializes a new instance of App.
func New(ctx context.Context, cfg *config.Config, l *slog.Logger) (*App, error) {
	a := &App{
		cfg: cfg,
		l:   l,
	}

	if err := a.initDB(ctx); err != nil {
		return nil, err
	}

	if err := a.migrateDB(); err != nil {
		return nil, err
	}

	if err := a.initEcho(); err != nil {
		return nil, err
	}

	repo := repository.New(a.dbPool)
	service := service.New(repo)
	handler := handler.New(service)

	// create foreign exchange instance
	fxClient := fx.New(
		cfg.FX.BaseURL,
		cfg.FX.APIKey,
		&http.Client{Timeout: cfg.FX.Timeout},
	)

	// create worker
	workerCfg := worker.Config{
		PollInterval:   a.cfg.Worker.Interval,
		LeaseDuration:  a.cfg.Worker.Duration,
		MaxAttempts:    a.cfg.Worker.Attempts,
		RetryBaseDelay: a.cfg.Worker.BackoffBaseDelay,
		RetryMaxDelay:  a.cfg.Worker.BackoffMaxDelay,
	}
	quoteWorker, err := worker.New(repo, fxClient, workerCfg, a.l)
	if err != nil {
		a.dbPool.Close()
		return nil, fmt.Errorf("create quote worker: %w", err)
	}
	a.w = quoteWorker

	apiGroup := a.e.Group("/api/v1")

	strictHandler := v1.NewStrictHandler(
		handler,
		[]v1.StrictMiddlewareFunc{
			custommiddleware.StrictErrorMiddleware,
		},
	)

	v1.RegisterHandlers(apiGroup, strictHandler)

	return a, nil
}

// Start performs a start of all functional services.
func (a *App) Start(ctx context.Context) error {
	a.l.Info("Starting...")

	workerCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.w.Run(workerCtx)
	}()

	if err := a.e.Start(a.cfg.HttpSrv.Addr); err != nil {
		return err
	}
	return nil
}

// Stop performs a graceful shutdown for all components.
func (a *App) Stop(ctx context.Context) error {
	a.l.Info("[!] Shutting down...")

	var stopErr error

	a.l.Info("Stopping http server...")
	if err := a.e.Shutdown(ctx); err != nil {
		stopErr = errors.Join(stopErr, fmt.Errorf("failed to shutdown http server: %w", err))
	}

	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()

	a.l.Info("Closing database pool...")
	a.dbPool.Close()

	if stopErr != nil {
		return stopErr
	}

	a.l.Info("Stopped gracefully. Have a good day!")
	return nil
}

// initDB initializes a new pool for PostgreSQL db.
func initDB(ctx context.Context, dbURL string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// initDB sets up PostgreSQL db.
func (a *App) initDB(ctx context.Context) error {
	dbPool, err := initDB(ctx, a.cfg.Postgres.URL, a.cfg.Postgres.MaxConns)
	if err != nil {
		return fmt.Errorf("failed to init db connection: %w", err)
	}
	a.dbPool = dbPool
	return nil
}

// migrateDB performs a migration to ensure the schema is up to date.
func (a *App) migrateDB() error {
	conn := sql.OpenDB(stdlib.GetConnector(*a.dbPool.Config().ConnConfig))
	defer conn.Close()

	return env.Migrate(conn)
}

// initEcho sets up a new Echo instance with logger.
func (a *App) initEcho() error {
	a.e = echo.New()
	a.e.HideBanner = true
	a.e.HidePort = true
	a.e.Pre(middleware.RemoveTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().URL.Path, "/api/v1/swagger")
		},
	}))
	a.e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		LogRemoteIP: true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				a.l.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("ip", v.RemoteIP),
					slog.String("latency", time.Now().Sub(v.StartTime).String()),
				)
			} else {
				a.l.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("ip", v.RemoteIP),
					slog.String("latency", time.Now().Sub(v.StartTime).String()),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	}))
	a.e.Use(middleware.Recover())

	a.e.GET("/api/v1/openapi.json", func(c echo.Context) error {
		spec, err := v1.GetSwagger()
		if err != nil {
			slog.Error("failed to get swagger spec", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, errorz.ErrInternalServerError.Error())
		}
		return c.JSON(http.StatusOK, spec)
	})

	swaggerUIHandler, err := swaggerui.Handler()
	if err != nil {
		return fmt.Errorf("failed to get swagger ui handler: %w", err)
	}

	uiHandler := http.StripPrefix("/api/v1/swagger", swaggerUIHandler)
	a.e.GET("/api/v1/swagger/*", echo.WrapHandler(uiHandler))
	a.e.GET("/api/v1/swagger", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/api/v1/swagger/")
	})

	return nil
}
