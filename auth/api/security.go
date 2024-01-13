package api

import (
	"errors"
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
	"net/http"
	"time"
)

func queryBans(s *auth.Service) flux.HandlerFunc {
	type request struct {
		UserID         string `json:"user_id"`
		Limit          int    `json:"limit"`
		Offset         int    `json:"offset"`
		IncludeExpired bool   `json:"include_expired"`
	}

	type response struct {
		UserID     string     `json:"user_id"`
		BanReason  string     `json:"ban_reason"`
		BannedBy   *string    `json:"banned_by"`
		BannedAt   time.Time  `json:"banned_at"`
		UnbannedAt time.Time  `json:"unbanned_at"`
		UpdatedAt  *time.Time `json:"updated_at"`
		UpdatedBy  *string    `json:"updated_by"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(&r); err != nil {
			return err
		}

		bans, err := s.Bans(f, &auth.BanQuery{
			UserID:         r.UserID,
			Limit:          r.Limit,
			Offset:         r.Offset,
			IncludeExpired: r.IncludeExpired,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrBanNotFound) {
				return flux.NotFoundError("Ban not found.")
			}
			return fmt.Errorf("api.queryBans: %w", err)
		}

		res := []response{}
		for _, ban := range bans {
			res = append(res, response{
				UserID:     ban.UserID,
				BanReason:  ban.BanReason,
				BannedBy:   ban.BannedBy,
				BannedAt:   ban.BannedAt,
				UnbannedAt: ban.UnbannedAt,
				UpdatedAt:  ban.UpdatedAt,
				UpdatedBy:  ban.UpdatedBy,
			})
		}
		return f.Respond(http.StatusOK, res)
	}
}
