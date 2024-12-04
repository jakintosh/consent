package routing

import (
	"log"
	"net/http"
)

func api_login(w http.ResponseWriter, r *http.Request) {
	log.Printf("login: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}

func api_logout(w http.ResponseWriter, r *http.Request) {
	log.Printf("logout: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}

func api_refresh(w http.ResponseWriter, r *http.Request) {
	log.Printf("refresh: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}
