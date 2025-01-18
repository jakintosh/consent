package client

import (
	"time"

	"git.sr.ht/~jakintosh/consent/internal/tokens"
)

type AccessToken tokens.AccessToken

func (t *AccessToken) Decode(str string) error { return (*tokens.AccessToken)(t).Decode(str) }
func (t *AccessToken) Issuer() string          { return (*tokens.AccessToken)(t).Issuer() }
func (t *AccessToken) IssuedAt() time.Time     { return (*tokens.AccessToken)(t).IssuedAt() }
func (t *AccessToken) Expiration() time.Time   { return (*tokens.AccessToken)(t).Expiration() }
func (t *AccessToken) Audience() []string      { return (*tokens.AccessToken)(t).Audience() }
func (t *AccessToken) Subject() string         { return (*tokens.AccessToken)(t).Subject() }
func (t *AccessToken) Encoded() string         { return (*tokens.AccessToken)(t).Encoded() }

type RefreshToken tokens.RefreshToken

func (t *RefreshToken) Decode(str string) error { return (*tokens.RefreshToken)(t).Decode(str) }
func (t *RefreshToken) Issuer() string          { return (*tokens.RefreshToken)(t).Issuer() }
func (t *RefreshToken) IssuedAt() time.Time     { return (*tokens.RefreshToken)(t).IssuedAt() }
func (t *RefreshToken) Expiration() time.Time   { return (*tokens.RefreshToken)(t).Expiration() }
func (t *RefreshToken) Audience() []string      { return (*tokens.RefreshToken)(t).Audience() }
func (t *RefreshToken) Subject() string         { return (*tokens.RefreshToken)(t).Subject() }
func (t *RefreshToken) Secret() string          { return (*tokens.RefreshToken)(t).Secret() }
func (t *RefreshToken) Encoded() string         { return (*tokens.RefreshToken)(t).Encoded() }
