package auth

import (
	"glut/common/flux"
	"glut/common/valid"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

const (
	defaultRoleQueryLimit = 20
	maxRoleQueryLimit     = 100
)

type Role struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	Meta        *RoleMeta  `json:"meta,omitempty"`
}

type RoleMeta struct {
	CreatedByID       string  `json:"created_by_id"`
	CreatedByUsername string  `json:"created_by_username"`
	CreatedByEmail    string  `json:"created_by_email"`
	UpdatedByID       *string `json:"updated_by_id"`
	UpdatedByUsername *string `json:"updated_by_username"`
	UpdatedByEmail    *string `json:"updated_by_email"`
}

type RoleQuery struct {
	ID       string `json:"id"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Detailed bool   `json:"detailed"`
}

func (s *Service) Roles(f *flux.Flow, in RoleQuery) ([]Role, error) {
	var errs valid.Errors
	if in.ID != "" && !valid.IsUUID(in.ID) {
		errs = append(errs, valid.Error{Field: "id", Error: "Invalid id."})
	}
	if len(errs) != 0 {
		return nil, errs
	}

	if in.Limit <= 0 || in.Limit > maxRoleQueryLimit {
		in.Limit = defaultRoleQueryLimit
	}
	if in.Offset < 0 {
		in.Offset = 0
	}

	cols := []any{"r.id", "r.name", "r.description", "r.created_at", "r.updated_at"}
	if in.Detailed {
		cols = append(cols, "cb.id", "cb.username", "cb.email", "ub.id", "ub.username", "ub.email")
	}
	q := psql.Select(
		sm.Columns(cols...),
		sm.From("auth.roles").As("r"),
	)
	if in.Detailed {
		q.Apply(
			sm.LeftJoin("auth.users").As("cb").OnEQ(
				psql.Raw("r.created_by"), psql.Raw("cb.id"),
			),
			sm.LeftJoin("auth.users").As("ub").OnEQ(
				psql.Raw("r.updated_by"), psql.Raw("ub.id"),
			),
		)
	}
	if in.ID != "" {
		q.Apply(
			sm.Where(psql.Quote("id").EQ(psql.Arg(in.ID))),
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

	var roles []Role
	for rows.Next() {
		var id string
		var name string
		var description string
		var createdAt time.Time
		var updatedAt *time.Time
		var createdByID string
		var createdByUsername string
		var createdByEmail string
		var updatedByID *string
		var updatedByUsername *string
		var updatedByEmail *string

		dest := []any{&id, &name, &description, &createdAt, &updatedAt}
		if in.Detailed {
			dest = append(dest, &createdByID, &createdByUsername, &createdByEmail, &updatedByID, &updatedByUsername, &updatedByEmail)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		role := Role{
			ID:          id,
			Name:        name,
			Description: description,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}
		if in.Detailed {
			role.Meta = &RoleMeta{
				CreatedByID:       createdByID,
				CreatedByUsername: createdByUsername,
				CreatedByEmail:    createdByEmail,
				UpdatedByID:       updatedByID,
				UpdatedByUsername: updatedByUsername,
				UpdatedByEmail:    updatedByEmail,
			}
		}
		roles = append(roles, role)
	}
	if in.ID != "" && len(roles) == 0 {
		return nil, ErrRoleNotFound
	}
	return roles, nil
}
