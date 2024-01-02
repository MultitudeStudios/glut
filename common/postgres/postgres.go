package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultURL               = "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	defaultMinOpenConns      = 5
	defaultMaxOpenConns      = 10
	defaultMaxConnLifetime   = 10 * time.Minute
	defaultMaxConnIdleTime   = 5 * time.Minute
	defaultHealthCheckPeriod = 5 * time.Minute
)

type Config struct {
	URL               string
	MinOpenConns      int
	MaxOpenConns      int
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

func New(ctx context.Context, cfg *Config) (*pgxpool.Pool, error) {
	if cfg.URL == "" {
		cfg.URL = defaultURL
	}
	if cfg.MinOpenConns == 0 {
		cfg.MinOpenConns = defaultMinOpenConns
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = defaultMaxOpenConns
	}
	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = defaultMaxConnLifetime
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = defaultMaxConnIdleTime
	}
	if cfg.HealthCheckPeriod == 0 {
		cfg.HealthCheckPeriod = defaultHealthCheckPeriod
	}

	poolConf, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, err
	}
	poolConf.MinConns = int32(cfg.MinOpenConns)
	poolConf.MaxConns = int32(cfg.MaxOpenConns)
	poolConf.MaxConnLifetime = cfg.MaxConnLifetime
	poolConf.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConf.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConf)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
