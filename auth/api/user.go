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

func (a *API) QueryUsers(f *flux.Flow, r *QueryUsersRequest) ([]UserResponse, error) {
	users, err := a.service.Users(f, &auth.UserQuery{
		ID:     r.ID,
		Limit:  r.Limit,
		Offset: r.Offset,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		if errors.Is(err, auth.ErrUserNotFound) {
			return nil, flux.NotFoundError("User not found.")
		}
		return nil, fmt.Errorf("api.QueryUsers: %w", err)
	}

	res := []UserResponse{}
	for _, user := range users {
		res = append(res, UserResponse{
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
	return res, nil
}

func (a *API) CreateUser(f *flux.Flow, r *CreateUserRequest) (*UserResponse, error) {
	user, err := a.service.CreateUser(f, &auth.NewUserInput{
		Username: r.Username,
		Email:    r.Email,
		Password: r.Password,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		if errors.Is(err, auth.ErrUserExists) {
			return nil, flux.NewError("user_exists", http.StatusConflict, "User already exists.")
		}
		return nil, fmt.Errorf("api.CreateUser: %w", err)
	}

	res := &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
	return res, nil
}

func (a *API) DeleteUsers(f *flux.Flow, r *DeleteUsersRequest) (*DeleteUsersResponse, error) {
	count, err := a.service.DeleteUsers(f, &auth.DeleteUsersInput{
		IDs: r.IDs,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		return nil, fmt.Errorf("api.DeleteUsers: %w", err)
	}

	res := &DeleteUsersResponse{count}
	return res, nil
}

func (a *API) MyUser(f *flux.Flow, _ flux.Empty) (*UserResponse, error) {
	users, err := a.service.Users(f, &auth.UserQuery{
		ID: f.Session.User,
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		return nil, fmt.Errorf("api.MyUser: %w", err)
	}
	user := users[0]
	res := &UserResponse{
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
	return res, nil
}

func (a *API) DeleteMyUser(f *flux.Flow, _ flux.Empty) (flux.Empty, error) {
	_, err := a.service.DeleteUsers(f, &auth.DeleteUsersInput{
		IDs: []string{f.Session.User},
	})
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
		}
		return nil, fmt.Errorf("api.DeleteMyUser: %w", err)
	}
	return nil, nil
}

type QueryUsersRequest struct {
	ID     string `json:"id"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type UserResponse struct {
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

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type DeleteUsersRequest struct {
	IDs []string `json:"ids"`
}

type DeleteUsersResponse struct {
	Count int `json:"count"`
}
