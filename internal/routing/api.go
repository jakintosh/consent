package routing

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func buildAPIRouter(r *mux.Router) {
	api := r.PathPrefix("/api/v1/").
		Methods("POST").
		Subrouter()

	api.HandleFunc("/login", api_v1_login)
	api.HandleFunc("/logout", api_v1_logout)
	api.HandleFunc("/refresh", api_v1_refresh)
}

func api_v1_login(w http.ResponseWriter, r *http.Request) {
	log.Printf("login: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}

func api_v1_logout(w http.ResponseWriter, r *http.Request) {
	log.Printf("logout: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}

func api_v1_refresh(w http.ResponseWriter, r *http.Request) {
	log.Printf("refresh: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}
