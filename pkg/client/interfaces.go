package client

import "net/http"

// Verifier validates authorization from HTTP requests.
// Consuming projects should depend on this interface rather than *Client
// to enable testing with mock implementations.
type Verifier interface {
	VerifyAuthorization(w http.ResponseWriter, r *http.Request) (*AccessToken, error)
	VerifyAuthorizationGetCSRF(w http.ResponseWriter, r *http.Request) (*AccessToken, string, error)
	VerifyAuthorizationCheckCSRF(w http.ResponseWriter, r *http.Request, csrf string) (*AccessToken, string, error)
}

// AuthorizationCodeHandler provides the OAuth authorization code callback.
type AuthorizationCodeHandler interface {
	HandleAuthorizationCode() http.HandlerFunc
}

// AuthClient exposes both authorization verification and auth code handling.
type AuthClient interface {
	Verifier
	AuthorizationCodeHandler
}

// Compile-time check that *Client implements Verifier.
var _ Verifier = (*Client)(nil)
var _ AuthorizationCodeHandler = (*Client)(nil)
var _ AuthClient = (*Client)(nil)
