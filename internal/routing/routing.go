package routing

import (
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/app"
	"github.com/gorilla/mux"
)

func BuildRouter() *mux.Router {
	r := mux.NewRouter()

	// ui routes
	r.HandleFunc("/login", app.Login)

	// router for api
	s := r.PathPrefix("/api/").
		Methods("POST").
		Subrouter()

	s.HandleFunc("/login", api.LoginForm).
		Methods("POST").
		Headers("Content-Type", "application/x-www-form-urlencoded")
	s.HandleFunc("/login", api.LoginJson).
		Methods("POST").
		Headers("Content-Type", "application/json")

	s.HandleFunc("/logout", api.Logout)
	s.HandleFunc("/refresh", api.Refresh)
	s.HandleFunc("/register", api.Register)

	return r
}
