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
	returnTo := sanitizeReturnTo(r.URL.Query().Get("return_to"))

	_, err := a.auth.Verifier.VerifyAuthorization(w, r)
	if err == nil {
		http.Redirect(w, r, returnTo, http.StatusSeeOther)
		return nil
	}

	a.returnTemplate(w, r, http.StatusOK, "login.html", loginPageData{
		ReturnTo: returnTo,
	})
	return nil
}

func (a *App) handlePostLogin(
	w http.ResponseWriter,
	r *http.Request,
) *appError {

	// parse input
	if err := r.ParseForm(); err != nil {
		return appErr(errLoginFormInvalid, err)
	}
	returnTo := sanitizeReturnTo(r.FormValue("return_to"))
	handle := r.FormValue("handle")
	secret := r.FormValue("secret")

	// validate input
	if handle == "" || secret == "" {
		w.WriteHeader(http.StatusBadRequest)
		a.returnTemplate(w, r, http.StatusUnauthorized, "login.html", loginPageData{
			Handle:   handle,
			ReturnTo: returnTo,
			Error:    "Enter both your handle and secret.",
		})
		return nil
	}

	// call service
	redirectURL, err := a.service.Login(handle, secret, service.InternalServiceName, returnTo)
	if err != nil {
		// handle errors
		switch {
		case errors.Is(err, service.ErrInvalidCredentials),
			errors.Is(err, service.ErrAccountNotFound):
			w.WriteHeader(http.StatusUnauthorized)
			a.returnTemplate(w, r, http.StatusUnauthorized, "login.html", loginPageData{
				Handle:   handle,
				ReturnTo: returnTo,
				Error:    "Invalid handle or secret.",
			})
			return nil
		default:
			return appErr(errLoginFailed, err)
		}
	}

	// intended outcome
	http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
	return nil
}

func sanitizeReturnTo(
	returnTo string,
) string {
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
