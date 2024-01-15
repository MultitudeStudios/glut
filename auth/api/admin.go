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
		if err := f.Bind(&r); err != nil {
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
				return ErrInvalidPassword
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
		if err := f.Bind(&r); err != nil {
			return err
		}

		if err := s.ChangeEmail(f, &auth.ChangeEmailInput{
			Token:    r.Token,
			Email:    r.Email,
			Password: r.Password,
		}); err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrUnauthorized) {
				return flux.UnauthorizedError
			}
			if errors.Is(err, auth.ErrInvalidPassword) {
				return ErrInvalidPassword
			}
			if errors.Is(err, auth.ErrInvalidToken) {
				return ErrInvalidToken
			}
			return fmt.Errorf("api.changeEmail: %w", err)
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
		if err := f.Bind(&r); err != nil {
			return err
		}

		if err := s.VerifyUser(f, &auth.VerifyUserInput{
			Token: r.Token,
		}); err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				return flux.UnauthorizedError
			}
			if errors.Is(err, auth.ErrInvalidToken) {
				return ErrInvalidToken
			}
			if errors.Is(err, auth.ErrUserVerified) {
				return ErrUserVerified
			}
			if errors.Is(err, auth.ErrTryLater) {
				return flux.TryLaterError("User verification was initiated recently. Try again later.")
			}
			return fmt.Errorf("api.verifyUser: %w", err)
		}
		return f.Respond(http.StatusOK, nil)
	}
}

func resetPassword(s *auth.Service) flux.HandlerFunc {
	type request struct {
		Token    string `json:"token"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	return func(f *flux.Flow) error {
		var r request
		if err := f.Bind(&r); err != nil {
			return err
		}

		if err := s.ResetPassword(f, &auth.ResetPasswordInput{
			Token:    r.Token,
			Email:    r.Email,
			Password: r.Password,
		}); err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrInvalidToken) {
				return ErrInvalidToken
			}
			return fmt.Errorf("api.resetPassword: %w", err)
		}
		return f.Respond(http.StatusOK, nil)
	}
}
