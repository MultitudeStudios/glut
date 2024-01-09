package main

import (
	"context"
	"flag"
	"glut/auth"
	authapi "glut/auth/api"
	"glut/common/flux"
	"glut/common/log"
	"glut/common/postgres"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := log.New()
	if err := run(ctx, logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	defer func() {
		// Recover from panics and log the error.
		if x := recover(); x != nil {
			logger.Error("A panic occurred.",
				slog.Any("error", x),
				slog.String("stack", string(debug.Stack())))
			panic(x)
		}
	}()

	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	if cfg.Debug {
		log.SetDebug()
		logger.Debug("Debugging enabled.")
	}

	db, err := postgres.New(ctx, &postgres.Config{
		URL: cfg.Database.URL,
	})
	if err != nil {
		return err
	}
	if err := db.Ping(ctx); err != nil {
		return err
	}
	defer db.Close()

	s := flux.NewServer(&flux.ServerOptions{
		Debug:             true,
		Logger:            logger,
		Port:              cfg.Server.Port,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ShutdownTimeout:   cfg.Server.ShutdownTimeout,
		Authenticator:     auth.NewAuthenticator(db),
	})

	authapi.Handler(s, auth.NewService(db, &auth.Config{}))

	if err := s.Start(ctx); err != nil {
		return err
	}
	defer s.Stop()
	return nil
}

func loadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg *Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

type Config struct {
	Debug    bool            `yaml:"debug"`
	Server   *ServerConfig   `yaml:"server"`
	Database *DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	Port              int           `yaml:"port"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	URL               string        `yaml:"url"`
	MinOpenConns      int           `yaml:"min_open_conns"`
	MaxOpenConns      int           `yaml:"max_open_conns"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time"`
	HealthcheckPeriod time.Duration `yaml:"healthcheck_period"`
}
