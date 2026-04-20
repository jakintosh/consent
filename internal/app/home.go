package app

import (
	"errors"
	"net/http"
	"net/url"

	"git.sr.ht/~jakintosh/consent/pkg/client"
)

type homePageData struct {
	Authenticated bool
	LoginURL      string
	LogoutURL     string
}

func (a *App) handleGetHome(
	w http.ResponseWriter,
	r *http.Request,
) *appError {
	// get authorization
	accessToken, csrfSecret, err := a.auth.Verifier.VerifyAuthorizationGetCSRF(w, r)
	if err != nil {
		if !errors.Is(err, client.ErrTokenAbsent) {
			logAppErr(r, "failed to verify authorization: "+err.Error())
		}
	}

	// build page data
	var data homePageData
	if accessToken != nil {
		logoutUrl, err := buildLogoutURL(a.auth.LogoutURL, csrfSecret)
		if err != nil {
			return appErr(errHomeSessionUI, err)
		}
		data = homePageData{
			Authenticated: true,
			LoginURL:      a.auth.LoginURL,
			LogoutURL:     logoutUrl,
		}
	} else {
		data = homePageData{
			Authenticated: false,
			LoginURL:      a.auth.LoginURL,
		}
	}

	// render page
	a.returnTemplate(w, r, http.StatusOK, "home.html", data)
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
