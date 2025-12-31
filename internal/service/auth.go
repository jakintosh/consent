package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Login(
	handle string,
	secret string,
	serviceName string,
) (
	*url.URL,
	error,
) {
	if err := s.authenticate(handle, secret); err != nil {
		return nil, err
	}

	svcDef, err := s.catalog.GetService(serviceName)
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

	err = s.insertRefresh(
		refreshToken.Subject(),
		refreshToken.Encoded(),
		refreshToken.Expiration().Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	redirectURL := buildRedirectURL(svcDef.Redirect, refreshToken.Encoded())

	return redirectURL, nil
}

func (s *Service) authenticate(
	handle string,
	secret string,
) error {
	hash, err := s.getSecret(handle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %s", ErrAccountNotFound, handle)
		}
		return fmt.Errorf("%w: failed to retrieve secret: %v", ErrInternal, err)
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(secret))
	if err != nil {
		return ErrInvalidCredentials
	}

	return nil
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
