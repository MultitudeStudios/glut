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

func myUser(s *auth.Service) flux.HandlerFunc {
	type response struct {
		ID          string     `json:"id"`
		Username    string     `json:"username"`
		Email       string     `json:"email"`
		CreatedAt   time.Time  `json:"created_at"`
		UpdatedAt   *time.Time `json:"updated_at"`
		CreatedBy   *string    `json:"created_by"`
		UpdatedBy   *string    `json:"updated_by"`
		LastLoginAt *time.Time `json:"last_login_at"`
		LastLoginIP *string    `json:"last_login_ip"`
	}

	return func(f *flux.Flow) error {
		users, err := s.Users(f, &auth.UserQuery{
			ID: f.Session.User,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrUserNotFound) {
				return flux.NotFoundError("User not found.")
			}
			return fmt.Errorf("api.myUser: %w", err)
		}
		user := users[0]
		res := &response{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			CreatedBy:   user.CreatedBy,
			UpdatedBy:   user.UpdatedBy,
			LastLoginAt: user.LastLoginAt,
			LastLoginIP: user.LastLoginIP,
		}
		return f.Respond(http.StatusOK, res)
	}
}

func mySessions(s *auth.Service) flux.HandlerFunc {
	type response struct {
		ID        string    `json:"id"`
		Token     string    `json:"token"`
		UserID    string    `json:"user_id"`
		UserIP    string    `json:"user_ip"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	return func(f *flux.Flow) error {
		sessions, err := s.Sessions(f, &auth.SessionQuery{
			UserID: f.Session.User,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.mySessions: %w", err)
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

func logout(s *auth.Service) flux.HandlerFunc {
	type request struct {
		IDs []string `json:"ids"`
	}

	type response struct {
		Count int `json:"count"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(&r); err != nil {
			return err
		}

		count, err := s.ClearSessions(f, &auth.ClearSessionInput{
			IDs:    r.IDs,
			UserID: f.Session.User,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.Logout: %w", err)
		}
		return f.Respond(http.StatusOK, response{count})
	}
}

func renewSession(s *auth.Service) flux.HandlerFunc {
	type response struct {
		ExpiresAt time.Time `json:"expires_at"`
	}

	return func(f *flux.Flow) error {
		newExpiry, err := s.RenewSession(f)
		if err != nil {
			if errors.Is(err, auth.ErrSessionNotFound) {
				return flux.UnauthorizedError
			}
			return fmt.Errorf("api.renewSession: %w", err)
		}
		return f.Respond(http.StatusOK, &response{newExpiry})
	}
}

func deleteMyUser(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		_, err := s.DeleteUsers(f, &auth.DeleteUsersInput{
			IDs: []string{f.Session.User},
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.deleteMyUser: %w", err)
		}
		return f.Respond(http.StatusOK, nil)
	}
}
