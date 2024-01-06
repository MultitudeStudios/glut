package auth

import (
	"context"
	"glut/common/flux"
	"glut/common/sqlutil"
	"glut/common/valid"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

const (
	defaultUserQueryLimit = 20
	maxUserQueryLimit     = 100
)

type User struct {
	ID          string
	Username    string
	Email       string
	Password    string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	VerifiedAt  *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	LastLoginAt *time.Time
	LastLoginIP *string
}

type UserQuery struct {
	ID     string
	Limit  int
	Offset int
}

type NewUserInput struct {
	Username string
	Email    string
	Password string
}

func (s *Service) Users(f *flux.Flow, uq *UserQuery) ([]User, error) {
	var errs valid.Errors
	if uq.ID != "" && !valid.IsUUID(uq.ID) {
		errs = append(errs, valid.Error{Field: "id", Error: "Invalid id."})
	}
	if len(errs) != 0 {
		return nil, errs
	}

	if uq.Limit <= 0 || uq.Limit > maxUserQueryLimit {
		uq.Limit = defaultUserQueryLimit
	}
	if uq.Offset < 0 {
		uq.Offset = 0
	}

	users, err := queryUsers(f.Context(), s.db, uq)
	if err != nil {
		return nil, err
	}
	if uq.ID != "" && len(users) == 0 {
		return nil, ErrUserNotFound
	}
	return users, nil
}

func (s *Service) CreateUser(f *flux.Flow, in *NewUserInput) (User, error) {
	var errs valid.Errors
	if in.Username == "" {
		errs = append(errs, valid.Error{Field: "username", Error: "Required."})
	}
	if in.Email == "" {
		errs = append(errs, valid.Error{Field: "email", Error: "Required."})
	}
	if in.Password == "" {
		errs = append(errs, valid.Error{Field: "password", Error: "Required."})
	}
	if len(errs) != 0 {
		return User{}, errs
	}

	ctx := f.Context()
	exists, err := userExists(ctx, s.db, in.Username)
	if err != nil {
		return User{}, err
	}
	if exists {
		return User{}, ErrUserExists
	}

	user, err := newUser(f, in.Username, in.Email, in.Password)
	if err != nil {
		return User{}, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	if err := saveUser(ctx, tx, user); err != nil {
		return User{}, err
	}
	if err := s.createUserVerificationToken(f, tx, user.ID); err != nil {
		return User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func queryUsers(ctx context.Context, db sqlutil.DB, uq *UserQuery) ([]User, error) {
	q := psql.Select(
		sm.Columns(
			"id",
			"username",
			"email",
			"created_at",
			"updated_at",
			"created_by",
			"updated_by",
			"last_login_at",
			"last_login_ip",
		),
		sm.From("auth.users"),
	)

	if uq.ID != "" {
		q.Apply(
			sm.Where(psql.Quote("id").EQ(psql.Arg(uq.ID))),
		)
	}
	if uq.Limit != 0 {
		q.Apply(
			sm.Limit(uq.Limit),
		)
	}
	if uq.Offset != 0 {
		q.Apply(
			sm.Offset(uq.Offset),
		)
	}

	sql, args := q.MustBuild()
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	users, err := scanUsers(rows)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func userExists(ctx context.Context, db sqlutil.DB, username string) (bool, error) {
	q := `
	SELECT
	EXISTS (
		SELECT id
		FROM auth.users
		WHERE username = $1
	);`

	var exists bool
	if err := db.QueryRow(ctx, q, username).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func saveUser(ctx context.Context, db sqlutil.DB, user User) error {
	q := psql.Insert(
		im.Into("auth.users",
			"id", "username", "email", "password_hash", "created_at",
		),
		im.Values(psql.Arg(user.ID, user.Username, user.Email, user.Password, user.CreatedAt)),
	)

	sql, args := q.MustBuild()
	if _, err := db.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return nil
}

func newUser(f *flux.Flow, username, email, pass string) (User, error) {
	passwordHash, err := hashPassword(pass)
	if err != nil {
		return User{}, nil
	}

	user := User{
		ID:        uuid.New().String(),
		Username:  username,
		Email:     email,
		Password:  passwordHash,
		CreatedAt: f.Start(),
	}
	return user, nil
}

func scanUsers(rows pgx.Rows) ([]User, error) {
	defer rows.Close()

	var users []User
	for rows.Next() {
		var id string
		var username string
		var email string
		var createdAt time.Time
		var updatedAt *time.Time
		var createdBy *string
		var updatedBy *string
		var lastLoginAt *time.Time
		var lastLoginIP *string

		if err := rows.Scan(
			&id,
			&username,
			&email,
			&createdAt,
			&updatedAt,
			&createdBy,
			&updatedBy,
			&lastLoginAt,
			&lastLoginIP,
		); err != nil {
			return nil, err
		}
		users = append(users, User{
			ID:          id,
			Username:    username,
			Email:       email,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
			CreatedBy:   createdBy,
			UpdatedBy:   updatedBy,
			LastLoginAt: lastLoginAt,
			LastLoginIP: lastLoginIP,
		})
	}
	return users, nil
}
