package service

import (
	"fmt"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func (s *Service) RefreshTokens(
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

	deleted, err := s.deleteRefresh(encodedRefreshToken)
	if err != nil {
		return "", "", fmt.Errorf("%w: refresh token couldn't be deleted: %v", ErrInternal, err)
	}
	if !deleted {
		return "", "", ErrTokenNotFound
	}

	accessToken, err := s.tokenIssuer.IssueAccessToken(
		token.Subject(),
		token.Audience(),
		time.Minute*30,
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: couldn't issue access token: %v", ErrInternal, err)
	}

	newRefreshToken, err := s.tokenIssuer.IssueRefreshToken(
		token.Subject(),
		token.Audience(),
		time.Hour*72,
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: couldn't issue refresh token: %v", ErrInternal, err)
	}

	err = s.insertRefresh(
		newRefreshToken.Subject(),
		newRefreshToken.Encoded(),
		newRefreshToken.Expiration().Unix(),
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: failed to store refresh token: %v", ErrInternal, err)
	}

	return accessToken.Encoded(), newRefreshToken.Encoded(), nil
}

func (s *Service) RevokeRefreshToken(
	refreshToken string,
) error {
	deleted, err := s.deleteRefresh(refreshToken)
	if err != nil {
		return fmt.Errorf("%w: failed to delete refresh token: %v", ErrInternal, err)
	}
	if !deleted {
		return ErrTokenNotFound
	}
	return nil
}
