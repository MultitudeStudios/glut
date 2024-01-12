package api

import (
	"errors"
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
	"net/http"
)

func changePassword(s *auth.Service) flux.HandlerFunc {
	type request struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}

		if err := s.ChangePassword(f, &auth.ChangePasswordInput{
			OldPassword: r.OldPassword,
			NewPassword: r.NewPassword,
		}); err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrInvalidPassword) {
				return flux.NewError("invalid_password", http.StatusForbidden, "Invalid password.")
			}
			return fmt.Errorf("api.changePassword: %w", err)
		}
		return f.Respond(http.StatusOK, nil)
	}
}

func changeEmail(s *auth.Service) flux.HandlerFunc {
	type request struct {
		Token    string `json:"token"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}
		return nil
	}
}

func verifyUser(s *auth.Service) flux.HandlerFunc {
	type request struct {
		Token string `json:"token"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(r); err != nil {
			return err
		}

		if err := s.VerifyUser(f, &auth.VerifyUserInput{
			Token: r.Token,
		}); err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				return flux.UnauthorizedError
			}
			if errors.Is(err, auth.ErrInvalidToken) {
				return flux.NewError("invalid_token", http.StatusUnprocessableEntity, "Invalid token.")
			}
			if errors.Is(err, auth.ErrUserVerified) {
				return flux.NewError("user_verified", http.StatusConflict, "User already verified.")
			}
			if errors.Is(err, auth.ErrTryLater) {
				return flux.TryLaterError("User verification was initiated recently. Try again later.")
			}
			return fmt.Errorf("api.verifyUser: %w", err)
		}
		return f.Respond(http.StatusOK, nil)
	}
}
