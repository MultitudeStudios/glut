package auth

import (
	"fmt"
	"glut/common/flux"
	"glut/common/valid"
	"time"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

const (
	defaultBanQueryLimit = 20
	maxBanQueryLimit     = 100
)

type Ban struct {
	UserID      string
	Reason      string
	Description *string
	BannedBy    *string
	BannedAt    time.Time
	UnbannedAt  time.Time
}

type BanQuery struct {
	UserID         string
	Limit          int
	Offset         int
	IncludeExpired bool
}

type BanUserInput struct {
	UserID      string
	Reason      string
	Description *string
	Duration    int64
	Replace     bool
}

type UnbanUserInput struct {
	UserID string
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
			"reason",
			"description",
			"banned_by",
			"banned_at",
			"unbanned_at",
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
			psql.WhereOr(
				sm.Where(psql.Quote("unbanned_at").GT(psql.Arg(f.Time))),
				sm.Where(psql.Quote("banned_at").EQ(psql.Quote("unbanned_at"))),
			),
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
		var reason string
		var description *string
		var bannedBy *string
		var bannedAt time.Time
		var unbannedAt time.Time

		if err := rows.Scan(
			&userID,
			&reason,
			&description,
			&bannedBy,
			&bannedAt,
			&unbannedAt,
		); err != nil {
			return nil, err
		}
		bans = append(bans, Ban{
			UserID:      userID,
			Reason:      reason,
			Description: description,
			BannedBy:    bannedBy,
			BannedAt:    bannedAt,
			UnbannedAt:  unbannedAt,
		})
	}
	if in.UserID != "" && len(bans) == 0 {
		return nil, ErrBanNotFound
	}
	return bans, nil
}

func (s *Service) BanUser(f *flux.Flow, in *BanUserInput) (Ban, error) {
	var errs valid.Errors
	if in.UserID == "" {
		errs = append(errs, valid.Error{Field: "user_id", Error: "Required."})
	} else if !valid.IsUUID(in.UserID) {
		errs = append(errs, valid.Error{Field: "user_id", Error: "Invalid id."})
	}
	if in.Reason == "" {
		errs = append(errs, valid.Error{Field: "reason", Error: "Required."})
	}
	if in.Duration == 0 {
		errs = append(errs, valid.Error{Field: "duration", Error: "Required."})
	}
	if len(errs) != 0 {
		return Ban{}, errs
	}

	tx, err := s.db.Begin(f.Ctx)
	if err != nil {
		return Ban{}, err
	}
	defer tx.Rollback(f.Ctx)

	var unbannedAt = f.Time
	if in.Duration > 0 {
		unbannedAt = unbannedAt.Add(time.Duration(in.Duration) * time.Second)
	}

	sql, args := psql.Select(
		sm.From("auth.users"),
		sm.Columns("id"),
		sm.Where(psql.Quote("id").EQ(psql.Arg(in.UserID))),
		sm.ForNoKeyUpdate(),
	).MustBuild()

	var userExists bool
	if err := tx.QueryRow(f.Ctx, fmt.Sprintf("SELECT EXISTS (%s)", sql), args...).Scan(&userExists); err != nil {
		return Ban{}, err
	}
	if !userExists {
		return Ban{}, ErrUserNotFound
	}

	sql, args = psql.Select(
		sm.From("auth.bans"),
		sm.Columns("user_id"),
		psql.WhereAnd(
			sm.Where(psql.Quote("user_id").EQ(psql.Arg(in.UserID))),
			psql.WhereOr(
				sm.Where(psql.Quote("unbanned_at").GT(psql.Arg(f.Time))),
				sm.Where(psql.Quote("banned_at").EQ(psql.Quote("unbanned_at"))),
			),
		),
		sm.ForNoKeyUpdate(),
	).MustBuild()

	var banExists bool
	if err := tx.QueryRow(f.Ctx, fmt.Sprintf("SELECT EXISTS (%s)", sql), args...).Scan(&banExists); err != nil {
		return Ban{}, err
	}
	if banExists && !in.Replace {
		return Ban{}, ErrBanExists
	}

	ban := Ban{
		UserID:      in.UserID,
		Reason:      in.Reason,
		Description: in.Description,
		BannedBy:    &f.Session.User,
		BannedAt:    f.Time,
		UnbannedAt:  unbannedAt,
	}

	q := psql.Insert(
		im.Into("auth.bans", "user_id", "reason", "description", "banned_by", "banned_at", "unbanned_at"),
		im.Values(psql.Arg(ban.UserID, ban.Reason, ban.Description, ban.BannedBy, ban.BannedAt, ban.UnbannedAt)),
	)
	if !banExists || in.Replace {
		q.Apply(
			im.OnConflict("user_id").DoUpdate().SetExcluded("reason", "description", "banned_by", "banned_at", "unbanned_at"),
		)
	}
	sql, args = q.MustBuild()

	if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
		return Ban{}, err
	}

	sql, args = psql.Delete(
		dm.From("auth.sessions"),
		dm.Where(psql.Quote("user_id").EQ(psql.Arg(in.UserID))),
	).MustBuild()

	if _, err := tx.Exec(f.Ctx, sql, args...); err != nil {
		return Ban{}, err
	}

	if err := tx.Commit(f.Ctx); err != nil {
		return Ban{}, err
	}
	return ban, nil
}

func (s *Service) UnbanUser(f *flux.Flow, in *UnbanUserInput) error {
	var errs valid.Errors
	if in.UserID == "" {
		errs = append(errs, valid.Error{Field: "user_id", Error: "Required."})
	} else if !valid.IsUUID(in.UserID) {
		errs = append(errs, valid.Error{Field: "user_id", Error: "Invalid id."})
	}
	if len(errs) != 0 {
		return errs
	}

	sql, args := psql.Delete(
		dm.From("auth.bans"),
		dm.Where(psql.Quote("user_id").EQ(psql.Arg(in.UserID))),
	).MustBuild()

	res, err := s.db.Exec(f.Ctx, sql, args...)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrBanNotFound
	}
	return nil
}
