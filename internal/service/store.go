package service

import "git.sr.ht/~jakintosh/consent/pkg/tokens"

// Store handles persistence of identity data and refresh tokens.
type Store interface {
	InsertIdentity(handle string, secret []byte) error
	GetSecret(handle string) ([]byte, error)

	InsertRefreshToken(token *tokens.RefreshToken) error
	DeleteRefreshToken(jwt string) (deleted bool, err error)
	GetRefreshTokenOwner(jwt string) (handle string, err error)
}
