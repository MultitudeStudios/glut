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

func queryUsers(s *auth.Service) flux.HandlerFunc {
	type request struct {
		ID     string `json:"id"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}

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
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}
		users, err := s.Users(f, &auth.UserQuery{
			ID:     r.ID,
			Limit:  r.Limit,
			Offset: r.Offset,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrUserNotFound) {
				return flux.NotFoundError("User not found.")
			}
			return fmt.Errorf("api.queryUsers: %w", err)
		}

		res := []response{}
		for _, user := range users {
			res = append(res, response{
				ID:          user.ID,
				Username:    user.Username,
				Email:       user.Email,
				CreatedAt:   user.CreatedAt,
				UpdatedAt:   user.UpdatedAt,
				CreatedBy:   user.CreatedBy,
				UpdatedBy:   user.UpdatedBy,
				LastLoginAt: user.LastLoginAt,
				LastLoginIP: user.LastLoginIP,
			})
		}
		return f.Respond(http.StatusOK, res)
	}
}

func createUser(s *auth.Service) flux.HandlerFunc {
	type request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		ID        string    `json:"id"`
		Username  string    `json:"username"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}

		user, err := s.CreateUser(f, &auth.NewUserInput{
			Username: r.Username,
			Email:    r.Email,
			Password: r.Password,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrUserExists) {
				flux.NewError("user_exists", http.StatusConflict, "User already exists.")
			}
			return fmt.Errorf("api.createUser: %w", err)
		}

		res := &response{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		}
		return f.Respond(http.StatusOK, res)
	}
}

func deleteUsers(s *auth.Service) flux.HandlerFunc {
	type request struct {
		IDs []string `json:"ids"`
	}

	type response struct {
		Count int `json:"count"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}

		count, err := s.DeleteUsers(f, &auth.DeleteUsersInput{
			IDs: r.IDs,
		})
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.deleteUsers: %w", err)
		}
		return f.Respond(http.StatusOK, &response{count})
	}
}

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
