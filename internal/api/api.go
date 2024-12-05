package api

import (
	"log"
	"net/http"
)

func logErr(r *http.Request, msg string) {
	log.Printf("%s %s: %s\n", r.Method, r.RequestURI, msg)
}

func Login(w http.ResponseWriter, r *http.Request) {
	log.Printf("login: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	log.Printf("logout: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	log.Printf("refresh: %s %s\n", r.Method, r.RequestURI)
	w.WriteHeader(http.StatusOK)
}
