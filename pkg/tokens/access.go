package tokens

import (
	"log"
	"strings"
	"time"
)

// ==============================================

// AccessTokenClaims represents the JWT claims for an access token.
// It contains standard JWT claims (exp, iat, iss, aud, sub) and sits between
// the JSON representation in the token and the AccessToken Go struct.
type AccessTokenClaims struct {
	Expiration int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
	Issuer     string `json:"iss"`
	Audience   string `json:"aud"`
	Subject    string `json:"sub"`
}

func (claims *AccessTokenClaims) validate(validator Validator) error {
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

// AccessToken represents a short-lived JWT token used for API authorization.
// Access tokens are issued by the consent server and validated by backend applications.
// They contain the user's identity (subject) and the intended application (audience).
//
// Access tokens are typically valid for a short duration (e.g., 1 hour) and should be
// stored in HTTP-only cookies or authorization headers.
type AccessToken struct {
	issuer     string
	issuedAt   time.Time
	expiration time.Time
	audience   []string
	subject    string
	encoded    string
}

func (t *AccessToken) Issuer() string        { return t.issuer }
func (t *AccessToken) IssuedAt() time.Time   { return t.issuedAt }
func (t *AccessToken) Expiration() time.Time { return t.expiration }
func (t *AccessToken) Audience() []string    { return t.audience }
func (t *AccessToken) Subject() string       { return t.subject }
func (t *AccessToken) Encoded() string       { return t.encoded }

func (token *AccessToken) Decode(encToken string, validator Validator) error {
	claims, err := decodeToken[*AccessTokenClaims](encToken, validator)
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

func (token *AccessToken) intoClaims() *AccessTokenClaims {
	claims := &AccessTokenClaims{}
	claims.Issuer = token.issuer
	claims.IssuedAt = token.issuedAt.Unix()
	claims.Expiration = token.expiration.Unix()
	claims.Audience = strings.Join(token.audience, " ")
	claims.Subject = token.subject
	return claims
}

func (token *AccessToken) fromClaims(claims *AccessTokenClaims, encToken string) {
	token.issuer = claims.Issuer
	token.issuedAt = time.Unix(claims.IssuedAt, 0)
	token.expiration = time.Unix(claims.Expiration, 0)
	token.audience = strings.Split(claims.Audience, " ")
	token.subject = claims.Subject
	token.encoded = encToken
}
