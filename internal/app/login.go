package app

import (
	"net/http"
)

func (a *App) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"ReturnTo": r.URL.Query().Get("return_to"),
		}

		a.returnTemplate("login.html", data, w, r)
	}
}
