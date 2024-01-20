package auth

import (
	"fmt"
	"glut/common/flux"
	"glut/common/sqlutil"
	"glut/common/valid"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

const (
	defaultUserQueryLimit = 20
	maxUserQueryLimit     = 100
	defaultUserSortBy     = "created_at"
	defaultUserSortDir    = sqlutil.SortDirAsc
)

type User struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Password    string     `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	VerifiedAt  *time.Time `json:"verified_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
	LastLoginIP *string    `json:"last_login_ip"`
}

type UserQuery struct {
	ID       string `json:"id"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Sort     string `json:"sort"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type CreateUserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type DeleteUsersInput struct {
	IDs []string `json:"ids"`
}

var userSortByMap map[string]struct{} = map[string]struct{}{
	"username":      {},
	"email":         {},
	"created_at":    {},
	"last_login_at": {},
}

func getUserSort(value string) (string, string, bool) {
	if value == "" {
		return defaultUserSortBy, defaultUserSortDir, true
	}
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return "", "", false
	}
	sortBy := parts[0]
	sortDir := strings.ToUpper(parts[1])
	if _, ok := userSortByMap[sortBy]; !ok {
		return "", "", false
	}
	if sortDir != sqlutil.SortDirAsc && sortDir != sqlutil.SortDirDesc {
		return "", "", false
	}
	return sortBy, sortDir, true
}

func (s *Service) Users(f *flux.Flow, in UserQuery) ([]User, error) {
	var errs valid.Errors
	if in.ID != "" && !valid.IsUUID(in.ID) {
		errs = append(errs, valid.Error{Field: "id", Error: "Invalid id."})
	}
	sortBy, sortDir, ok := getUserSort(in.Sort)
	if !ok {
		errs = append(errs, valid.Error{Field: "sort", Error: "Invalid sort."})
	}
	if len(errs) != 0 {
		return nil, errs
	}

	if in.Limit <= 0 || in.Limit > maxUserQueryLimit {
		in.Limit = defaultUserQueryLimit
	}
	if in.Offset < 0 {
		in.Offset = 0
	}

	q := psql.Select(
		sm.Columns(
			"id",
			"username",
			"email",
			"created_at",
			"last_login_at",
			"last_login_ip",
		),
		sm.From("auth.users"),
	)
	if in.ID != "" {
		q.Apply(
			sm.Where(psql.Quote("id").EQ(psql.Arg(in.ID))),
		)
	}
	if in.Email != "" {
		q.Apply(
			sm.Where(psql.Quote("email").ILike(psql.Arg(fmt.Sprintf("%%%s%%", in.Email)))),
		)
	}
	if in.Username != "" {
		q.Apply(
			sm.Where(psql.Quote("username").ILike(psql.Arg(fmt.Sprintf("%%%s%%", in.Username)))),
		)
	}
	if sortDir == sqlutil.SortDirAsc {
		q.Apply(sm.OrderBy(sortBy).Asc())
	} else {
		q.Apply(sm.OrderBy(sortBy).Desc())
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

	users := []User{}
	for rows.Next() {
		var id string
		var username string
		var email string
		var createdAt time.Time
		var lastLoginAt *time.Time
		var lastLoginIP *string

		if err := rows.Scan(
			&id,
			&username,
			&email,
			&createdAt,
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
			LastLoginAt: lastLoginAt,
			LastLoginIP: lastLoginIP,
		})
	}
	if in.ID != "" && len(users) == 0 {
		return nil, ErrUserNotFound
	}
	return users, nil
}

func (s *Service) CreateUser(f *flux.Flow, in CreateUserInput) (User, error) {
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

	sql, args := psql.Select(
		sm.Columns("id"),
		sm.From("auth.users"),
		sm.Where(psql.Quote("username").EQ(psql.Arg(in.Username))),
	).MustBuild()

	var userExists bool
	if err := s.db.QueryRow(f.Ctx, sqlutil.Exists(sql), args...).Scan(&userExists); err != nil {
		return User{}, err
	}
	if userExists {
		return User{}, ErrUserExists
	}

	passwordHash, err := hashPassword(in.Password)
	if err != nil {
		return User{}, nil
	}
	user := User{
		ID:        uuid.New().String(),
		Username:  in.Username,
		Email:     in.Email,
		Password:  passwordHash,
		CreatedAt: f.Time,
	}

	tx, err := s.db.Begin(f.Ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(f.Ctx)

	sql, args = psql.Insert(
		im.Into("auth.users",
			"id", "username", "email", "password_hash", "created_at",
		),
		im.Values(psql.Arg(user.ID, user.Username, user.Email, user.Password, user.CreatedAt)),
	).MustBuild()

	if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
		return User{}, err
	}

	token := s.createUserVerificationToken(f.Session.User, f.Time)
	if err := saveToken(f, tx, token); err != nil {
		return User{}, err
	}

	// TODO: send user verification email with token

	if err := tx.Commit(f.Ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) DeleteUsers(f *flux.Flow, in DeleteUsersInput) (int, error) {
	var errs valid.Errors
	if len(in.IDs) == 0 {
		errs = append(errs, valid.Error{Field: "ids", Error: "Required."})
	}
	if !valid.IsUUIDSlice(in.IDs) {
		errs = append(errs, valid.Error{Field: "ids", Error: "Contains invalid id."})
	}
	if len(errs) != 0 {
		return 0, errs
	}

	q := psql.Delete(dm.From("auth.users"))
	if in.IDs != nil {
		q.Apply(dm.Where(
			psql.Quote("id").In(
				psql.Arg(sqlutil.AnySlice(in.IDs)...)),
		))
	}
	sql, args := q.MustBuild()

	res, err := s.db.Exec(f.Ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return int(res.RowsAffected()), nil
}
