package api

import "net/http"

func (a *API) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", a.Login())
	mux.HandleFunc("POST /logout", a.Logout())
	mux.HandleFunc("POST /refresh", a.Refresh())
	mux.HandleFunc("POST /register", a.Register())
	return mux
}
