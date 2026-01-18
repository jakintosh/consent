package service

import (
	"fmt"
	"net/http"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"golang.org/x/crypto/bcrypt"
)

type RegistrationRequest struct {
	Handle   string `json:"username"`
	Password string `json:"password"`
}

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

	err = s.store.InsertIdentity(handle, hashPass)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrHandleExists
		}
		return fmt.Errorf("%w: failed to insert account: %v", ErrInternal, err)
	}

	return nil
}

func (s *Service) handleRegister(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[RegistrationRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = s.Register(req.Handle, req.Password)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}
