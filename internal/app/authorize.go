package app

import (
	"errors"
	"net/http"
	"net/url"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/pkg/client"
)

type authorizePageData struct {
	IntegrationName    string
	IntegrationDisplay string
	RequestedScopes    []service.ScopeDefinition
	GrantedScopes      []service.ScopeDefinition
	MissingScopes      []service.ScopeDefinition
	State              string
	CSRF               string
}

func (a *App) handleGetAuthorize(
	w http.ResponseWriter,
	r *http.Request,
) *appError {
	// parse query params
	svcName := r.URL.Query().Get("integration")
	scopes := r.URL.Query()["scope"]
	state := r.URL.Query().Get("state")

	// get auth status
	accessToken, csrf, err := a.auth.Verifier.VerifyAuthorizationGetCSRF(w, r)
	if err != nil {
		if !errors.Is(err, client.ErrTokenAbsent) {
			logAppErr(r, "failed to verify authorization: "+err.Error())
		}
		http.Redirect(w, r, a.loginReturnToURL(r), http.StatusSeeOther)
		return nil
	}
	sub := accessToken.Subject()

	// get a review of what needs to be authorized
	review, err := a.service.ReviewAuthorizationRequest(sub, svcName, scopes, state)
	if err != nil {
		return appErr(errAuthorizePrepare, err)
	}

	switch review.NeedsApproval() {
	case true: // render authorize page
		a.returnTemplate(w, r, http.StatusOK, "authorize.html", authorizePageData{
			IntegrationName:    review.Request.Integration.Name,
			IntegrationDisplay: review.Request.Integration.Display,
			RequestedScopes:    review.RequestedScopes,
			GrantedScopes:      review.GrantedScopes,
			MissingScopes:      review.MissingScopes,
			State:              review.Request.State,
			CSRF:               csrf,
		})

	case false: // try auto-approve and redirect
		redirectURL, err := a.service.ApproveAuthorization(sub, review)
		if err != nil {
			return appErr(errAuthorizeAutoApprove, err)
		}
		http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
	}

	return nil
}

func (a *App) handlePostAuthorize(
	w http.ResponseWriter,
	r *http.Request,
) *appError {
	// parse form values
	if err := r.ParseForm(); err != nil {
		return appErr(errAuthorizeFormInvalid, err)
	}
	csrf := r.FormValue("csrf")
	action := r.FormValue("action")
	scopes := r.Form["scope"]
	state := r.FormValue("state")
	svc := r.FormValue("integration")

	// validate user
	accessToken, _, err := a.auth.Verifier.VerifyAuthorizationCheckCSRF(w, r, csrf)
	if err != nil {
		if errors.Is(err, client.ErrCSRFInvalid) {
			return appErr(errAuthorizeCSRFExpired, err)
		}
		if !errors.Is(err, client.ErrTokenAbsent) {
			logAppErr(r, "failed to verify authorization submit: "+err.Error())
		}
		http.Redirect(w, r, a.loginReturnToURL(r), http.StatusSeeOther)
		return nil
	}
	sub := accessToken.Subject()

	// review auth request
	review, err := a.service.ReviewAuthorizationRequest(sub, svc, scopes, state)
	if err != nil {
		return appErr(errAuthorizeSubmitInvalid, err)
	}

	// handle action and redirect
	switch action {
	case "approve":
		redirectURL, err := a.service.ApproveAuthorization(sub, review)
		if err != nil {
			return appErr(errAuthorizeApprove, err)
		}
		http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)

	case "deny":
		redirectURL, err := a.service.DenyAuthorization(review)
		if err != nil {
			return appErr(errAuthorizeDeny, err)
		}
		http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)

	default:
		return appErr(errAuthorizeActionMissing, nil)
	}

	return nil
}

func (a *App) loginReturnToURL(
	r *http.Request,
) string {
	loginURL, err := url.Parse(a.auth.LoginURL)
	if err != nil || loginURL == nil {
		return "/login?return_to=" + url.QueryEscape(r.URL.RequestURI())
	}
	query := loginURL.Query()
	query.Set("return_to", r.URL.RequestURI())
	loginURL.RawQuery = query.Encode()
	return loginURL.String()
}
