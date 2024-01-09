package auth

import (
	"context"
	"crypto/rand"
	"glut/common/sqlutil"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
)

type Token struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	Meta      *TokenMeta
}

type TokenMeta map[string]*string

func (s *Service) createUserVerificationToken(ctx context.Context, db sqlutil.DB, userID string, now time.Time) error {
	token, err := newToken(userID, now, s.cfg.VerificationTokenLength, s.cfg.VerificationTokenDuration, nil)
	if err != nil {
		return err
	}
	if err := saveToken(ctx, db, token); err != nil {
		return err
	}
	return nil
}

func newToken(userID string, now time.Time, length int, duration time.Duration, meta *TokenMeta) (Token, error) {
	id, err := generateToken(length)
	if err != nil {
		return Token{}, err
	}

	token := Token{
		ID:        id,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(duration),
		Meta:      meta,
	}
	return token, nil
}

const tokenChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = tokenChars[b%byte(len(tokenChars))]
	}
	return string(bytes), nil
}

func saveToken(ctx context.Context, db sqlutil.DB, token Token) error {
	sql, args := psql.Insert(
		im.Into("auth.tokens",
			"id", "user_id", "created_at", "expires_at", "meta",
		),
		im.Values(psql.Arg(token.ID, token.UserID, token.CreatedAt, token.ExpiresAt, token.Meta)),
	).MustBuild()

	_, err := db.Exec(ctx, sql, args...)
	return err
}
