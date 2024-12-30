package tokens

import (
	"fmt"
	"log"
	"strings"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/database"
)

type RefreshTokenClaims struct {
	Expiration int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
	Issuer     string `json:"iss"`
	Audience   string `json:"aud"`
	Subject    string `json:"sub"`
}

type RefreshToken struct {
	issuer     string
	issuedAt   time.Time
	expiration time.Time
	audience   []string
	subject    string
	encoded    string
}

func (t *RefreshToken) Issuer() string {
	return t.issuer
}

func (t *RefreshToken) IssuedAt() time.Time {
	return t.issuedAt
}

func (t *RefreshToken) Expiration() time.Time {
	return t.expiration
}

func (t *RefreshToken) Audience() []string {
	return t.audience
}

func (t *RefreshToken) Subject() string {
	return t.subject
}

func (t *RefreshToken) Encoded() string {
	return t.encoded
}

func IssueRefreshToken(
	subject string,
	audience []string,
	lifetime time.Duration,
) (*RefreshToken, error) {
	now := time.Now()
	token := &RefreshToken{
		issuer:     issuerDomain,
		issuedAt:   now,
		expiration: now.Add(lifetime),
		audience:   audience,
		subject:    subject,
	}

	// encode
	claims := new(RefreshTokenClaims).FromRefreshToken(token)
	encodedToken, err := encodeToken(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to encode refresh token: %v", err)
	}
	token.encoded = encodedToken

	// store
	err = database.InsertRefresh(subject, encodedToken, token.expiration.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to insert refresh token: %v", err)
	}

	log.Printf("issuing refresh token:\n%s\n", encodedToken)

	return token, nil
}

func (token *RefreshToken) Decode(tokenStr string) error {
	claims := new(RefreshTokenClaims)

	if err := validateToken(tokenStr, claims); err != nil {
		return fmt.Errorf("validation failure: %v", err)
	}

	if err := token.FromClaims(claims); err != nil {
		return fmt.Errorf("failed to validate refresh token claims: %v", err)
	}

	return nil
}

func (token *RefreshToken) FromClaims(claims *RefreshTokenClaims) error {
	now := time.Now()

	token.issuer = claims.Issuer
	if token.issuer != issuerDomain {
		return fmt.Errorf("invalid issuer")
	}

	token.issuedAt = time.Unix(claims.IssuedAt, 0)
	if token.issuedAt.After(now) {
		return fmt.Errorf("not valid yet")
	}

	token.expiration = time.Unix(claims.Expiration, 0)
	if token.expiration.Before(now) {
		return fmt.Errorf("expired")
	}

	token.audience = strings.Split(claims.Audience, " ")
	token.subject = claims.Subject

	return nil
}

func (claims *RefreshTokenClaims) FromRefreshToken(token *RefreshToken) *RefreshTokenClaims {
	claims.Issuer = token.issuer
	claims.IssuedAt = token.issuedAt.Unix()
	claims.Expiration = token.expiration.Unix()
	claims.Audience = strings.Join(token.audience, " ")
	claims.Subject = token.subject

	return claims
}
