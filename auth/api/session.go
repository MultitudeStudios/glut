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

func (a *API) QuerySessions(f *flux.Flow, r *QuerySessionsRequest) ([]SessionResponse, error) {
	sessions, err := a.service.Sessions(f, &auth.SessionQuery{
		ID:             r.ID,
		Limit:          r.Limit,
		Offset:         r.Offset,
		UserID:         r.UserID,
		IncludeExpired: r.IncludeExpired,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		return nil, fmt.Errorf("api.QuerySessions: %w", err)
	}

	res := []SessionResponse{}
	for _, sess := range sessions {
		res = append(res, SessionResponse{
			ID:        sess.ID,
			Token:     sess.Token,
			UserID:    sess.UserID,
			UserIP:    sess.UserIP,
			CreatedAt: sess.CreatedAt,
			ExpiresAt: sess.ExpiresAt,
		})
	}
	return res, nil
}

func (a *API) CreateSession(f *flux.Flow, r *CreateSessionRequest) (*CreateSessionResponse, error) {
	sess, err := a.service.CreateSession(f, &auth.Credentials{
		Username: r.Username,
		Password: r.Password,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, flux.NewError("invalid_credentials", http.StatusUnauthorized, "Invalid credentials.")
		}
		if errors.Is(err, auth.ErrSessionLimit) {
			return nil, flux.NewError("session_limit", http.StatusConflict, "Session limit reached.")
		}
		return nil, err
	}

	res := &CreateSessionResponse{
		ID:        sess.ID,
		Token:     sess.Token,
		ExpiresAt: sess.ExpiresAt,
	}
	return res, nil
}

func (a *API) ClearSessions(f *flux.Flow, r *ClearSessionsRequest) (*ClearSessionsResponse, error) {
	count, err := a.service.ClearSessions(f, &auth.ClearSessionInput{
		IDs:    r.IDs,
		UserID: r.UserID,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		return nil, fmt.Errorf("api.ClearSessions: %w", err)
	}

	res := &ClearSessionsResponse{count}
	return res, nil
}

func (a *API) MySessions(f *flux.Flow, _ flux.Empty) ([]SessionResponse, error) {
	sessions, err := a.service.Sessions(f, &auth.SessionQuery{
		UserID: f.Session.User,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		return nil, fmt.Errorf("api.MySessions: %w", err)
	}

	res := []SessionResponse{}
	for _, sess := range sessions {
		res = append(res, SessionResponse{
			ID:        sess.ID,
			Token:     sess.Token,
			UserID:    sess.UserID,
			UserIP:    sess.UserIP,
			CreatedAt: sess.CreatedAt,
			ExpiresAt: sess.ExpiresAt,
		})
	}
	return res, nil
}

func (a *API) Logout(f *flux.Flow, r *LogoutRequest) (*LogoutResponse, error) {
	count, err := a.service.ClearSessions(f, &auth.ClearSessionInput{
		IDs:    r.IDs,
		UserID: f.Session.User,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		return nil, fmt.Errorf("api.Logout: %w", err)
	}

	res := &LogoutResponse{count}
	return res, nil
}

func (a *API) RenewSession(f *flux.Flow, _ flux.Empty) (*RenewSessionResponse, error) {
	newExpiry, err := a.service.RenewSession(f)
	if err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			return nil, flux.UnauthorizedError
		}
		return nil, fmt.Errorf("api.RenewSession: %w", err)
	}

	res := &RenewSessionResponse{newExpiry}
	return res, nil
}

type QuerySessionsRequest struct {
	ID             string `json:"id"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
	UserID         string `json:"user_id"`
	IncludeExpired bool   `json:"include_expired"`
}

type SessionResponse struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	UserIP    string    `json:"user_ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CreateSessionRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateSessionResponse struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ClearSessionsRequest struct {
	IDs    []string `json:"ids"`
	UserID string   `json:"user_id"`
}

type ClearSessionsResponse struct {
	Count int `json:"count"`
}

type LogoutRequest struct {
	IDs []string `json:"ids"`
}

type LogoutResponse struct {
	Count int `json:"count"`
}

type RenewSessionResponse struct {
	ExpiresAt time.Time `json:"expires_at"`
}
