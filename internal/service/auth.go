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

type UserInfoProfile struct {
	Handle string
}

type UserInfo struct {
	Sub     string
	Profile *UserInfoProfile
}

func (s *Service) GetUserInfo(
	encodedAccessToken string,
) (
	*UserInfo,
	error,
) {
	accessToken := new(tokens.AccessToken)
	if err := accessToken.Decode(encodedAccessToken, s.resourceTokenValidator); err != nil {
		return nil, fmt.Errorf("%w: couldn't decode access token: %v", ErrTokenInvalid, err)
	}

	if !slices.Contains(accessToken.Scopes(), ScopeIdentity) {
		return nil, ErrInsufficientScope
	}

	user, err := s.store.GetUserBySubject(accessToken.Subject())
	if err != nil {
		return nil, ErrAccountNotFound
	}

	userInfo := &UserInfo{Sub: accessToken.Subject()}
	if slices.Contains(accessToken.Scopes(), ScopeProfile) {
		userInfo.Profile = &UserInfoProfile{
			Handle: user.Handle,
		}
	}

	return userInfo, nil
}

func (s *Service) GrantAuthCode(
	handle string,
	secret string,
	integrationName string,
	returnTo ...string,
) (
	*url.URL,
	error,
) {
	redirectReturnTo := ""
	if len(returnTo) > 0 {
		redirectReturnTo = returnTo[0]
	}

	secretHash, err := s.store.GetSecret(handle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, handle)
		}
		return nil, fmt.Errorf("%w: failed to retrieve secret: %v", ErrInternal, err)
	}

	err = bcrypt.CompareHashAndPassword(secretHash, []byte(secret))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.store.GetUserByHandle(handle)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, handle)
	}

	integration, err := s.GetIntegration(integrationName)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrIntegrationNotFound, integrationName)
	}

	if integrationName != InternalIntegrationName {
		return nil, ErrInvalidIntegration
	}

	refreshToken, err := s.tokenIssuer.IssueRefreshToken(
		user.Subject,
		[]string{integration.Audience},
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

	redirectURL, err := parseAndValidateRedirectURL(integration.Redirect)
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
