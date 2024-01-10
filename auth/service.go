package auth

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	defaultPasswordFunc              = comparePasswords
	minSessionTokenLength            = 16
	maxSessionTokenLength            = 64
	defaultSessionTokenLength        = 24
	minSessionTokenDuration          = 5 * time.Minute
	maxSessionTokenDuration          = 24 * time.Hour
	defaultSessionTokenDuration      = 30 * time.Minute
	minVerificationTokenLength       = 16
	maxVerificationTokenLength       = 64
	defaultVerificationTokenLength   = 24
	minVerificationTokenDuration     = 30 * time.Minute
	maxVerificationTokenDuration     = 72 * time.Hour
	defaultVerificationTokenDuration = 24 * time.Hour
	verificationTokenWaitPeriod      = 5 * time.Minute
)

// Service...
type Service struct {
	cfg *Config
	db  *pgxpool.Pool
}

// Config...
type Config struct {
	// SessionTokenLength...
	SessionTokenLength int
	// SessionTokenDuration...
	SessionTokenDuration time.Duration
	// VerificationTokenLength...
	VerificationTokenLength int
	// VerificationTokenDuration...
	VerificationTokenDuration time.Duration
	// PasswordChecker...
	PasswordChecker PasswordCompareFunc
}

// NewService...
func NewService(db *pgxpool.Pool, cfg *Config) *Service {
	if cfg.PasswordChecker == nil {
		cfg.PasswordChecker = defaultPasswordFunc
	}
	if cfg.SessionTokenLength < minSessionTokenLength || cfg.SessionTokenLength > maxSessionTokenLength {
		cfg.SessionTokenLength = defaultSessionTokenLength
	}
	if cfg.SessionTokenDuration < minSessionTokenDuration || cfg.SessionTokenDuration > maxSessionTokenDuration {
		cfg.SessionTokenDuration = defaultSessionTokenDuration
	}
	if cfg.VerificationTokenLength < minVerificationTokenLength || cfg.VerificationTokenLength > maxVerificationTokenLength {
		cfg.VerificationTokenLength = defaultVerificationTokenLength
	}
	if cfg.VerificationTokenDuration < minVerificationTokenDuration || cfg.VerificationTokenDuration > maxVerificationTokenDuration {
		cfg.VerificationTokenDuration = defaultVerificationTokenDuration
	}
	return &Service{
		cfg: cfg,
		db:  db,
	}
}
