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
	maxSessionsPerUser       = 10
)

type Session struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	UserIP    string    `json:"user_ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SessionQuery struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
	IncludeExpired bool   `json:"include_expired"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LogoutInput struct {
	IDs []string `json:"ids"`
}

type ClearSessionInput struct {
	IDs    []string `json:"ids"`
	UserID string   `json:"user_id"`
}

func (s *Service) Sessions(f *flux.Flow, in SessionQuery) ([]Session, error) {
	var errs valid.Errors
	if in.ID != "" && !valid.IsUUID(in.ID) {
		errs = append(errs, valid.Error{Field: "id", Error: "Invalid id."})
	}
	if in.UserID != "" && !valid.IsUUID(in.UserID) {
		errs = append(errs, valid.Error{Field: "user_id", Error: "Invalid id."})
	}
	if len(errs) != 0 {
		return nil, errs
	}

	if in.Limit <= 0 || in.Limit > maxSessionQueryLimit {
		in.Limit = defaultSessionQueryLimit
	}
	if in.Offset < 0 {
		in.Offset = 0
	}

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
	if in.ID != "" {
		q.Apply(
			sm.Where(psql.Quote("id").EQ(psql.Arg(in.ID))),
		)
	}
	if in.UserID != "" {
		q.Apply(
			sm.Where(psql.Quote("user_id").EQ(psql.Arg(in.UserID))),
		)
	}
	if !in.IncludeExpired {
		q.Apply(
			sm.Where(psql.Quote("expires_at").GT(psql.Arg(f.Time))),
		)
	}
	if in.Limit != 0 {
		q.Apply(
			sm.Limit(in.Limit),
		)
	}
	if in.Offset != 0 {
		q.Apply(
			sm.Offset(in.Offset),
		)
	}
	sql, args := q.MustBuild()

	rows, err := s.db.Query(f.Ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []Session{}
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
	if in.ID != "" && len(sessions) == 0 {
		return nil, ErrSessionNotFound
	}
	return sessions, nil
}

func (s *Service) CreateSession(f *flux.Flow, in Credentials) (Session, error) {
	var errs valid.Errors
	if in.Username == "" {
		errs = append(errs, valid.Error{Field: "username", Error: "Required."})
	}
	if in.Password == "" {
		errs = append(errs, valid.Error{Field: "password", Error: "Required."})
	}
	if len(errs) != 0 {
		return Session{}, errs
	}

	tx, err := s.db.Begin(f.Ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(f.Ctx)

	sql, args := psql.Select(
		sm.From("auth.users"),
		sm.Columns("id", "password_hash"),
		sm.Where(psql.Quote("username").EQ(psql.Arg(in.Username))),
		sm.ForNoKeyUpdate(),
	).MustBuild()

	var userID string
	var passwordHash string
	if err := tx.QueryRow(f.Ctx, sql, args...).Scan(&userID, &passwordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}

	if err := validatePassword(passwordHash, in.Password, s.cfg.PasswordChecker); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}

	sql, args = psql.Select(
		sm.From("auth.bans"),
		psql.WhereAnd(
			sm.Where(psql.Quote("user_id").EQ(psql.Arg(userID))),
			psql.WhereOr(
				sm.Where(psql.Quote("unbanned_at").GT(psql.Arg(f.Time))),
				sm.Where(psql.Quote("banned_at").EQ(psql.Quote("unbanned_at"))),
			),
		),
	).MustBuild()

	var isBanned bool
	if err := tx.QueryRow(f.Ctx, sqlutil.Exists(sql), args...).Scan(&isBanned); err != nil {
		return Session{}, err
	}
	if isBanned {
		return Session{}, ErrUserBanned
	}

	q := `SELECT COUNT(id) FROM auth.sessions WHERE user_id = $1;`

	var sessCount int
	if err := tx.QueryRow(f.Ctx, q, userID).Scan(&sessCount); err != nil {
		return Session{}, err
	}
	if sessCount >= maxSessionsPerUser {
		return Session{}, ErrSessionLimit
	}

	q = `
	SELECT session_number FROM generate_series (1,$1) session_number
	EXCEPT (
		SELECT session_number FROM auth.sessions WHERE user_id = $2
	)
	ORDER BY 1
	LIMIT 1;`

	var nextSessNum int
	if err := tx.QueryRow(f.Ctx, q, maxSessionsPerUser, userID).Scan(&nextSessNum); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrSessionLimit
		}
		return Session{}, err
	}

	sess := Session{
		ID:        uuid.New().String(),
		Token:     mustGenerateToken(s.cfg.TokenLength),
		UserID:    userID,
		UserIP:    f.IP,
		CreatedAt: f.Time,
		ExpiresAt: f.Time.Add(s.cfg.SessionTokenDuration),
	}

	sql, args = psql.Insert(
		im.Into("auth.sessions",
			"id", "token", "user_id", "user_ip", "session_number", "created_at", "expires_at",
		),
		im.Values(psql.Arg(sess.ID, sess.Token, sess.UserID, sess.UserIP, nextSessNum, sess.CreatedAt, sess.ExpiresAt)),
	).MustBuild()

	if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
		return Session{}, err
	}

	sql, args = psql.Update(
		um.Table("auth.users"),
		um.Set("last_login_at").ToArg(f.Time),
		um.Set("last_login_ip").ToArg(f.IP),
		um.Where(psql.Quote("id").EQ(psql.Arg(userID))),
	).MustBuild()

	if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
		return Session{}, err
	}
	if err := tx.Commit(f.Ctx); err != nil {
		return Session{}, err
	}
	return sess, nil
}

func (s *Service) ClearSessions(f *flux.Flow, in ClearSessionInput) (int, error) {
	var errs valid.Errors
	if len(in.IDs) == 0 && in.UserID == "" {
		errs = append(errs, valid.Error{Error: "Input required."})
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

	q := psql.Delete(dm.From("auth.sessions"))
	if in.IDs != nil {
		q.Apply(dm.Where(
			psql.Quote("id").In(
				psql.Arg(sqlutil.AnySlice(in.IDs)...)),
		))
	}
	if in.UserID != "" {
		q.Apply(dm.Where(
			psql.Quote("user_id").EQ(psql.Arg(in.UserID)),
		))
	}

	sql, args := q.MustBuild()
	res, err := s.db.Exec(f.Ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return int(res.RowsAffected()), nil
}

func (s *Service) RenewSession(f *flux.Flow) (time.Time, error) {
	newExpiry := f.Time.Add(s.cfg.SessionTokenDuration)

	sql, args := psql.Update(
		um.Table("auth.sessions"),
		um.Set("expires_at").ToArg(newExpiry),
		um.Where(psql.Quote("id").EQ(psql.Arg(f.Session.ID))),
	).MustBuild()

	res, err := s.db.Exec(f.Ctx, sql, args...)
	if err != nil {
		return time.Time{}, err
	}
	if res.RowsAffected() == 0 {
		return time.Time{}, ErrSessionNotFound
	}
	return newExpiry, nil
}

func deleteUserSessions(ctx context.Context, tx pgx.Tx, userID string) error {
	sql, args := psql.Delete(
		dm.From("auth.sessions"),
		dm.Where(psql.Quote("user_id").EQ(psql.Arg(userID))),
	).MustBuild()

	_, err := tx.Exec(ctx, sql, args...)
	return err
}
