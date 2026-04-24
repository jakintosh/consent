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

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
