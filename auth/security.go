package auth

import (
	"glut/common/flux"
	"glut/common/valid"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

const (
	defaultBanQueryLimit = 20
	maxBanQueryLimit     = 100
)

type Ban struct {
	UserID     string
	BanReason  string
	BannedBy   *string
	BannedAt   time.Time
	UnbannedAt time.Time
	UpdatedAt  *time.Time
	UpdatedBy  *string
}

type BanQuery struct {
	UserID         string
	Limit          int
	Offset         int
	IncludeExpired bool
}

func (s *Service) Bans(f *flux.Flow, in *BanQuery) ([]Ban, error) {
	var errs valid.Errors
	if in.UserID != "" && !valid.IsUUID(in.UserID) {
		errs = append(errs, valid.Error{Field: "user_id", Error: "Invalid id."})
	}
	if len(errs) != 0 {
		return nil, errs
	}

	if in.Limit <= 0 || in.Limit > maxBanQueryLimit {
		in.Limit = defaultBanQueryLimit
	}
	if in.Offset < 0 {
		in.Offset = 0
	}

	q := psql.Select(
		sm.Columns(
			"user_id",
			"ban_reason",
			"banned_by",
			"banned_at",
			"unbanned_at",
			"updated_at",
			"updated_by",
		),
		sm.From("auth.bans"),
	)
	if in.UserID != "" {
		q.Apply(
			sm.Where(psql.Quote("user_id").EQ(psql.Arg(in.UserID))),
		)
	}
	if !in.IncludeExpired {
		q.Apply(
			sm.Where(psql.Quote("unbanned_at").GT(psql.Arg(f.Time))),
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

	var bans []Ban
	for rows.Next() {
		var userID string
		var banReason string
		var bannedBy *string
		var bannedAt time.Time
		var unbannedAt time.Time
		var updatedAt *time.Time
		var updatedBy *string

		if err := rows.Scan(
			&userID,
			&banReason,
			&bannedBy,
			&bannedAt,
			&unbannedAt,
			&updatedAt,
			&updatedBy,
		); err != nil {
			return nil, err
		}
		bans = append(bans, Ban{
			UserID:     userID,
			BanReason:  banReason,
			BannedBy:   bannedBy,
			BannedAt:   bannedAt,
			UnbannedAt: unbannedAt,
			UpdatedAt:  updatedAt,
			UpdatedBy:  updatedBy,
		})
	}
	if in.UserID != "" && len(bans) == 0 {
		return nil, ErrBanNotFound
	}
	return bans, nil
}
