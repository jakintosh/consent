package api

import "github.com/gorilla/mux"

func BuildRouter(r *mux.Router) {

	r.HandleFunc("/login", LoginForm).
		Methods("POST").
		Headers("Content-Type", "application/x-www-form-urlencoded")
	r.HandleFunc("/login", LoginJson).
		Methods("POST").
		Headers("Content-Type", "application/json")

	r.HandleFunc("/logout", Logout)
	r.HandleFunc("/refresh", Refresh)
	r.HandleFunc("/register", Register)
}
