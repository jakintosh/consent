package service

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Subject string
	Handle  string
	Roles   []string
}

type UserUpdate struct {
	Handle *string
	Roles  *[]string
}

func (s *Service) CreateUser(
	handle string,
	password string,
	roles []string,
) (
	*User,
	error,
) {
	if handle == "" {
		return nil, ErrInvalidHandle
	}

	if err := s.validateRequiredRoles(roles); err != nil {
		return nil, err
	}

	subject, err := generateSubject()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate account subject: %v", ErrInternal, err)
	}

	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), s.passwordMode.Cost())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to hash password: %v", ErrInternal, err)
	}

	err = s.store.InsertUser(subject, handle, hashPass, roles)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrHandleExists
		}
		return nil, fmt.Errorf("%w: failed to insert account: %v", ErrInternal, err)
	}

	return &User{
		Subject: subject,
		Handle:  handle,
		Roles:   roles,
	}, nil
}

func (s *Service) GetUser(
	subject string,
) (
	*User,
	error,
) {
	if subject == "" {
		return nil, ErrInvalidUser
	}

	record, err := s.store.GetUserBySubject(subject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrUserNotFound, subject)
		}
		return nil, fmt.Errorf("%w: failed to get user: %v", ErrInternal, err)
	}

	return record, nil
}

func (s *Service) ListUsers() (
	[]User,
	error,
) {
	records, err := s.store.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list users: %v", ErrInternal, err)
	}
	return records, nil
}

func (s *Service) UpdateUser(
	subject string,
	updates *UserUpdate,
) (
	*User,
	error,
) {
	if subject == "" {
		return nil, ErrInvalidUser
	}
	if updates == nil {
		return nil, ErrInvalidUpdate
	}

	current, err := s.store.GetUserBySubject(subject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrUserNotFound, subject)
		}
		return nil, fmt.Errorf("%w: failed to get user: %v", ErrInternal, err)
	}

	if updates.Handle != nil {
		current.Handle = *updates.Handle
	}
	if updates.Roles != nil {
		if err := s.validateRequiredRoles(*updates.Roles); err != nil {
			return nil, err
		}
		current.Roles = *updates.Roles
	}

	if current.Handle == "" {
		return nil, ErrInvalidHandle
	}

	err = s.store.UpdateUser(subject, current.Handle, current.Roles)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrUserNotFound, subject)
		}
		if isUniqueConstraintError(err) {
			return nil, ErrHandleExists
		}
		if errors.Is(err, ErrLastAdmin) {
			return nil, ErrLastAdmin
		}
		return nil, fmt.Errorf("%w: failed to update user: %v", ErrInternal, err)
	}

	return &User{
		Subject: subject,
		Handle:  current.Handle,
		Roles:   append([]string(nil), current.Roles...),
	}, nil
}

func (s *Service) DeleteUser(
	subject string,
) error {
	if subject == "" {
		return ErrInvalidUser
	}

	deleted, err := s.store.DeleteUser(subject)
	if err != nil {
		if errors.Is(err, ErrLastAdmin) {
			return ErrLastAdmin
		}
		return fmt.Errorf("%w: failed to delete user: %v", ErrInternal, err)
	}
	if !deleted {
		return fmt.Errorf("%w: %s", ErrUserNotFound, subject)
	}
	return nil
}

func (s *Service) UserHasRole(
	subject string,
	role string,
) bool {
	if subject == "" || role == "" {
		return false
	}

	user, err := s.store.GetUserBySubject(subject)
	if err != nil {
		return false
	}

	return slices.Contains(user.Roles, role)
}

// UserHasAnyRole returns true if the user has at least one of the given roles,
// or if no roles are passed.
func (s *Service) UserHasAnyRole(
	subject string,
	roles []string,
) bool {

	// must have subject
	if subject == "" {
		return false
	}

	// but if roles are empty, is true
	if len(roles) == 0 {
		return true
	}

	user, err := s.store.GetUserBySubject(subject)
	if err != nil {
		return false
	}

	for _, role := range roles {
		if slices.Contains(user.Roles, role) {
			return true
		}
	}
	return false
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
