package service

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Register(
	handle string,
	password string,
) error {
	if handle == "" {
		return ErrInvalidHandle
	}

	subject, err := generateSubject()
	if err != nil {
		return fmt.Errorf("%w: failed to generate account subject: %v", ErrInternal, err)
	}

	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), s.passwordMode.Cost())
	if err != nil {
		return fmt.Errorf("%w: failed to hash password: %v", ErrInternal, err)
	}

	err = s.store.InsertIdentity(subject, handle, hashPass)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrHandleExists
		}
		return fmt.Errorf("%w: failed to insert account: %v", ErrInternal, err)
	}

	return nil
}
