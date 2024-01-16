package auth

import (
	"errors"
	"fmt"
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

type ResetPasswordInput struct {
	Token    string
	Username string
	Password string
}

type ForgotUsernameInput struct {
	Email string
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

	if err := updateUserPassword(f.Ctx, tx, f.Session.User, in.NewPassword); err != nil {
		return err
	}
	if err := deleteUserSessions(f.Ctx, tx, f.Session.User); err != nil {
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
		if err := deleteToken(f.Ctx, tx, token.ID); err != nil {
			return err
		}
		if err := deleteUserSessions(f.Ctx, tx, token.UserID); err != nil {
			return err
		}

		// TODO: send notification to previous email

		if err := tx.Commit(f.Ctx); err != nil {
			return err
		}
		return nil
	}

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
		if err := deleteToken(f.Ctx, tx, token.ID); err != nil {
			return err
		}
		if err := tx.Commit(f.Ctx); err != nil {
			return err
		}
		return nil
	}

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

	token := s.createUserVerificationToken(f.Session.User, f.Time)
	if err := saveToken(f, tx, token); err != nil {
		return err
	}

	// TODO: send email with token

	if err := tx.Commit(f.Ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) ResetPassword(f *flux.Flow, in *ResetPasswordInput) error {
	if in.Token != "" {
		var errs valid.Errors
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

		token, err := getToken(f, tx, in.Token, tokenKindResetPassword)
		if err != nil {
			return err
		}

		sql, args := psql.Select(
			sm.From("auth.users"),
			sm.Columns("id"),
			sm.Where(psql.Quote("id").EQ(psql.Arg(token.UserID))),
			sm.ForNoKeyUpdate(),
		).MustBuild()

		var userExists bool
		if err := tx.QueryRow(f.Ctx, fmt.Sprintf("SELECT EXISTS (%s)", sql), args...).Scan(&userExists); err != nil {
			return err
		}
		if !userExists {
			return ErrUserNotFound
		}
		if err := updateUserPassword(f.Ctx, tx, token.UserID, in.Password); err != nil {
			return err
		}
		if err := deleteUserSessions(f.Ctx, tx, token.UserID); err != nil {
			return err
		}
		if err := deleteToken(f.Ctx, tx, token.ID); err != nil {
			return err
		}
		if err := tx.Commit(f.Ctx); err != nil {
			return err
		}
		return nil
	}

	var errs valid.Errors
	if in.Username == "" {
		errs = append(errs, valid.Error{Field: "username", Error: "Required."})
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
		sm.Columns("id", "email"),
		sm.Where(psql.Quote("username").EQ(psql.Arg(in.Username))),
	).MustBuild()

	var userID string
	var email string
	if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&userID, &email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	token := Token{
		ID:        mustGenerateToken(s.cfg.TokenLength),
		UserID:    userID,
		Kind:      tokenKindResetPassword,
		CreatedAt: f.Time,
		ExpiresAt: f.Time.Add(s.cfg.ResetPasswordTokenDuration),
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

// ForgotUsername...
func (s *Service) ForgotUsername(f *flux.Flow, in *ForgotUsernameInput) error {
	var errs valid.Errors
	if in.Email == "" {
		errs = append(errs, valid.Error{Field: "email", Error: "Required."})
	}
	if len(errs) != 0 {
		return errs
	}

	sql, args := psql.Select(
		sm.From("auth.users"),
		sm.Columns("username"),
		sm.Where(psql.Quote("email").EQ(psql.Arg(in.Email))),
	).MustBuild()

	rows, err := s.db.Query(f.Ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return err
		}
		usernames = append(usernames, username)
	}

	if len(usernames) != 0 {
		// TODO: send email with usernames
	}
	return nil
}
