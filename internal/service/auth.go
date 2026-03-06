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
	Handle  string `json:"handle"`
	Secret  string `json:"secret"`
	Service string `json:"service"`
}

type LoginResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func (s *Service) Login(
	handle string,
	secret string,
	serviceName string,
) (
	*url.URL,
	error,
) {
	hash, err := s.store.GetSecret(handle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, handle)
		}
		return nil, fmt.Errorf("%w: failed to retrieve secret: %v", ErrInternal, err)
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(secret))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	svcDef, err := s.GetServiceByName(serviceName)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, serviceName)
	}

	refreshToken, err := s.tokenIssuer.IssueRefreshToken(
		handle,
		[]string{svcDef.Audience},
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

	return buildRedirectURL(redirectURL, refreshToken.Encoded()), nil
}

func buildRedirectURL(
	redirect *url.URL,
	refreshToken string,
) *url.URL {
	redirectURL := *redirect
	q := redirectURL.Query()
	q.Set("auth_code", refreshToken)
	redirectURL.RawQuery = q.Encode()
	return &redirectURL
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	switch r.Header.Get("Content-Type") {
	case "application/x-www-form-urlencoded":
		req = LoginRequest{
			Handle:  r.FormValue("handle"),
			Secret:  r.FormValue("secret"),
			Service: r.FormValue("service"),
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

	redirectURL, err := s.Login(req.Handle, req.Secret, req.Service)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
}
