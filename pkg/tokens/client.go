package tokens

import (
	"crypto/ecdsa"
	"slices"
	"strings"
)

// Client implements the Validator interface for backend applications.
// It holds the consent server's public key for signature verification and enforces
// that tokens are intended for this specific application (audience checking).
// Create a Client instance using InitClient.
type Client struct {
	verificationKey *ecdsa.PublicKey
	issuerDomain    string
	validAudience   string
}

//
// Validator interface

func (client *Client) VerifySignature(
	encHeader string,
	encClaims string,
	encSignature string,
) error {
	return verifySignature(
		encHeader,
		encClaims,
		encSignature,
		client.verificationKey,
	)
}

func (client *Client) ShouldValidateAudience() bool {
	return true
}

func (client *Client) ValidateDomain(issuerDomain string) bool {
	return issuerDomain == client.issuerDomain
}

func (client *Client) ValidateAudiences(audience string) bool {
	audiences := strings.Split(audience, " ")
	return slices.Contains(audiences, client.validAudience)
}
