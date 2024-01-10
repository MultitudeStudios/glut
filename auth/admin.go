package auth

import (
	"errors"
	"glut/common/flux"
	"glut/common/valid"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

type ChangePasswordInput struct {
	OldPassword string
	NewPassword string
}

type VerifyUserInput struct {
	Token string
}

func (s *Service) ChangePassword(f *flux.Flow, in *ChangePasswordInput) error {
	var errs valid.Errors
	if in.OldPassword == "" {
		errs = append(errs, valid.Error{Field: "old_password", Error: "Required."})
	}
	if in.NewPassword == "" {
		errs = append(errs, valid.Error{Field: "new_password", Error: "Required."})
	}
	if len(errs) != 0 {
		return errs
	}

	tx, err := s.db.Begin(f.Ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(f.Ctx)

	sql, args := psql.Select(
		sm.From("auth.users"),
		sm.Columns("password_hash"),
		sm.Where(psql.Quote("id").EQ(psql.Arg(f.Session.User))),
		sm.ForNoKeyUpdate(),
	).MustBuild()

	var passwordHash string
	if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&passwordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	if err := validatePassword(passwordHash, in.OldPassword, s.cfg.PasswordChecker); err != nil {
		return err
	}

	newPasswordHash, err := hashPassword(in.NewPassword)
	if err != nil {
		return err
	}

	sql, args = psql.Update(
		um.Table("auth.users"),
		um.Set("password_hash").ToArg(newPasswordHash),
		um.Where(psql.Quote("id").EQ(psql.Arg(f.Session.User))),
	).MustBuild()

	if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
		return err
	}

	if err := tx.Commit(f.Ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) VerifyUser(f *flux.Flow, in *VerifyUserInput) error {
	// Token provided; attempt to verify user using token.
	if in.Token != "" {
		tx, err := s.db.Begin(f.Ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(f.Ctx)

		sql, args := psql.Select(
			sm.From("auth.tokens"),
			sm.Columns("user_id"),
			psql.WhereAnd(
				sm.Where(psql.Quote("id").EQ(psql.Arg(in.Token))),
				sm.Where(psql.Quote("expires_at").GT(psql.Arg(f.Time))),
			),
			sm.ForNoKeyUpdate(),
		).MustBuild()

		var userID string
		if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidToken
			}
			return err
		}

		sql, args = psql.Update(
			um.Table("auth.users"),
			um.Set("verified_at").ToArg(f.Time),
			um.Where(psql.Quote("id").EQ(psql.Arg(userID))),
		).MustBuild()

		if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
			return err
		}

		sql, args = psql.Delete(
			dm.From("auth.tokens"),
			dm.Where(psql.Quote("id").EQ(psql.Arg(in.Token))),
		).MustBuild()

		if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
			return err
		}

		if err := tx.Commit(f.Ctx); err != nil {
			return err
		}
		return nil
	}

	// No token provided; initiate user verification process.
	if f.Session == nil {
		return ErrUnauthorized
	}

	tx, err := s.db.Begin(f.Ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(f.Ctx)

	sql, args := psql.Select(
		sm.From("auth.users"),
		sm.Columns("verified_at"),
		sm.Where(psql.Quote("id").EQ(psql.Arg(f.Session.User))),
	).MustBuild()

	var verifiedAt *time.Time
	if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&verifiedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if verifiedAt != nil {
		return ErrUserVerified
	}

	sql, args = psql.Select(
		sm.From("auth.tokens"),
		sm.Columns("created_at"),
		psql.WhereAnd(
			sm.Where(psql.Quote("user_id").EQ(psql.Arg(f.Session.User))),
			sm.Where(psql.Quote("kind").EQ(psql.Arg(tokenKindVerifyUser))),
		),
	).MustBuild()

	var tokenCreatedAt time.Time
	if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&tokenCreatedAt); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if !tokenCreatedAt.IsZero() && time.Since(tokenCreatedAt) < verificationTokenWaitPeriod {
		return ErrTryLater
	}
	if err := s.createUserVerificationToken(f, tx, f.Session.User); err != nil {
		return err
	}

	if err := tx.Commit(f.Ctx); err != nil {
		return err
	}
	return nil
}
