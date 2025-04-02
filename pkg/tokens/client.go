package tokens

import (
	"crypto/ecdsa"
	"slices"
	"strings"
)

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
