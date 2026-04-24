package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const (
	ProtectedAdminRoleName    = "admin"
	ProtectedAdminRoleDisplay = "Administrator"
)

type Role struct {
	Name    string
	Display string
}

type RoleUpdate struct {
	Display *string
}

func SeedSystemRoles(
	store Store,
) error {
	if store == nil {
		return fmt.Errorf("service: store required")
	}

	if _, err := store.GetRole(ProtectedAdminRoleName); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("service: check existing admin role: %w", err)
		}

		if err := store.InsertRole(ProtectedAdminRoleName, ProtectedAdminRoleDisplay); err != nil {
			return fmt.Errorf("service: create admin role: %w", err)
		}
	}

	return nil
}

func (s *Service) CreateRole(
	name string,
	display string,
) (
	*Role,
	error,
) {
	if name == "" {
		return nil, ErrInvalidHandle
	}
	if name == ProtectedAdminRoleName {
		return nil, ErrRoleProtected
	}

	if display == "" {
		return nil, ErrInvalidHandle
	}

	err := s.store.InsertRole(name, display)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrRoleExists
		}
		return nil, fmt.Errorf("%w: failed to insert role: %v", ErrInternal, err)
	}

	return &Role{
		Name:    name,
		Display: display,
	}, nil
}

func (s *Service) GetRole(
	name string,
) (
	*Role,
	error,
) {
	if name == "" {
		return nil, ErrInvalidHandle
	}

	record, err := s.store.GetRole(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, name)
		}
		return nil, fmt.Errorf("%w: failed to get role: %v", ErrInternal, err)
	}

	return &record, nil
}

func (s *Service) UpdateRole(
	name string,
	display *string,
) (
	*Role,
	error,
) {
	if name == "" {
		return nil, ErrInvalidHandle
	}
	if name == ProtectedAdminRoleName {
		return nil, ErrRoleProtected
	}

	updates := &RoleUpdate{
		Display: display,
	}

	err := s.store.UpdateRole(name, updates)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, name)
		}
		return nil, fmt.Errorf("%w: failed to update role: %v", ErrInternal, err)
	}

	if display != nil {
		if *display == "" {
			return nil, ErrInvalidHandle
		}
	}

	current, err := s.store.GetRole(name)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get role: %v", ErrInternal, err)
	}

	return &current, nil
}

func (s *Service) DeleteRole(
	name string,
) error {
	if name == "" {
		return ErrInvalidHandle
	}
	if name == ProtectedAdminRoleName {
		return ErrRoleProtected
	}

	deleted, err := s.store.DeleteRole(name)
	if err != nil {
		if isFKConstraintError(err) {
			return fmt.Errorf("%w: %s", ErrRoleInUse, name)
		}
		return fmt.Errorf("%w: failed to delete role: %v", ErrInternal, err)
	}
	if !deleted {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, name)
	}
	return nil
}

func (s *Service) ListRoles() (
	[]Role,
	error,
) {
	records, err := s.store.ListRoles()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list roles: %v", ErrInternal, err)
	}
	return records, nil
}

func isFKConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}
