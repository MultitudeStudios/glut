package api

import (
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
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
