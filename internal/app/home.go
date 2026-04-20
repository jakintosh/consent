package app

import (
	"errors"
	"net/http"
	"net/url"

	"git.sr.ht/~jakintosh/consent/pkg/client"
)

func (a *App) handleGetHome(
	w http.ResponseWriter,
	r *http.Request,
) *appError {
	data := map[string]any{
		"Authenticated": false,
		"LoginURL":      a.auth.LoginURL,
	}

	_, csrfSecret, err := a.auth.Verifier.VerifyAuthorizationGetCSRF(w, r)
	if err != nil {
		if !errors.Is(err, client.ErrTokenAbsent) {
			logAppErr(r, "failed to verify authorization: "+err.Error())
		}
	} else {
		data["Authenticated"] = true
		logoutURL, err := buildLogoutURL(a.auth.LogoutURL, csrfSecret)
		if err != nil {
			return appErr(errHomeSessionUI, err)
		}
		data["LogoutURL"] = logoutURL
	}

	a.returnTemplate(w, r, "home.html", data)
	return nil
}

func buildLogoutURL(
	logoutURL string,
	csrfSecret string,
) (
	string,
	error,
) {
	parsed, err := url.Parse(logoutURL)
	if err != nil {
		return "", err
	}

	queries := parsed.Query()
	queries.Set("csrf", csrfSecret)
	parsed.RawQuery = queries.Encode()

	return parsed.String(), nil
}
