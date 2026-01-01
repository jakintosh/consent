package testing

import "git.sr.ht/~jakintosh/consent/pkg/tokens"

// Type aliases for convenience - consumers don't need to import pkg/tokens directly.
type (
	AccessToken  = tokens.AccessToken
	RefreshToken = tokens.RefreshToken
	Issuer       = tokens.Issuer
	Validator    = tokens.Validator
)
