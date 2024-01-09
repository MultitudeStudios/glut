package auth

import (
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
	ctx := f.Context()
	now := f.Start()

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
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
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
	if sq.ID != "" && len(sessions) == 0 {
		return nil, ErrSessionNotFound
	}
	return sessions, nil
}

func (s *Service) CreateSession(f *flux.Flow, creds *Credentials) (Session, error) {
	ctx := f.Context()
	now := f.Start()

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

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)

	sql, args := psql.Select(
		sm.From("auth.users"),
		sm.Columns("id", "password_hash"),
		sm.Where(psql.Quote("username").EQ(psql.Arg(creds.Username))),
		sm.ForNoKeyUpdate(),
	).MustBuild()

	var userID string
	var passwordHash string
	if err := tx.QueryRow(ctx, sql, args...).Scan(&userID, &passwordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}

	if err := validatePassword(passwordHash, creds.Password, s.cfg.PasswordChecker); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}

	token, err := generateToken(s.cfg.SessionTokenLength)
	if err != nil {
		return Session{}, err
	}
	sess := Session{
		ID:        uuid.New().String(),
		Token:     token,
		UserID:    userID,
		UserIP:    f.IP(),
		CreatedAt: now,
		ExpiresAt: now.Add(s.cfg.SessionTokenDuration),
	}

	sql, args = psql.Insert(
		im.Into("auth.sessions",
			"id", "token", "user_id", "user_ip", "created_at", "expires_at",
		),
		im.Values(psql.Arg(sess.ID, sess.Token, sess.UserID, sess.UserIP, sess.CreatedAt, sess.ExpiresAt)),
	).MustBuild()

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return Session{}, err
	}

	sql, args = psql.Update(
		um.Table("auth.users"),
		um.Set("last_login_at").ToArg(f.Start()),
		um.Set("last_login_ip").ToArg(f.IP()),
		um.Where(psql.Quote("id").EQ(psql.Arg(userID))),
	).MustBuild()

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return Session{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return sess, nil

}

func (s *Service) ClearSessions(f *flux.Flow, in *ClearSessionInput) (int, error) {
	ctx := f.Context()

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
	res, err := s.db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return int(res.RowsAffected()), nil
}

func (s *Service) RenewSession(f *flux.Flow, id string) (time.Time, error) {
	ctx := f.Context()
	now := f.Start()
	expiresAt := now.Add(s.cfg.SessionTokenDuration)

	sql, args := psql.Update(
		um.Table("auth.sessions"),
		um.Set("expires_at").ToArg(expiresAt),
		um.Where(psql.Quote("id").EQ(psql.Arg(id))),
	).MustBuild()

	res, err := s.db.Exec(ctx, sql, args...)
	if err != nil {
		return time.Time{}, err
	}
	if res.RowsAffected() == 0 {
		return time.Time{}, ErrSessionNotFound
	}
	return expiresAt, nil
}
