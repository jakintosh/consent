package service

import "git.sr.ht/~jakintosh/consent/pkg/tokens"

// IdentityStore handles persistence of user identity data
type IdentityStore interface {
	InsertIdentity(handle string, secret []byte) error
	GetSecret(handle string) ([]byte, error)
}

// RefreshStore handles persistence of refresh tokens
type RefreshStore interface {
	InsertRefreshToken(token *tokens.RefreshToken) error
	DeleteRefreshToken(jwt string) (deleted bool, err error)
	GetRefreshTokenOwner(jwt string) (handle string, err error)
}
