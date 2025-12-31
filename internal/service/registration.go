package service

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Register(
	handle string,
	password string,
) error {
	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("%w: failed to hash password: %v", ErrInternal, err)
	}

	err = s.insertAccount(handle, hashPass)
	if err != nil {
		return fmt.Errorf("%w: failed to insert account: %v", ErrInternal, err)
	}

	return nil
}
