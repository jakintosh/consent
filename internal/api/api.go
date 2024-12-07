package api

import (
	"log"
	"net/http"
)

func jsonErr(w http.ResponseWriter, r *http.Request) {
	apiErr(r, "bad json")
	w.WriteHeader(http.StatusBadRequest)
}

func apiErr(r *http.Request, msg string) {
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
