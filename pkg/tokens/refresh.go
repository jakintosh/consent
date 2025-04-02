package tokens

import (
	"log"
	"strings"
	"time"
)

// RefreshTokenClaims is a struct that represents the claims section of a JWT for the refresh token.
// It sits between the JSON representation in the token and the [RefreshToken] Go struct.
// It can be validated against module level _issuerDomain, _validAudience, and current time.
// It implements the `validate()` function as part of the [claims] interface.
type RefreshTokenClaims struct {
	Expiration int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
	Issuer     string `json:"iss"`
	Audience   string `json:"aud"`
	Subject    string `json:"sub"`
	Secret     string `json:"secret"`
}

func (claims *RefreshTokenClaims) validate(validator Validator) error {
	now := time.Now()

	if time.Unix(claims.IssuedAt, 0).After(now) {
		return ErrTokenNotIssued()
	}

	if time.Unix(claims.Expiration, 0).Before(now) {
		return ErrTokenExpired()
	}

	if !validator.ValidateDomain(claims.Issuer) {
		return ErrTokenInvalidIssuer()
	}

	if validator.ShouldValidateAudience() {
		if !validator.ValidateAudiences(claims.Audience) {
			return ErrTokenInvalidAudience()
		}
	}

	return nil
}

// ==============================================

type RefreshToken struct {
	issuer     string
	issuedAt   time.Time
	expiration time.Time
	audience   []string
	subject    string
	secret     string
	encoded    string
}

func (t *RefreshToken) Issuer() string        { return t.issuer }
func (t *RefreshToken) IssuedAt() time.Time   { return t.issuedAt }
func (t *RefreshToken) Expiration() time.Time { return t.expiration }
func (t *RefreshToken) Audience() []string    { return t.audience }
func (t *RefreshToken) Subject() string       { return t.subject }
func (t *RefreshToken) Secret() string        { return t.secret }
func (t *RefreshToken) Encoded() string       { return t.encoded }

func (token *RefreshToken) Decode(encToken string, validator Validator) error {
	claims, err := decodeToken[*RefreshTokenClaims](encToken, validator)
	if err != nil {
		if true {
			// TODO: make this actually check log level
			log.Println(err.Context())
		}
		return err
	}
	token.fromClaims(*claims, encToken)
	return nil
}

func (token *RefreshToken) intoClaims() *RefreshTokenClaims {
	claims := &RefreshTokenClaims{}
	claims.Issuer = token.issuer
	claims.IssuedAt = token.issuedAt.Unix()
	claims.Expiration = token.expiration.Unix()
	claims.Audience = strings.Join(token.audience, " ")
	claims.Subject = token.subject
	claims.Secret = token.secret
	return claims
}

func (token *RefreshToken) fromClaims(claims *RefreshTokenClaims, encToken string) {
	token.issuer = claims.Issuer
	token.issuedAt = time.Unix(claims.IssuedAt, 0)
	token.expiration = time.Unix(claims.Expiration, 0)
	token.audience = strings.Split(claims.Audience, " ")
	token.subject = claims.Subject
	token.secret = claims.Secret
	token.encoded = encToken
}
