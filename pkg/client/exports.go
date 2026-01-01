package client

import (
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

type TokenClient = tokens.Client
type TokenServer = tokens.Server

type TokenIssuer = tokens.Issuer
type TokenValidator = tokens.Validator

type AccessToken = tokens.AccessToken
type RefreshToken = tokens.RefreshToken
