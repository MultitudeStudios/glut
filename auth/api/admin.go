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
	return func(f *flux.Flow) error {
		var in auth.ChangePasswordInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		if err := s.ChangePassword(f, in); err != nil {
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
	return func(f *flux.Flow) error {
		var in auth.ChangeEmailInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		if err := s.ChangeEmail(f, in); err != nil {
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
	return func(f *flux.Flow) error {
		var in auth.VerifyUserInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		if err := s.VerifyUser(f, in); err != nil {
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
	return func(f *flux.Flow) error {
		var in auth.ResetPasswordInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		if err := s.ResetPassword(f, in); err != nil {
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

func forgotUsername(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.ForgotUsernameInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		if err := s.ForgotUsername(f, in); err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.forgotUsername: %w", err)
		}
		return f.Respond(http.StatusOK, nil)
	}
}
