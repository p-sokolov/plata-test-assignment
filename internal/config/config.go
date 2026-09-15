package config

import (
	"context"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HttpSrv  httpServer
	Postgres postgres
	FX       fxConfig
	Worker   workerConfig
}

type httpServer struct {
	Addr string `env:"SERVER_ADDR" env-default:":8080"`
}

type postgres struct {
	URL      string `env:"POSTGRES_URL" env-required:"true"`
	MaxConns int32  `env:"POSTGRES_MAX_CONNS" env-default:"10"`
}

type fxConfig struct {
	BaseURL string        `env:"FX_BASE_URL" env-required:"true"`
	APIKey  string        `env:"FX_ACCESS_KEY" env-required:"true"`
	Timeout time.Duration `env:"FX_TIMEOUT" env-default:"10s"`
}

type workerConfig struct {
	Interval         time.Duration `env:"WORKER_POLL_INTERVAL" env-default:"1s"`
	Duration         time.Duration `env:"WORKER_LEASE_DURATION" env-default:"30s"`
	Attempts         int32         `env:"WORKER_MAX_ATTEMPTS" env-default:"3"`
	BackoffBaseDelay time.Duration `env:"WORKER_BASE_DELAY" env-default:"1s"`
	BackoffMaxDelay  time.Duration `env:"WORKER_MAX_DELAY" env-default:"50s"`
}

// New creates a new config instance
func New(ctx context.Context) (*Config, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var cfg Config

	// Read .env file
	// If failed to read file, will try ReadEnv
	if err := cleanenv.ReadConfig(".env", &cfg); err == nil {
		return &cfg, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Read env
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
