package auth

import (
	"cmp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	defaultPasswordFunc = comparePasswords

	defaultTokenLength = 24
	minTokenLength     = 16
	maxTokenLength     = 64

	minSessionTokenDuration     = 5 * time.Minute
	maxSessionTokenDuration     = 24 * time.Hour
	defaultSessionTokenDuration = 30 * time.Minute

	minVerificationTokenDuration     = 30 * time.Minute
	maxVerificationTokenDuration     = 72 * time.Hour
	defaultVerificationTokenDuration = 24 * time.Hour

	minVerificationTokenWaitTime     = 1 * time.Minute
	maxVerificationTokenWaitTime     = 15 * time.Minute
	defaultVerificationTokenWaitTime = 5 * time.Minute

	minChangeEmailTokenDuration     = 15 * time.Minute
	maxChangeEmailTokenDuration     = 24 * time.Hour
	defaultChangeEmailTokenDuration = 3 * time.Hour

	minResetPasswordTokenDuration     = 15 * time.Minute
	maxResetPasswordTokenDuration     = 24 * time.Hour
	defaultResetPasswordTokenDuration = 3 * time.Hour
)

// Service...
type Service struct {
	cfg *Config
	db  *pgxpool.Pool
}

// Config...
type Config struct {
	// TokenLength...
	TokenLength int
	// SessionTokenDuration...
	SessionTokenDuration time.Duration
	// VerificationTokenDuration...
	VerificationTokenDuration time.Duration
	// VerificationTokenWaitTime...
	VerificationTokenWaitTime time.Duration
	// ChangeEmailTokenDuration...
	ChangeEmailTokenDuration time.Duration
	// ResetPasswordTokenDuration...
	ResetPasswordTokenDuration time.Duration
	// PasswordChecker...
	PasswordChecker PasswordCompareFunc
}

// NewService...
func NewService(db *pgxpool.Pool, cfg *Config) *Service {
	if cfg.PasswordChecker == nil {
		cfg.PasswordChecker = defaultPasswordFunc
	}
	cfg.TokenLength = minMaxValue(cfg.TokenLength, minTokenLength, maxTokenLength, defaultTokenLength)
	cfg.SessionTokenDuration = minMaxValue(cfg.SessionTokenDuration, minSessionTokenDuration, maxSessionTokenDuration, defaultSessionTokenDuration)
	cfg.VerificationTokenDuration = minMaxValue(cfg.VerificationTokenDuration, minVerificationTokenDuration, maxVerificationTokenDuration, defaultVerificationTokenDuration)
	cfg.VerificationTokenWaitTime = minMaxValue(cfg.VerificationTokenWaitTime, minVerificationTokenWaitTime, maxVerificationTokenWaitTime, defaultVerificationTokenWaitTime)
	cfg.ChangeEmailTokenDuration = minMaxValue(cfg.ChangeEmailTokenDuration, minChangeEmailTokenDuration, maxChangeEmailTokenDuration, defaultChangeEmailTokenDuration)
	cfg.ResetPasswordTokenDuration = minMaxValue(cfg.ResetPasswordTokenDuration, minResetPasswordTokenDuration, maxResetPasswordTokenDuration, defaultResetPasswordTokenDuration)

	return &Service{
		cfg: cfg,
		db:  db,
	}
}

func minMaxValue[T cmp.Ordered](val, min, max, defaultVal T) T {
	if val < min || val > max {
		return defaultVal
	}
	return val
}
