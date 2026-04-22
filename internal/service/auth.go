package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"golang.org/x/crypto/bcrypt"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

type SubjectProfile struct {
	Handle string
}

type Viewer struct {
	Profile *SubjectProfile
}

func (s *Service) GetViewer(
	encodedAccessToken string,
) (
	*Viewer,
	error,
) {
	accessToken := new(tokens.AccessToken)
	if err := accessToken.Decode(encodedAccessToken, s.resourceTokenValidator); err != nil {
		return nil, fmt.Errorf("%w: couldn't decode access token: %v", ErrTokenInvalid, err)
	}

	if !slices.Contains(accessToken.Scopes(), ScopeIdentity) {
		return nil, ErrInsufficientScope
	}

	identity, err := s.store.GetIdentityBySubject(accessToken.Subject())
	if err != nil {
		return nil, ErrAccountNotFound
	}

	viewer := &Viewer{}
	if slices.Contains(accessToken.Scopes(), ScopeProfile) {
		viewer.Profile = &SubjectProfile{
			Handle: identity.Handle,
		}
	}

	return viewer, nil
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

	redirectURL, err := parseAndValidateRedirectURL(svcDef.Redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid redirect URL: %v", ErrInternal, ErrInvalidRedirect)
	}

	return buildAuthCodeRedirectURL(redirectURL, refreshToken.Encoded(), "", redirectReturnTo), nil
}

func (s *Service) RevokeRefreshToken(
	encodedRefreshToken string,
) error {
	deleted, err := s.store.DeleteRefreshToken(encodedRefreshToken)
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
