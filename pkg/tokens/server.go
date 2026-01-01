package tokens

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"time"
)

// Server implements both Issuer and Validator interfaces for the consent auth server.
// It holds the private signing key for issuing tokens and the corresponding public key
// for verification. Create a Server instance using InitServer.
type Server struct {
	signingKey      *ecdsa.PrivateKey
	verificationKey *ecdsa.PublicKey
	issuerDomain    string
}

//
// Issuer interface

func (server *Server) SignHash(hash []byte) (string, error) {
	r, s, err := ecdsa.Sign(rand.Reader, server.signingKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %v", err)
	}
	encSignature, err := encodeSignature(r, s)
	if err != nil {
		return "", fmt.Errorf("failed to encode signature: %v", err)
	}
	return encSignature, nil
}

func (server *Server) IssueRefreshToken(
	subject string,
	audience []string,
	lifetime time.Duration,
) (*RefreshToken, error) {

	now := time.Now()
	exp := now.Add(lifetime)
	secret, err := generateCSRFCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate csrf secret: %v", err)
	}
	token := &RefreshToken{
		issuer:     server.issuerDomain,
		issuedAt:   now,
		expiration: exp,
		audience:   audience,
		subject:    subject,
		secret:     secret,
	}

	claims := token.intoClaims()
	encToken, err := encodeToken(claims, server)
	if err != nil {
		return nil, fmt.Errorf("failed to encode refresh token: %v", err)
	}
	token.encoded = encToken

	return token, nil
}

func (server *Server) IssueAccessToken(
	subject string,
	audience []string,
	lifetime time.Duration,
) (*AccessToken, error) {

	now := time.Now()
	exp := now.Add(lifetime)
	token := &AccessToken{
		issuer:     server.issuerDomain,
		issuedAt:   now,
		expiration: exp,
		audience:   audience,
		subject:    subject,
	}

	claims := token.intoClaims()
	encodedToken, err := encodeToken(claims, server)
	if err != nil {
		return nil, fmt.Errorf("failed to encode access token: %v", err)
	}
	token.encoded = encodedToken

	return token, nil
}

//
// Validator interface

func (server *Server) VerifySignature(
	encHeader string,
	encClaims string,
	encSignature string,
) error {
	return verifySignature(
		encHeader,
		encClaims,
		encSignature,
		server.verificationKey,
	)
}
func (server *Server) ShouldValidateAudience() bool {
	return false
}

func (server *Server) ValidateDomain(issuerDomain string) bool {
	return issuerDomain == server.issuerDomain
}
func (server *Server) ValidateAudiences(audience string) bool {
	return false
}
