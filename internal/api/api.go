package api

import (
	"crypto/ecdsa"
	"encoding/json"
	"log"
	"net/http"
)

var signingKey *ecdsa.PrivateKey

func Init(privateKey *ecdsa.PrivateKey) {
	signingKey = privateKey
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

func Logout(w http.ResponseWriter, r *http.Request) {
	log.Printf("logout: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	log.Printf("refresh: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}
