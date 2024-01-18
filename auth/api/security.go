package api

import (
	"errors"
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
	"net/http"
)

func queryBans(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.BanQuery
		if err := f.Bind(&in); err != nil {
			return err
		}

		bans, err := s.Bans(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrBanNotFound) {
				return flux.NotFoundError("Ban not found.")
			}
			return fmt.Errorf("api.queryBans: %w", err)
		}
		return f.Respond(http.StatusOK, bans)
	}
}

func banUser(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.BanUserInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		ban, err := s.BanUser(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrUserNotFound) {
				return flux.NotFoundError("User not found.")
			}
			if errors.Is(err, auth.ErrBanExists) {
				return flux.ExistsError("Ban already exists.")
			}
			return fmt.Errorf("api.banUser: %w", err)
		}
		return f.Respond(http.StatusOK, ban)
	}
}

func unbanUser(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.UnbanUserInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		if err := s.UnbanUser(f, in); err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrBanNotFound) {
				return flux.NotFoundError("Ban not found.")
			}
			return fmt.Errorf("api.unbanUser: %w", err)
		}
		return f.Respond(http.StatusOK, nil)
	}
}
