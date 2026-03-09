package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Handle   string `json:"handle"`
	Secret   string `json:"secret"`
	Service  string `json:"service"`
	ReturnTo string `json:"returnTo"`
}

type LoginResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func (s *Service) Login(
	handle string,
	secret string,
	serviceName string,
	returnTo ...string,
) (
	*url.URL,
	error,
) {
	redirectReturnTo := ""
	if len(returnTo) > 0 {
		redirectReturnTo = returnTo[0]
	}

	identity, err := s.store.GetIdentityByHandle(handle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, handle)
		}
		return nil, fmt.Errorf("%w: failed to retrieve secret: %v", ErrInternal, err)
	}

	err = bcrypt.CompareHashAndPassword(identity.Secret, []byte(secret))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	svcDef, err := s.GetServiceByName(serviceName)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, serviceName)
	}

	if serviceName != InternalServiceName {
		return nil, ErrInvalidService
	}

	refreshToken, err := s.tokenIssuer.IssueRefreshToken(
		identity.Subject,
		[]string{svcDef.Audience},
		nil,
		time.Second*10,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to issue refresh token: %v", ErrInternal, err)
	}

	err = s.store.InsertRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	redirectURL, err := parseFullURL(svcDef.Redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid redirect URL: %v", ErrInternal, ErrInvalidRedirect)
	}

	return buildAuthCodeRedirectURL(redirectURL, refreshToken.Encoded(), "", redirectReturnTo), nil
}

func buildAuthCodeRedirectURL(
	redirect *url.URL,
	refreshToken string,
	state string,
	returnTo string,
) *url.URL {
	redirectURL := *redirect
	q := redirectURL.Query()
	q.Set("auth_code", refreshToken)
	if state != "" {
		q.Set("state", state)
	}
	if returnTo != "" {
		q.Set("return_to", returnTo)
	}
	redirectURL.RawQuery = q.Encode()
	return &redirectURL
}

func buildAuthorizationErrorRedirectURL(
	redirect *url.URL,
	errorCode string,
	state string,
) *url.URL {
	redirectURL := *redirect
	q := redirectURL.Query()
	q.Set("error", errorCode)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()
	return &redirectURL
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	switch r.Header.Get("Content-Type") {
	case "application/x-www-form-urlencoded":
		req = LoginRequest{
			Handle:   r.FormValue("handle"),
			Secret:   r.FormValue("secret"),
			Service:  r.FormValue("service"),
			ReturnTo: r.FormValue("return_to"),
		}
		if req.Handle == "" || req.Secret == "" || req.Service == "" {
			wire.WriteError(w, http.StatusBadRequest, "Missing form fields")
			return
		}
	case "application/json":
		var err error
		if req, err = decodeRequest[LoginRequest](r); err != nil {
			wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
			return
		}
	default:
		wire.WriteError(w, http.StatusUnsupportedMediaType, "Unsupported content type")
		return
	}

	redirectURL, err := s.Login(req.Handle, req.Secret, req.Service, req.ReturnTo)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
}
