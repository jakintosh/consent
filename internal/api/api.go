package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"git.sr.ht/~jakintosh/consent/internal/service"
)

type API struct {
	service *service.Service
}

func New(svc *service.Service) *API {
	return &API{
		service: svc,
	}
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

func httpStatusFromError(err error) int {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrAccountNotFound):
		return http.StatusUnauthorized
	case errors.Is(err, service.ErrServiceNotFound),
		errors.Is(err, service.ErrTokenInvalid),
		errors.Is(err, service.ErrTokenNotFound),
		errors.Is(err, service.ErrInvalidHandle):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrHandleExists):
		return http.StatusConflict
	case errors.Is(err, service.ErrInternal):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	logApiErr(r, err.Error())
	w.WriteHeader(httpStatusFromError(err))
}
