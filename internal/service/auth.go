package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"golang.org/x/crypto/bcrypt"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// ── Login ──

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

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

type MeResponse struct {
	Profile *MeProfile `json:"profile,omitempty"`
}

type MeProfile struct {
	Handle string `json:"handle"`
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

func (s *Service) RevokeRefreshToken(refreshToken string) error {
	deleted, err := s.store.DeleteRefreshToken(refreshToken)
	if err != nil {
		return fmt.Errorf("%w: failed to delete refresh token: %v", ErrInternal, err)
	}
	if !deleted {
		return ErrTokenNotFound
	}
	return nil
}
func (s *Service) RefreshAccessToken(
	encodedRefreshToken string,
) (
	string,
	string,
	error,
) {
	token := tokens.RefreshToken{}
	if err := token.Decode(encodedRefreshToken, s.tokenValidator); err != nil {
		return "", "", fmt.Errorf("%w: couldn't decode refresh token: %v", ErrTokenInvalid, err)
	}

	deleted, err := s.store.DeleteRefreshToken(encodedRefreshToken)
	if err != nil {
		return "", "", fmt.Errorf("%w: refresh token couldn't be deleted: %v", ErrInternal, err)
	}
	if !deleted {
		return "", "", ErrTokenNotFound
	}

	accessToken, err := s.tokenIssuer.IssueAccessToken(
		token.Subject(),
		token.Audience(),
		token.Scopes(),
		time.Minute*30,
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: couldn't issue access token: %v", ErrInternal, err)
	}

	newRefreshToken, err := s.tokenIssuer.IssueRefreshToken(
		token.Subject(),
		token.Audience(),
		token.Scopes(),
		time.Hour*72,
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: couldn't issue refresh token: %v", ErrInternal, err)
	}

	err = s.store.InsertRefreshToken(newRefreshToken)
	if err != nil {
		return "", "", fmt.Errorf("%w: failed to store refresh token: %v", ErrInternal, err)
	}

	return accessToken.Encoded(), newRefreshToken.Encoded(), nil
}
func (s *Service) handleLogin(
	w http.ResponseWriter,
	r *http.Request,
) {
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

func (s *Service) handleLogout(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[LogoutRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = s.RevokeRefreshToken(req.RefreshToken)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (s *Service) handleRefresh(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[RefreshRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	accessToken, refreshToken, err := s.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	response := RefreshResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}
	wire.WriteData(w, http.StatusOK, response)
}

func (s *Service) handleMe(
	w http.ResponseWriter,
	r *http.Request,
) {
	encodedToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		wire.WriteError(w, httpStatusFromError(ErrTokenInvalid), ErrTokenInvalid.Error())
		return
	}

	accessToken := new(tokens.AccessToken)
	if err := accessToken.Decode(encodedToken, s.resourceTokenValidator); err != nil {
		wire.WriteError(w, httpStatusFromError(ErrTokenInvalid), fmt.Sprintf("%v: couldn't decode access token: %v", ErrTokenInvalid, err))
	}

	if !slices.Contains(accessToken.Scopes(), ScopeIdentity) {
		wire.WriteError(w, httpStatusFromError(ErrInsufficientScope), ErrInsufficientScope.Error())
		return
	}

	identity, err := s.store.GetIdentityBySubject(accessToken.Subject())
	if err != nil {
		wire.WriteError(w, httpStatusFromError(ErrAccountNotFound), ErrAccountNotFound.Error())
		return
	}

	response := MeResponse{}
	if slices.Contains(accessToken.Scopes(), ScopeProfile) {
		response.Profile = &MeProfile{
			Handle: identity.Handle,
		}
	}

	wire.WriteData(w, http.StatusOK, response)
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

func bearerToken(
	header string,
) (
	string,
	bool,
) {
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	encodedToken := strings.TrimSpace(parts[1])
	if encodedToken == "" {
		return "", false
	}
	return encodedToken, true
}
