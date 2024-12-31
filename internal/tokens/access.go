package tokens

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"
)

// ==============================================

type AccessTokenClaims struct {
	Expiration int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
	Issuer     string `json:"iss"`
	Audience   string `json:"aud"`
	Subject    string `json:"sub"`
	Secret     string `json:"secret"`
}

func (claims *AccessTokenClaims) validate() error {
	now := time.Now()

	if claims.Issuer != _issuerDomain {
		return fmt.Errorf("invalid issuer")
	}

	if time.Unix(claims.IssuedAt, 0).After(now) {
		return fmt.Errorf("not valid yet")
	}

	if time.Unix(claims.Expiration, 0).Before(now) {
		return fmt.Errorf("expired")
	}

	if _validAudience != nil {
		audiences := strings.Split(claims.Audience, " ")
		if !slices.Contains(audiences, *_validAudience) {
			return fmt.Errorf("no valid audience")
		}
	}

	return nil
}

func (claims *AccessTokenClaims) fromToken(token *AccessToken) *AccessTokenClaims {
	claims.Issuer = token.issuer
	claims.IssuedAt = token.issuedAt.Unix()
	claims.Expiration = token.expiration.Unix()
	claims.Audience = strings.Join(token.audience, " ")
	claims.Subject = token.subject
	claims.Secret = token.secret

	return claims
}

// ==============================================

type AccessToken struct {
	issuer     string
	issuedAt   time.Time
	expiration time.Time
	audience   []string
	subject    string
	secret     string
	encoded    string
}

func (t *AccessToken) Issuer() string        { return t.issuer }
func (t *AccessToken) IssuedAt() time.Time   { return t.issuedAt }
func (t *AccessToken) Expiration() time.Time { return t.expiration }
func (t *AccessToken) Audience() []string    { return t.audience }
func (t *AccessToken) Subject() string       { return t.subject }
func (t *AccessToken) Secret() string        { return t.secret }
func (t *AccessToken) Encoded() string       { return t.encoded }

func (token *AccessToken) Decode(encToken string) error {
	claims := AccessTokenClaims{}
	if err := validateToken(encToken, &claims); err != nil {
		if true {
			log.Println(err.Context())
		}
		return err
	}
	token.fromClaims(&claims, encToken)
	return nil
}

func (token *AccessToken) fromClaims(claims *AccessTokenClaims, encToken string) {
	token.issuer = claims.Issuer
	token.issuedAt = time.Unix(claims.IssuedAt, 0)
	token.expiration = time.Unix(claims.Expiration, 0)
	token.audience = strings.Split(claims.Audience, " ")
	token.subject = claims.Subject
	token.secret = claims.Secret
	token.encoded = encToken
}

// ==============================================

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
		issuer:     _issuerDomain,
		issuedAt:   now,
		expiration: now.Add(lifetime),
		audience:   audience,
		subject:    subject,
		secret:     secret,
	}

	// encode
	claims := new(AccessTokenClaims).fromToken(token)
	if encodedToken, err := encodeToken(claims); err != nil {
		return nil, fmt.Errorf("failed to encode access token: %v", err)
	} else {
		token.encoded = encodedToken
	}

	return token, nil
}
