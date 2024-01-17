package api

import (
	"errors"
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
	"net/http"
)

func queryUsers(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.UserQuery
		if err := f.Bind(&in); err != nil {
			return err
		}
		users, err := s.Users(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrUserNotFound) {
				return flux.NotFoundError("User not found.")
			}
			return fmt.Errorf("api.queryUsers: %w", err)
		}
		return f.Respond(http.StatusOK, users)
	}
}

func createUser(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.CreateUserInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		user, err := s.CreateUser(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrUserExists) {
				return flux.ExistsError("User already exists.")
			}
			return fmt.Errorf("api.createUser: %w", err)
		}
		return f.Respond(http.StatusOK, user)
	}
}

func deleteUsers(s *auth.Service) flux.HandlerFunc {
	type response struct {
		Count int `json:"count"`
	}

	return func(f *flux.Flow) error {
		var in auth.DeleteUsersInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		count, err := s.DeleteUsers(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.deleteUsers: %w", err)
		}
		return f.Respond(http.StatusOK, &response{count})
	}
}
