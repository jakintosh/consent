package app

import (
	"errors"
	"net/http"
	"net/url"

	"git.sr.ht/~jakintosh/consent/internal/service"
)

type loginPageData struct {
	Handle   string
	ReturnTo string
	Error    string
}

func (a *App) handleGetLogin(
	w http.ResponseWriter,
	r *http.Request,
) *appError {
	if _, err := a.auth.Verifier.VerifyAuthorization(w, r); err == nil {
		http.Redirect(w, r, loginReturnTo(r.URL.Query().Get("return_to")), http.StatusSeeOther)
		return nil
	}

	a.returnTemplate(w, r, "login.html", loginPageData{ReturnTo: loginReturnTo(r.URL.Query().Get("return_to"))})
	return nil
}

func (a *App) handlePostLogin(
	w http.ResponseWriter,
	r *http.Request,
) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errLoginFormInvalid, err)
	}

	data := loginPageData{
		Handle:   r.FormValue("handle"),
		ReturnTo: loginReturnTo(r.FormValue("return_to")),
	}

	if data.Handle == "" || r.FormValue("secret") == "" {
		w.WriteHeader(http.StatusBadRequest)
		a.returnTemplate(w, r, "login.html", loginPageData{Handle: data.Handle, ReturnTo: data.ReturnTo, Error: "Enter both your handle and secret."})
		return nil
	}

	redirectURL, err := a.service.Login(data.Handle, r.FormValue("secret"), service.InternalServiceName, data.ReturnTo)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrAccountNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
			a.returnTemplate(w, r, "login.html", loginPageData{Handle: data.Handle, ReturnTo: data.ReturnTo, Error: "Invalid handle or secret."})
			return nil
		}

		return appErr(errLoginFailed, err)
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
	return nil
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
