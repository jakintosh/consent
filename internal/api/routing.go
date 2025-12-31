package api

import "github.com/gorilla/mux"

func (a *API) BuildRouter(r *mux.Router) {
	r.HandleFunc("/login", a.LoginForm()).
		Methods("POST").
		Headers("Content-Type", "application/x-www-form-urlencoded")
	r.HandleFunc("/login", a.LoginJson()).
		Methods("POST").
		Headers("Content-Type", "application/json")

	r.HandleFunc("/logout", a.Logout())
	r.HandleFunc("/refresh", a.Refresh())
	r.HandleFunc("/register", a.Register())
}
