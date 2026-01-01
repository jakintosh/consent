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

// Compile-time check that *Client implements Verifier.
var _ Verifier = (*Client)(nil)
