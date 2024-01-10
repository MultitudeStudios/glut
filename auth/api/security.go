package api

import (
	"errors"
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
	"time"
)

func (a *API) QueryBans(f *flux.Flow, r *QueryBansRequest) ([]BanResponse, error) {
	bans, err := a.service.Bans(f, &auth.BanQuery{
		UserID:         r.UserID,
		Limit:          r.Limit,
		Offset:         r.Offset,
		IncludeExpired: r.IncludeExpired,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		if errors.Is(err, auth.ErrBanNotFound) {
			return nil, flux.NotFoundError("Ban not found.")
		}
		return nil, fmt.Errorf("api.QueryBans: %w", err)
	}

	res := []BanResponse{}
	for _, ban := range bans {
		res = append(res, BanResponse{
			UserID:     ban.UserID,
			BanReason:  ban.BanReason,
			BannedBy:   ban.BannedBy,
			BannedAt:   ban.BannedAt,
			UnbannedAt: ban.UnbannedAt,
			UpdatedAt:  ban.UpdatedAt,
			UpdatedBy:  ban.UpdatedBy,
		})
	}
	return res, nil
}

type QueryBansRequest struct {
	UserID         string `json:"user_id"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
	IncludeExpired bool   `json:"include_expired"`
}

type BanResponse struct {
	UserID     string     `json:"user_id"`
	BanReason  string     `json:"ban_reason"`
	BannedBy   *string    `json:"banned_by"`
	BannedAt   time.Time  `json:"banned_at"`
	UnbannedAt time.Time  `json:"unbanned_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
	UpdatedBy  *string    `json:"updated_by"`
}
