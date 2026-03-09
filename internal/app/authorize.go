package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/pkg/client"
)

type authorizePageData struct {
	ServiceName    string
	ServiceDisplay string
	Scopes         []service.ScopeDefinition
	GrantedScopes  []string
	MissingScopes  []string
	State          string
	CSRF           string
}

func (a *App) Authorize() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := a.service.PrepareAuthorizationRequest(
			r.URL.Query().Get("service"),
			r.URL.Query()["scope"],
			r.URL.Query().Get("state"),
		)
		if err != nil {
			logAppErr(r, fmt.Sprintf("invalid authorization request: %v", err))
			w.WriteHeader(http.StatusBadRequest)
			w.Write(badRequestHTML)
			return
		}

		accessToken, csrf, err := a.auth.Verifier.VerifyAuthorizationGetCSRF(w, r)
		if err != nil {
			if !errors.Is(err, client.ErrTokenAbsent) {
				logAppErr(r, fmt.Sprintf("failed to verify authorization: %v", err))
			}
			http.Redirect(w, r, a.loginReturnToURL(r), http.StatusSeeOther)
			return
		}

		decision, err := a.service.GetAuthorizationDecision(accessToken.Subject(), req)
		if err != nil {
			logAppErr(r, fmt.Sprintf("failed to prepare authorization decision: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(serverErrorHTML)
			return
		}

		if len(decision.MissingScopes) == 0 {
			redirectURL, err := a.service.IssueAuthorizationCodeRedirect(accessToken.Subject(), req.Service.Name, req.Scopes, req.State)
			if err != nil {
				logAppErr(r, fmt.Sprintf("failed to issue authorization code: %v", err))
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(serverErrorHTML)
				return
			}
			http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
			return
		}

		data := authorizePageData{
			ServiceName:    req.Service.Name,
			ServiceDisplay: req.Service.Display,
			Scopes:         service.ScopeDefinitions(req.Scopes),
			GrantedScopes:  decision.GrantedScopes,
			MissingScopes:  decision.MissingScopes,
			State:          req.State,
			CSRF:           csrf,
		}
		a.returnTemplate("authorize.html", data, w, r)
	}
}

func (a *App) AuthorizeSubmit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logAppErr(r, fmt.Sprintf("failed to parse authorize form: %v", err))
			w.WriteHeader(http.StatusBadRequest)
			w.Write(badRequestHTML)
			return
		}

		req, err := a.service.PrepareAuthorizationRequest(
			r.FormValue("service"),
			r.Form["scope"],
			r.FormValue("state"),
		)
		if err != nil {
			logAppErr(r, fmt.Sprintf("invalid authorization submit: %v", err))
			w.WriteHeader(http.StatusBadRequest)
			w.Write(badRequestHTML)
			return
		}

		accessToken, _, err := a.auth.Verifier.VerifyAuthorizationCheckCSRF(w, r, r.FormValue("csrf"))
		if err != nil {
			if errors.Is(err, client.ErrCSRFInvalid) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if !errors.Is(err, client.ErrTokenAbsent) {
				logAppErr(r, fmt.Sprintf("failed to verify authorization submit: %v", err))
			}
			http.Redirect(w, r, a.loginReturnToURL(r), http.StatusSeeOther)
			return
		}

		action := r.FormValue("action")
		if action == "deny" {
			redirectURL, err := a.service.IssueAuthorizationDeniedRedirect(req.Service.Name, req.State)
			if err != nil {
				logAppErr(r, fmt.Sprintf("failed to deny authorization: %v", err))
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(serverErrorHTML)
				return
			}
			http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
			return
		}
		if action != "approve" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(badRequestHTML)
			return
		}

		decision, err := a.service.GetAuthorizationDecision(accessToken.Subject(), req)
		if err != nil {
			logAppErr(r, fmt.Sprintf("failed to compute authorization decision: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(serverErrorHTML)
			return
		}

		redirectURL, err := a.service.ApproveAuthorization(accessToken.Subject(), decision)
		if err != nil {
			logAppErr(r, fmt.Sprintf("failed to approve authorization: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(serverErrorHTML)
			return
		}

		http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
	}
}

func (a *App) loginReturnToURL(r *http.Request) string {
	loginURL, err := url.Parse(a.auth.LoginURL)
	if err != nil || loginURL == nil {
		return "/login?return_to=" + url.QueryEscape(r.URL.RequestURI())
	}
	query := loginURL.Query()
	query.Set("return_to", r.URL.RequestURI())
	loginURL.RawQuery = query.Encode()
	return loginURL.String()
}
