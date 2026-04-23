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
	*RoleDefinition,
	error,
) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidHandle
	}
	if name == ProtectedAdminRoleName {
		return nil, ErrRoleProtected
	}

	if strings.TrimSpace(display) == "" {
		return nil, ErrInvalidHandle
	}

	err := s.store.InsertRole(name, display)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrRoleExists
		}
		return nil, fmt.Errorf("%w: failed to insert role: %v", ErrInternal, err)
	}

	return &RoleDefinition{
		Name:    name,
		Display: display,
	}, nil
}

func (s *Service) GetRole(
	name string,
) (
	*RoleDefinition,
	error,
) {
	name = strings.TrimSpace(name)
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
	*RoleDefinition,
	error,
) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidHandle
	}
	if name == ProtectedAdminRoleName {
		return nil, ErrRoleProtected
	}

	current, err := s.store.GetRole(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, name)
		}
		return nil, fmt.Errorf("%w: failed to get role: %v", ErrInternal, err)
	}

	if display != nil {
		current.Display = strings.TrimSpace(*display)
	}

	if current.Display == "" {
		return nil, ErrInvalidHandle
	}

	err = s.store.UpdateRoleDisplay(name, current.Display)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, name)
		}
		return nil, fmt.Errorf("%w: failed to update role: %v", ErrInternal, err)
	}

	return &current, nil
}

func (s *Service) DeleteRole(
	name string,
) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidHandle
	}
	if name == ProtectedAdminRoleName {
		return ErrRoleProtected
	}

	count, err := s.store.CountUsersWithRole(name)
	if err != nil {
		return fmt.Errorf("%w: failed to count users with role: %v", ErrInternal, err)
	}
	if count > 0 {
		return fmt.Errorf("%w: %s", ErrRoleInUse, name)
	}

	deleted, err := s.store.DeleteRole(name)
	if err != nil {
		return fmt.Errorf("%w: failed to delete role: %v", ErrInternal, err)
	}
	if !deleted {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, name)
	}
	return nil
}

func (s *Service) ListRoles() (
	[]RoleDefinition,
	error,
) {
	records, err := s.store.ListRoles()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list roles: %v", ErrInternal, err)
	}
	return records, nil
}
