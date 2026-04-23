package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Subject string
	Handle  string
	Roles   []string
}

func userFromRecord(record IdentityRecord) *User {
	roles := append([]string(nil), record.Roles...)
	return &User{
		Subject: record.Subject,
		Handle:  record.Handle,
		Roles:   roles,
	}
}

func usersFromRecords(records []IdentityRecord) []User {
	users := make([]User, 0, len(records))
	for _, record := range records {
		users = append(users, *userFromRecord(record))
	}
	return users
}

func normalizeRoles(roles []string) ([]string, error) {
	normalized := make([]string, 0, len(roles))
	for _, role := range roles {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			return nil, ErrInvalidRole
		}
		if strings.ContainsAny(trimmed, " \t\r\n\f\v") {
			return nil, ErrInvalidRole
		}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}

func (s *Service) CreateUser(
	handle string,
	password string,
	roles []string,
) (
	*User,
	error,
) {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return nil, ErrInvalidHandle
	}

	normalizedRoles, err := normalizeRoles(roles)
	if err != nil {
		return nil, err
	}

	if len(normalizedRoles) > 0 {
		if err := s.store.ValidateRoleNames(normalizedRoles); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrRoleNotFound, err)
		}
	}

	subject, err := generateSubject()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate account subject: %v", ErrInternal, err)
	}

	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), s.passwordMode.Cost())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to hash password: %v", ErrInternal, err)
	}

	err = s.store.InsertUser(subject, handle, hashPass, normalizedRoles)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrHandleExists
		}
		return nil, fmt.Errorf("%w: failed to insert account: %v", ErrInternal, err)
	}

	return &User{
		Subject: subject,
		Handle:  handle,
		Roles:   normalizedRoles,
	}, nil
}

func (s *Service) GetUser(
	subject string,
) (
	*User,
	error,
) {
	subject = strings.TrimSpace(subject)
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

	return userFromRecord(record), nil
}

func (s *Service) ListUsers() ([]User, error) {
	records, err := s.store.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list users: %v", ErrInternal, err)
	}
	return usersFromRecords(records), nil
}

func (s *Service) UpdateUser(
	subject string,
	handle *string,
	roles *[]string,
) (
	*User,
	error,
) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, ErrInvalidUser
	}

	current, err := s.store.GetUserBySubject(subject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrUserNotFound, subject)
		}
		return nil, fmt.Errorf("%w: failed to get user: %v", ErrInternal, err)
	}

	if handle != nil {
		current.Handle = strings.TrimSpace(*handle)
	}
	if roles != nil {
		current.Roles, err = normalizeRoles(*roles)
		if err != nil {
			return nil, err
		}
		if len(current.Roles) > 0 {
			if err := s.store.ValidateRoleNames(current.Roles); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrRoleNotFound, err)
			}
		}
	} else {
		current.Roles = append([]string(nil), current.Roles...)
	}

	if current.Handle == "" {
		return nil, ErrInvalidHandle
	}

	err = s.store.UpdateUser(subject, current.Handle, current.Roles)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrUserNotFound, subject)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrHandleExists
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
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ErrInvalidUser
	}

	deleted, err := s.store.DeleteUser(subject)
	if err != nil {
		return fmt.Errorf("%w: failed to delete user: %v", ErrInternal, err)
	}
	if !deleted {
		return fmt.Errorf("%w: %s", ErrUserNotFound, subject)
	}
	return nil
}

func (s *Service) Register(
	handle string,
	password string,
) error {
	_, err := s.CreateUser(handle, password, nil)
	return err
}
