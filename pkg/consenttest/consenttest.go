package consenttest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// Keys holds cryptographic keys for testing.
type Keys struct {
	SigningKey      *ecdsa.PrivateKey
	VerificationKey *ecdsa.PublicKey
	IssuerDomain    string
}

// Session holds token strings and metadata for a test session.
type Session struct {
	AccessToken      string
	RefreshToken     string
	CSRF             string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// CookieOptions configures cookie attributes for test cookies.
type CookieOptions struct {
	Secure   bool
	SameSite http.SameSite
	Path     string
	MaxAge   int // if 0, derived from token expiration
}

// NewKeys generates a new ECDSA P-256 keypair for testing.
func NewKeys(issuerDomain string) (*Keys, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	return &Keys{
		SigningKey:      privateKey,
		VerificationKey: &privateKey.PublicKey,
		IssuerDomain:    issuerDomain,
	}, nil
}

// NewSession creates a new test session with minted access and refresh tokens.
func NewSession(keys *Keys, subject, audience string, accessLifetime, refreshLifetime time.Duration) (*Session, error) {
	// Initialize server-side issuer using the test keys
	issuer, _ := tokens.InitServer(keys.SigningKey, keys.IssuerDomain)

	// Issue access token
	audiences := []string{audience}
	accessToken, err := issuer.IssueAccessToken(subject, audiences, accessLifetime)
	if err != nil {
		return nil, err
	}

	// Issue refresh token
	refreshToken, err := issuer.IssueRefreshToken(subject, audiences, refreshLifetime)
	if err != nil {
		return nil, err
	}

	return &Session{
		AccessToken:      accessToken.Encoded(),
		RefreshToken:     refreshToken.Encoded(),
		CSRF:             refreshToken.Secret(),
		AccessExpiresAt:  accessToken.Expiration(),
		RefreshExpiresAt: refreshToken.Expiration(),
	}, nil
}

// Cookies creates HTTP cookies for the session tokens.
func Cookies(sess *Session, opts CookieOptions) (access, refresh *http.Cookie) {
	now := time.Now()

	accessMaxAge := opts.MaxAge
	if accessMaxAge == 0 {
		accessMaxAge = int(sess.AccessExpiresAt.Sub(now).Seconds())
	}

	refreshMaxAge := opts.MaxAge
	if refreshMaxAge == 0 {
		refreshMaxAge = int(sess.RefreshExpiresAt.Sub(now).Seconds())
	}

	path := opts.Path
	if path == "" {
		path = "/"
	}

	access = &http.Cookie{
		Name:     "accessToken",
		Value:    sess.AccessToken,
		Path:     path,
		MaxAge:   accessMaxAge,
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.SameSite,
	}

	refresh = &http.Cookie{
		Name:     "refreshToken",
		Value:    sess.RefreshToken,
		Path:     path,
		MaxAge:   refreshMaxAge,
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.SameSite,
	}

	return access, refresh
}

// AddCookies adds session cookies to an HTTP request.
func AddCookies(r *http.Request, sess *Session, opts CookieOptions) {
	access, refresh := Cookies(sess, opts)
	r.AddCookie(access)
	r.AddCookie(refresh)
}

// Validator creates a token validator for the given keys and audience.
func Validator(keys *Keys, audience string) tokens.Validator {
	return tokens.InitClient(keys.VerificationKey, keys.IssuerDomain, audience)
}
