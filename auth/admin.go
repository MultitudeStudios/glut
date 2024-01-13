package auth

import (
	"errors"
	"glut/common/flux"
	"glut/common/valid"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

type ChangePasswordInput struct {
	OldPassword string
	NewPassword string
}

type ChangeEmailInput struct {
	Token    string
	Email    string
	Password string
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

func (s *Service) ChangeEmail(f *flux.Flow, in *ChangeEmailInput) error {
	if in.Token != "" {
		tx, err := s.db.Begin(f.Ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(f.Ctx)

		token, err := getToken(f, tx, in.Token, tokenKindChangeEmail)
		if err != nil {
			return err
		}
		newEmail := token.Get(tokenMetaNewEmail)
		if newEmail == "" {
			return ErrInvalidToken
		}

		sql, args := psql.Select(
			sm.From("auth.users"),
			sm.Columns("email"),
			sm.Where(psql.Quote("id").EQ(psql.Arg(token.UserID))),
			sm.ForNoKeyUpdate(),
		).MustBuild()

		var email string
		if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&email); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUserNotFound
			}
			return err
		}

		sql, args = psql.Update(
			um.Table("auth.users"),
			um.Set("email").ToArg(newEmail),
			um.Where(psql.Quote("id").EQ(psql.Arg(token.UserID))),
		).MustBuild()

		if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
			return err
		}
		if err := deleteToken(f, tx, token.ID); err != nil {
			return err
		}

		// TODO: send notification to previous email

		if err := tx.Commit(f.Ctx); err != nil {
			return err
		}
		return nil
	}

	// No token provided; initiate email change process.
	if f.Session == nil {
		return ErrUnauthorized
	}
	var errs valid.Errors
	if in.Email == "" {
		errs = append(errs, valid.Error{Field: "email", Error: "Required."})
	}
	if in.Password == "" {
		errs = append(errs, valid.Error{Field: "password", Error: "Required."})
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
		sm.Columns("email", "password_hash"),
		sm.Where(psql.Quote("id").EQ(psql.Arg(f.Session.User))),
		sm.ForNoKeyUpdate(),
	).MustBuild()

	var email string
	var passwordHash string
	if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&email, &passwordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if err := validatePassword(passwordHash, in.Password, s.cfg.PasswordChecker); err != nil {
		return err
	}

	token := Token{
		ID:        mustGenerateToken(s.cfg.TokenLength),
		UserID:    f.Session.User,
		Kind:      tokenKindChangeEmail,
		CreatedAt: f.Time,
		ExpiresAt: f.Time.Add(s.cfg.ChangeEmailTokenDuration),
		Meta: map[string]*string{
			tokenMetaNewEmail: &in.Email,
		},
	}
	if err := saveToken(f, tx, token); err != nil {
		return err
	}

	// TODO: send email with token

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

		token, err := getToken(f, tx, in.Token, tokenKindVerifyUser)
		if err != nil {
			return err
		}

		sql, args := psql.Update(
			um.Table("auth.users"),
			um.Set("verified_at").ToArg(f.Time),
			um.Where(psql.Quote("id").EQ(psql.Arg(token.UserID))),
		).MustBuild()

		if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
			return err
		}
		if err := deleteToken(f, tx, token.ID); err != nil {
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
		sm.ForNoKeyUpdate(),
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
	if !tokenCreatedAt.IsZero() && time.Since(tokenCreatedAt) < s.cfg.VerificationTokenWaitTime {
		return ErrTryLater
	}

	token := Token{
		ID:        mustGenerateToken(s.cfg.TokenLength),
		UserID:    f.Session.User,
		Kind:      tokenKindVerifyUser,
		CreatedAt: f.Time,
		ExpiresAt: f.Time.Add(s.cfg.VerificationTokenDuration),
	}
	if err := saveToken(f, tx, token); err != nil {
		return err
	}

	// TODO: send email with verification token

	if err := tx.Commit(f.Ctx); err != nil {
		return err
	}
	return nil
}
