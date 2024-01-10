package api

import (
	"errors"
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
	"net/http"
)

func (a *API) ChangePassword(f *flux.Flow, r *ChangePasswordRequest) (flux.Empty, error) {
	if err := a.service.ChangePassword(f, &auth.ChangePasswordInput{
		OldPassword: r.OldPassword,
		NewPassword: r.NewPassword,
	}); err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		if errors.Is(err, auth.ErrInvalidPassword) {
			return nil, flux.NewError("invalid_password", http.StatusForbidden, "Invalid password.")
		}
		return nil, fmt.Errorf("api.ChangePassword: %w", err)
	}
	return nil, nil
}

func (a *API) VerifyUser(f *flux.Flow, r *VerifyUserRequest) (flux.Empty, error) {
	if err := a.service.VerifyUser(f, &auth.VerifyUserInput{
		Token: r.Token,
	}); err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			return nil, flux.UnauthorizedError
		}
		if errors.Is(err, auth.ErrInvalidToken) {
			return nil, flux.NewError("invalid_token", http.StatusUnprocessableEntity, "Invalid token.")
		}
		if errors.Is(err, auth.ErrUserVerified) {
			return nil, flux.NewError("user_verified", http.StatusConflict, "User already verified.")
		}
		if errors.Is(err, auth.ErrTryLater) {
			return nil, flux.TryLaterError("User verification was initiated recently. Try again later.")
		}
		return nil, fmt.Errorf("api.VerifyUser: %w", err)
	}
	return nil, nil
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type VerifyUserRequest struct {
	Token string `json:"token"`
}
