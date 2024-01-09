package auth

import (
	"errors"
	"glut/common/flux"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// NewAuthenticator...
func NewAuthenticator(db *pgxpool.Pool) flux.Authenticator {
	return func(f *flux.Flow, token string) (*flux.Session, error) {
		ctx := f.Context()
		now := f.Start()

		q := psql.Select(
			sm.From("auth.sessions"),
			sm.Columns("id", "user_id"),
			psql.WhereAnd(
				sm.Where(psql.Quote("token").EQ(psql.Arg(token))),
				sm.Where(psql.Quote("expires_at").GT(psql.Arg(now))),
			),
		)

		sql, args := q.MustBuild()
		var id string
		var userID string
		if err := db.QueryRow(ctx, sql, args...).Scan(&id, &userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, flux.ErrInvalidAuthToken
			}
			return nil, err
		}

		res := &flux.Session{
			ID:   id,
			User: userID,
		}
		return res, nil
	}
}
