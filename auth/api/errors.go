package api

import (
	"glut/common/flux"
	"net/http"
)

var (
	ErrInvalidToken       = flux.NewError("invalid_token", http.StatusUnprocessableEntity, "Invalid token.")
	ErrUserVerified       = flux.NewError("user_verified", http.StatusConflict, "User already verified.")
	ErrInvalidPassword    = flux.NewError("invalid_password", http.StatusForbidden, "Invalid password.")
	ErrInvalidCredentials = flux.NewError("invalid_credentials", http.StatusUnauthorized, "Invalid credentials.")
	ErrUserBanned         = flux.NewError("user_banned", http.StatusForbidden, "User is banned.")
	ErrSessionLimit       = flux.NewError("session_limit", http.StatusConflict, "Session limit reached.")
)
