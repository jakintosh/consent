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

	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), s.passwordMode.Cost())
	if err != nil {
		return fmt.Errorf("%w: failed to hash password: %v", ErrInternal, err)
	}

	err = s.identityStore.InsertIdentity(handle, hashPass)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrHandleExists
		}
		return fmt.Errorf("%w: failed to insert account: %v", ErrInternal, err)
	}

	return nil
}
