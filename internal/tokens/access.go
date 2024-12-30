package tokens

import (
	"fmt"
	"strings"
	"time"
)

type AccessToken struct {
	issuer     string
	issuedAt   time.Time
	expiration time.Time
	audience   []string
	subject    string
	secret     string
	encoded    string
}

type AccessTokenClaims struct {
	Expiration int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
	Issuer     string `json:"iss"`
	Audience   string `json:"aud"`
	Subject    string `json:"sub"`
	Secret     string `json:"secret"`
}

func (t *AccessToken) Issuer() string {
	return t.issuer
}

func (t *AccessToken) IssuedAt() time.Time {
	return t.issuedAt
}

func (t *AccessToken) Expiration() time.Time {
	return t.expiration
}

func (t *AccessToken) Audience() []string {
	return t.audience
}

func (t *AccessToken) Subject() string {
	return t.subject
}

func (t *AccessToken) Secret() string {
	return t.secret
}

func (t *AccessToken) Encoded() string {
	return t.encoded
}

func IssueAccessToken(
	subject string,
	audience []string,
	lifetime time.Duration,
) (*AccessToken, error) {
	now := time.Now()
	secret, err := generateCSRFCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate csrf secret: %v", err)
	}
	token := &AccessToken{
		issuer:     issuerDomain,
		issuedAt:   now,
		expiration: now.Add(lifetime),
		audience:   audience,
		subject:    subject,
		secret:     secret,
	}

	// encode
	claims := new(AccessTokenClaims).FromAccessToken(token)
	if encodedToken, err := encodeToken(claims); err != nil {
		return nil, fmt.Errorf("failed to encode access token: %v", err)
	} else {
		token.encoded = encodedToken
	}

	return token, nil
}

func (token *AccessToken) Decode(tokenStr string) error {
	claims := new(AccessTokenClaims)

	if err := validateToken(tokenStr, claims); err != nil {
		return fmt.Errorf("failed to validate token: %v", err)
	}

	if err := token.FromClaims(claims); err != nil {
		return fmt.Errorf("failed to validate access token claims: %v", err)
	}

	return nil
}

func (token *AccessToken) FromClaims(claims *AccessTokenClaims) error {
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
	token.secret = claims.Secret

	return nil
}

func (claims *AccessTokenClaims) FromAccessToken(token *AccessToken) *AccessTokenClaims {
	claims.Issuer = token.issuer
	claims.IssuedAt = token.issuedAt.Unix()
	claims.Expiration = token.expiration.Unix()
	claims.Audience = strings.Join(token.audience, " ")
	claims.Subject = token.subject
	claims.Secret = token.secret

	return claims
}
