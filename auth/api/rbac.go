package api

import (
	"errors"
	"fmt"
	"glut/auth"
	"glut/common/flux"
	"glut/common/valid"
	"net/http"
)

func queryRoles(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.RoleQuery
		if err := f.Bind(&in); err != nil {
			return err
		}

		roles, err := s.Roles(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrRoleNotFound) {
				return flux.NotFoundError("Role not found.")
			}
			return fmt.Errorf("api.queryRoles: %w", err)
		}
		return f.Respond(http.StatusOK, roles)
	}
}

func createRole(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.CreateRoleInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		role, err := s.CreateRole(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrRoleExists) {
				return flux.ExistsError("Role already exists.")
			}
			return fmt.Errorf("api.createRole: %w", err)
		}
		return f.Respond(http.StatusOK, role)
	}
}

func updateRole(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.UpdateRoleInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		if err := s.UpdateRole(f, in); err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrRoleNotFound) {
				return flux.NotFoundError("Role not found.")
			}
			if errors.Is(err, auth.ErrRoleExists) {
				return flux.ExistsError("Role already exists.")
			}
			return fmt.Errorf("api.updateRole: %w", err)
		}
		return f.Respond(http.StatusOK, nil)
	}
}

func deleteRole(s *auth.Service) flux.HandlerFunc {
	type response struct {
		Count int `json:"count"`
	}

	return func(f *flux.Flow) error {
		var in auth.DeleteRoleInput
		if err := f.Bind(&in); err != nil {
			return err
		}

		count, err := s.DeleteRole(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			return fmt.Errorf("api.deleteRole: %w", err)
		}
		return f.Respond(http.StatusOK, &response{count})
	}
}

func queryPermissions(s *auth.Service) flux.HandlerFunc {
	return func(f *flux.Flow) error {
		var in auth.PermissionQuery
		if err := f.Bind(&in); err != nil {
			return err
		}

		permissions, err := s.Permissions(f, in)
		if err != nil {
			if verr, ok := err.(valid.Errors); ok {
				return flux.ValidationError(verr)
			}
			if errors.Is(err, auth.ErrPermissionNotFound) {
				return flux.NotFoundError("Permission not found.")
			}
			return fmt.Errorf("api.queryPermissions: %w", err)
		}
		return f.Respond(http.StatusOK, permissions)
	}
}
