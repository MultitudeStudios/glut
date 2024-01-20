package auth

import "errors"

var (
	ErrTryLater           = errors.New("try again later")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserVerified       = errors.New("user already verified")
	ErrUserBanned         = errors.New("user banned")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionLimit       = errors.New("reached session limit")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidToken       = errors.New("invalid token")
	ErrBanNotFound        = errors.New("ban not found")
	ErrBanExists          = errors.New("ban already exists")
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleExists         = errors.New("role already exists")
	ErrPermissionNotFound = errors.New("permission not found")
)
