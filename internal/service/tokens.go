package service

import (
	"fmt"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

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

func (s *Service) RevokeRefreshToken(
	refreshToken string,
) error {
	deleted, err := s.store.DeleteRefreshToken(refreshToken)
	if err != nil {
		return fmt.Errorf("%w: failed to delete refresh token: %v", ErrInternal, err)
	}
	if !deleted {
		return ErrTokenNotFound
	}
	return nil
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

	accessToken, refreshToken, err := s.RefreshTokens(req.RefreshToken)
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
