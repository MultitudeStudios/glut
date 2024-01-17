package api

import (
	"errors"
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
	"net/http"
)

func querySessions(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var q auth.SessionQuery
		if err := f.Bind(&q); err != nil {
			return err
		}

		sessions, err := s.Sessions(f, q)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.QuerySessions: %w", err)
		}
		return f.Respond(http.StatusOK, sessions)
	}
}

func createSession(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var creds auth.Credentials
		if err := f.Bind(&creds); err != nil {
			return err
		}

		sess, err := s.CreateSession(f, creds)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrInvalidCredentials) {
				return ErrInvalidCredentials
			}
			if errors.Is(err, auth.ErrUserBanned) {
				return auth.ErrUserBanned
			}
			if errors.Is(err, auth.ErrSessionLimit) {
				return ErrSessionLimit
			}
			return fmt.Errorf("api.createSession: %w", err)
		}
		return f.Respond(http.StatusOK, sess)
	}
}

func clearSessions(s *auth.Service) flux.HandlerFunc {
	type response struct {
		Count int `json:"count"`
	}

	return func(f *flux.Flow) error {
		var in auth.ClearSessionInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		count, err := s.ClearSessions(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.clearSessions: %w", err)
		}
		return f.Respond(http.StatusOK, &response{count})
	}
}
