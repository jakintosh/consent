package api

import (
	"encoding/json"
	"log"
	"net/http"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type RegistrationRequest struct {
	Handle   string `json:"username"`
	Password string `json:"password"`
}

func Register(w http.ResponseWriter, r *http.Request) {

	var req RegistrationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		jsonErr(w, r)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hashPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		apiErr(r, "failed to hash password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = database.InsertAccount(req.Handle, hashPass)
	if err != nil {
		apiErr(r, "failed to insert user")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("%s %s: %v\n", r.Method, r.RequestURI, req)
	w.WriteHeader(http.StatusOK)
}
