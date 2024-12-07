package routing

import (
	"git.sr.ht/~jakintosh/consent/internal/api"
	"github.com/gorilla/mux"
)

func BuildRouter() *mux.Router {
	r := mux.NewRouter()

	// rout for api
	s := r.PathPrefix("/api/").
		Methods("POST").
		Subrouter()
	s.HandleFunc("/login", api.Login)
	s.HandleFunc("/logout", api.Logout)
	s.HandleFunc("/refresh", api.Refresh)
	s.HandleFunc("/register", api.Register)

	return r
}
