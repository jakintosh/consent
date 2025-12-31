package api

import (
	"encoding/json"
	"log"
	"net/http"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

var (
	services       *Services
	tokenIssuer    tokens.Issuer
	tokenValidator tokens.Validator
)

func Init(
	i tokens.Issuer,
	v tokens.Validator,
	s *Services,
	dbPath string,
) {
	tokenIssuer = i
	tokenValidator = v
	services = s
	initDatabase(dbPath)
}

func decodeRequest[T any](req *T, w http.ResponseWriter, r *http.Request) bool {
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logApiErr(r, "bad json request")
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	return true
}

func returnJson(data any, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func logApiErr(r *http.Request, msg string) {
	log.Printf("%s %s: %s\n", r.Method, r.RequestURI, msg)
}
