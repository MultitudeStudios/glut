package auth

import (
	"context"
	"errors"
	"glut/common/flux"
	"glut/common/sqlutil"
	"glut/common/valid"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

const (
	defaultSessionQueryLimit = 20
	maxSessionQueryLimit     = 100
)

type Session struct {
	ID        string
	Token     string
	UserID    string
	UserIP    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionQuery struct {
	ID             string
	UserID         string
	Limit          int
	Offset         int
	IncludeExpired bool
}

type Credentials struct {
	Username string
	Password string
}

type ClearSessionInput struct {
	IDs    []string
	UserID string
}

func (s *Service) Sessions(f *flux.Flow, sq *SessionQuery) ([]Session, error) {
	var errs valid.Errors
	if sq.ID != "" && !valid.IsUUID(sq.ID) {
		errs = append(errs, valid.Error{Field: "id", Error: "Invalid id."})
	}
	if sq.UserID != "" && !valid.IsUUID(sq.UserID) {
		errs = append(errs, valid.Error{Field: "user_id", Error: "Invalid id."})
	}
	if len(errs) != 0 {
		return nil, errs
	}

	if sq.Limit <= 0 || sq.Limit > maxSessionQueryLimit {
		sq.Limit = defaultSessionQueryLimit
	}
	if sq.Offset < 0 {
		sq.Offset = 0
	}

	sessions, err := querySessions(f, s.db, sq)
	if err != nil {
		return nil, err
	}
	if sq.ID != "" && len(sessions) == 0 {
		return nil, ErrSessionNotFound
	}
	return sessions, nil
}

func (s *Service) CreateSession(f *flux.Flow, creds *Credentials) (Session, error) {
	var errs valid.Errors
	if creds.Username == "" {
		errs = append(errs, valid.Error{Field: "username", Error: "Required."})
	}
	if creds.Password == "" {
		errs = append(errs, valid.Error{Field: "password", Error: "Required."})
	}
	if len(errs) != 0 {
		return Session{}, errs
	}

	ctx := f.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)

	user, err := userForAuth(ctx, tx, creds.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}

	if err := validatePassword(user.Password, creds.Password, s.cfg.PasswordChecker); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}

	sess, err := newSession(f, user.ID, s.cfg.SessionTokenLength, s.cfg.SessionTokenDuration)
	if err != nil {
		return Session{}, err
	}
	if err := saveSession(ctx, tx, sess); err != nil {
		return Session{}, err
	}
	if err := updateUserLastLogin(f, tx, user.ID); err != nil {
		return Session{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return sess, nil

}

func (s *Service) ClearSessions(f *flux.Flow, in *ClearSessionInput) (int, error) {
	var errs valid.Errors
	if len(in.IDs) == 0 && in.UserID == "" {
		errs = append(errs, valid.Error{Error: "Input cannot be empty."})
	}
	if !valid.IsUUIDSlice(in.IDs) {
		errs = append(errs, valid.Error{Field: "ids", Error: "Contains invalid id."})
	}
	if in.UserID != "" && !valid.IsUUID(in.UserID) {
		errs = append(errs, valid.Error{Field: "user_id", Error: "Invalid id."})
	}
	if len(errs) != 0 {
		return 0, errs
	}

	count, err := deleteSessions(f.Context(), s.db, in)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func querySessions(f *flux.Flow, db sqlutil.DB, sq *SessionQuery) ([]Session, error) {
	ctx := f.Context()
	now := f.Start()

	q := psql.Select(
		sm.Columns(
			"id",
			"token",
			"user_id",
			"user_ip",
			"created_at",
			"expires_at",
		),
		sm.From("auth.sessions"),
	)

	if sq.ID != "" {
		q.Apply(
			sm.Where(psql.Quote("id").EQ(psql.Arg(sq.ID))),
		)
	}
	if sq.UserID != "" {
		q.Apply(
			sm.Where(psql.Quote("user_id").EQ(psql.Arg(sq.UserID))),
		)
	}
	if !sq.IncludeExpired {
		q.Apply(
			sm.Where(psql.Quote("expires_at").GT(psql.Arg(now))),
		)
	}
	if sq.Limit != 0 {
		q.Apply(
			sm.Limit(sq.Limit),
		)
	}
	if sq.Offset != 0 {
		q.Apply(
			sm.Offset(sq.Offset),
		)
	}

	sql, args := q.MustBuild()
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	sessions, err := scanSessions(rows)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func saveSession(ctx context.Context, db sqlutil.DB, sess Session) error {
	q := psql.Insert(
		im.Into("auth.sessions",
			"id", "token", "user_id", "user_ip", "created_at", "expires_at",
		),
		im.Values(psql.Arg(sess.ID, sess.Token, sess.UserID, sess.UserIP, sess.CreatedAt, sess.ExpiresAt)),
	)

	sql, args := q.MustBuild()
	if _, err := db.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return nil
}

func deleteSessions(ctx context.Context, db sqlutil.DB, in *ClearSessionInput) (int, error) {
	q := psql.Delete(dm.From("auth.sessions"))
	if in.IDs != nil {
		q.Apply(dm.Where(
			psql.Quote("id").In(
				psql.Arg(sqlutil.InSlice(in.IDs)...)),
		))
	}
	if in.UserID != "" {
		q.Apply(dm.Where(
			psql.Quote("user_id").EQ(psql.Arg(in.UserID)),
		))
	}

	sql, args := q.MustBuild()
	res, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return int(res.RowsAffected()), nil
}

func userForAuth(ctx context.Context, db sqlutil.DB, username string) (User, error) {
	q := psql.Select(
		sm.From("auth.users"),
		sm.Columns("id", "password_hash"),
		sm.Where(psql.Quote("username").EQ(psql.Arg(username))),
		sm.ForNoKeyUpdate(),
	)

	sql, args := q.MustBuild()
	var userID string
	var passwordHash string
	if err := db.QueryRow(ctx, sql, args...).Scan(
		&userID,
		&passwordHash,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	user := User{
		ID:       userID,
		Password: passwordHash,
	}
	return user, nil
}

func updateUserLastLogin(f *flux.Flow, db sqlutil.DB, userID string) error {
	q := psql.Update(
		um.Table("auth.users"),
		um.Set("last_login_at").ToArg(f.Start()),
		um.Set("last_login_ip").ToArg(f.IP()),
		um.Where(psql.Quote("id").EQ(psql.Arg(userID))),
	)

	sql, args := q.MustBuild()
	if _, err := db.Exec(f.Context(), sql, args...); err != nil {
		return err
	}
	return nil
}

func newSession(f *flux.Flow, userID string, tokenLength int, duration time.Duration) (Session, error) {
	token, err := generateToken(tokenLength)
	if err != nil {
		return Session{}, err
	}
	sess := Session{
		ID:        uuid.New().String(),
		Token:     token,
		UserID:    userID,
		UserIP:    f.IP(),
		CreatedAt: f.Start(),
		ExpiresAt: f.Start().Add(duration),
	}
	return sess, nil
}

func scanSessions(rows pgx.Rows) ([]Session, error) {
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var id string
		var token string
		var userID string
		var userIP string
		var createdAt time.Time
		var expiresAt time.Time

		if err := rows.Scan(
			&id,
			&token,
			&userID,
			&userIP,
			&createdAt,
			&expiresAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, Session{
			ID:        id,
			Token:     token,
			UserID:    userID,
			UserIP:    userIP,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
		})
	}
	return sessions, nil
}
