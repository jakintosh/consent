package app

import (
	"net/http"
	"net/url"
)

func (a *App) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := a.auth.Verifier.VerifyAuthorization(w, r); err == nil {
			http.Redirect(w, r, loginReturnTo(r.URL.Query().Get("return_to")), http.StatusSeeOther)
			return
		}

		data := map[string]string{
			"ReturnTo": r.URL.Query().Get("return_to"),
		}

		a.returnTemplate("login.html", data, w, r)
	}
}

func loginReturnTo(returnTo string) string {
	if returnTo == "" {
		return "/"
	}
	parsed, err := url.Parse(returnTo)
	if err != nil ||
		parsed == nil ||
		parsed.IsAbs() ||
		parsed.Host != "" ||
		parsed.Path == "" ||
		parsed.Path[0] != '/' {
		return "/"
	}
	return parsed.String()
}
