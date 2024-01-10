package auth

import (
	"crypto/rand"
	"glut/common/flux"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
)

const tokenKindVerifyUser = "verify_user"

type Token struct {
	ID        string
	UserID    string
	Kind      string
	CreatedAt time.Time
	ExpiresAt time.Time
	Meta      *TokenMeta
}

type TokenMeta map[string]*string

func (s *Service) createUserVerificationToken(f *flux.Flow, tx pgx.Tx, userID string) error {
	tokenID, err := generateToken(s.cfg.VerificationTokenLength)
	if err != nil {
		return err
	}

	token := Token{
		ID:        tokenID,
		UserID:    userID,
		Kind:      tokenKindVerifyUser,
		CreatedAt: f.Time,
		ExpiresAt: f.Time.Add(s.cfg.VerificationTokenDuration),
	}

	sql, args := psql.Insert(
		im.Into("auth.tokens",
			"id", "user_id", "kind", "created_at", "expires_at", "meta",
		),
		im.Values(psql.Arg(token.ID, token.UserID, token.Kind, token.CreatedAt, token.ExpiresAt, token.Meta)),
		im.OnConflict("user_id", "kind").DoUpdate().SetExcluded("id", "created_at", "expires_at", "meta"),
	).MustBuild()

	_, err = tx.Exec(f.Ctx, sql, args...)
	return err
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
