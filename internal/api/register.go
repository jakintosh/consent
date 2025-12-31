package api

import (
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type RegistrationRequest struct {
	Handle   string `json:"username"`
	Password string `json:"password"`
}

func (a *API) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegistrationRequest
		if ok := decodeRequest(&req, w, r); !ok {
			return
		}

		hashPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			logApiErr(r, fmt.Sprintf("failed to hash password: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = insertAccount(a.db, req.Handle, hashPass)
		if err != nil {
			logApiErr(r, fmt.Sprintf("failed to insert user: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
