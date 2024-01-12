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

func querySessions(s *auth.Service) flux.HandlerFunc {
	type request struct {
		ID             string `json:"id"`
		Limit          int    `json:"limit"`
		Offset         int    `json:"offset"`
		UserID         string `json:"user_id"`
		IncludeExpired bool   `json:"include_expired"`
	}

	type response struct {
		ID        string    `json:"id"`
		Token     string    `json:"token"`
		UserID    string    `json:"user_id"`
		UserIP    string    `json:"user_ip"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}

		sessions, err := s.Sessions(f, &auth.SessionQuery{
			ID:             r.ID,
			Limit:          r.Limit,
			Offset:         r.Offset,
			UserID:         r.UserID,
			IncludeExpired: r.IncludeExpired,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.QuerySessions: %w", err)
		}

		res := []response{}
		for _, sess := range sessions {
			res = append(res, response{
				ID:        sess.ID,
				Token:     sess.Token,
				UserID:    sess.UserID,
				UserIP:    sess.UserIP,
				CreatedAt: sess.CreatedAt,
				ExpiresAt: sess.ExpiresAt,
			})
		}
		return f.Respond(http.StatusOK, res)
	}
}

func createSession(s *auth.Service) flux.HandlerFunc {
	type request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type response struct {
		ID        string    `json:"id"`
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}

		sess, err := s.CreateSession(f, &auth.Credentials{
			Username: r.Username,
			Password: r.Password,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrInvalidCredentials) {
				return flux.NewError("invalid_credentials", http.StatusUnauthorized, "Invalid credentials.")
			}
			if errors.Is(err, auth.ErrSessionLimit) {
				return flux.NewError("session_limit", http.StatusConflict, "Session limit reached.")
			}
			return fmt.Errorf("api.createSession: %w", err)
		}

		res := &response{
			ID:        sess.ID,
			Token:     sess.Token,
			ExpiresAt: sess.ExpiresAt,
		}
		return f.Respond(http.StatusOK, res)
	}
}

func clearSessions(s *auth.Service) flux.HandlerFunc {
	type request struct {
		IDs    []string `json:"ids"`
		UserID string   `json:"user_id"`
	}

	type response struct {
		Count int `json:"count"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}

		count, err := s.ClearSessions(f, &auth.ClearSessionInput{
			IDs:    r.IDs,
			UserID: r.UserID,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.clearSessions: %w", err)
		}
		return f.Respond(http.StatusOK, &response{count})
	}
}
