package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"git.sr.ht/~jakintosh/consent/pkg/client"
)

func (a *App) Home() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Authenticated": false,
			"LoginURL":      a.auth.LoginURL,
		}

		_, csrfSecret, err := a.auth.Verifier.VerifyAuthorizationGetCSRF(w, r)
		if err != nil {
			if !errors.Is(err, client.ErrTokenAbsent) {
				logAppErr(r, fmt.Sprintf("failed to verify authorization: %v", err))
			}
		} else {
			data["Authenticated"] = true
			logoutURL, err := buildLogoutURL(a.auth.LogoutURL, csrfSecret)
			if err != nil {
				logAppErr(r, fmt.Sprintf("failed to build logout URL: %v", err))
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(serverErrorHTML)
				return
			}
			data["LogoutURL"] = logoutURL
		}

		a.returnTemplate("home.html", data, w, r)
	}
}

func buildLogoutURL(logoutURL string, csrfSecret string) (string, error) {
	parsed, err := url.Parse(logoutURL)
	if err != nil {
		return "", err
	}

	queries := parsed.Query()
	queries.Set("csrf", csrfSecret)
	parsed.RawQuery = queries.Encode()

	return parsed.String(), nil
}
