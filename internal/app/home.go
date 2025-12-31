package app

import (
	"net/http"
)

func (a *App) Home() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
