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
	q := &auth.UserQuery{
		ID:     r.ID,
		Limit:  r.Limit,
		Offset: r.Offset,
	}
	users, err := a.service.Users(f, q)
	if err != nil {
		if verr, ok := err.(valid.Errors); ok {
			return nil, flux.ValidationError(verr)
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
	in := &auth.NewUserInput{
		Username: r.Username,
		Email:    r.Email,
		Password: r.Password,
	}
	user, err := a.service.CreateUser(f, in)
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
	in := &auth.DeleteUsersInput{
		IDs: r.IDs,
	}
	count, err := a.service.DeleteUsers(f, in)
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
	sess := f.Session()
	if sess == nil {
		return nil, flux.UnauthorizedError
	}
	q := &auth.UserQuery{
		ID: sess.User,
	}
	users, err := a.service.Users(f, q)
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
	sess := f.Session()
	if sess == nil {
		return nil, flux.UnauthorizedError
	}
	in := &auth.DeleteUsersInput{
		IDs: []string{sess.User},
	}
	_, err := a.service.DeleteUsers(f, in)
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
