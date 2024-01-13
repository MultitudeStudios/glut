package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"glut/common/flux"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

const (
	tokenKindVerifyUser  = "verify_user"
	tokenKindChangeEmail = "change_email"
	tokenMetaNewEmail    = "new_email"
)

type Token struct {
	ID        string
	UserID    string
	Kind      string
	CreatedAt time.Time
	ExpiresAt time.Time
	Meta      map[string]*string
}

func (t *Token) Get(key string) string {
	v, ok := t.Meta[key]
	if !ok || v == nil {
		return ""
	}
	return *v
}

func getToken(f *flux.Flow, tx pgx.Tx, id, kind string) (Token, error) {
	sql, args := psql.Select(
		sm.From("auth.tokens"),
		sm.Columns("user_id", "meta", "created_at", "expires_at"),
		psql.WhereAnd(
			sm.Where(psql.Quote("id").EQ(psql.Arg(id))),
			sm.Where(psql.Quote("kind").EQ(psql.Arg(kind))),
			sm.Where(psql.Quote("expires_at").GT(psql.Arg(f.Time))),
		),
		sm.ForNoKeyUpdate(),
	).MustBuild()

	var userID string
	var meta pgtype.Hstore
	var createdAt time.Time
	var expiresAt time.Time
	if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&userID, &meta, &createdAt, &expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Token{}, ErrInvalidToken
		}
		return Token{}, fmt.Errorf("auth.getToken: %w", err)
	}

	token := Token{
		ID:        id,
		UserID:    userID,
		Kind:      kind,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		Meta:      meta,
	}
	return token, nil
}

func saveToken(f *flux.Flow, tx pgx.Tx, token Token) error {
	sql, args := psql.Insert(
		im.Into("auth.tokens",
			"id", "user_id", "kind", "created_at", "expires_at", "meta",
		),
		im.Values(psql.Arg(token.ID, token.UserID, token.Kind, token.CreatedAt, token.ExpiresAt, token.Meta)),
		im.OnConflict("user_id", "kind").DoUpdate().SetExcluded("id", "created_at", "expires_at", "meta"),
	).MustBuild()

	if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
		return fmt.Errorf("auth.saveToken: %w", err)
	}
	return nil
}

func deleteToken(f *flux.Flow, tx pgx.Tx, id string) error {
	sql, args := psql.Delete(
		dm.From("auth.tokens"),
		dm.Where(psql.Quote("id").EQ(psql.Arg(id))),
	).MustBuild()

	if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
		return fmt.Errorf("auth.deleteToken: %w", err)
	}
	return nil
}

const tokenChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generateToken: %w", err)
	}
	for i, b := range bytes {
		bytes[i] = tokenChars[b%byte(len(tokenChars))]
	}
	return string(bytes), nil
}

func mustGenerateToken(length int) string {
	token, err := generateToken(length)
	if err != nil {
		panic(err)
	}
	return token
}
