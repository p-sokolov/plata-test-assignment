package config

import (
	"context"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HttpSrv  httpServer
	Postgres postgres
}

type httpServer struct {
	Addr string `env:"SERVER_ADDR" env-default:":8080"`
}

type postgres struct {
	URL      string `env:"POSTGRES_URL" env-required:"true"`
	MaxConns int32  `env:"POSTGRES_MAX_CONNS" env-default:"100"`
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