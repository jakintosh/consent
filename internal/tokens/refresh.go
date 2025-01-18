package tokens

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	db "git.sr.ht/~jakintosh/consent/internal/database"
)

// ==============================================

type RefreshTokenClaims struct {
	Expiration int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
	Issuer     string `json:"iss"`
	Audience   string `json:"aud"`
	Subject    string `json:"sub"`
	Secret     string `json:"secret"`
}

func (claims *RefreshTokenClaims) validate() error {
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

func (claims *RefreshTokenClaims) fromToken(token *RefreshToken) *RefreshTokenClaims {
	claims.Issuer = token.issuer
	claims.IssuedAt = token.issuedAt.Unix()
	claims.Expiration = token.expiration.Unix()
	claims.Audience = strings.Join(token.audience, " ")
	claims.Subject = token.subject
	claims.Secret = token.secret

	return claims
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

func (token *RefreshToken) Decode(encToken string) error {
	claims := RefreshTokenClaims{}
	if err := validateToken(encToken, &claims); err != nil {
		if true {
			// TODO: make this actually check log level
			log.Println(err.Context())
		}
		return err
	}
	token.fromClaims(&claims, encToken)
	return nil
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

// ==============================================

func IssueRefreshToken(
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
		issuer:     _issuerDomain,
		issuedAt:   now,
		expiration: exp,
		audience:   audience,
		subject:    subject,
		secret:     secret,
	}

	// encode
	claims := new(RefreshTokenClaims).fromToken(token)
	encToken, err := encodeToken(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to encode refresh token: %v", err)
	}
	token.encoded = encToken

	// store
	if err = db.InsertRefresh(subject, encToken, exp.Unix()); err != nil {
		return nil, fmt.Errorf("failed to insert refresh token: %v", err)
	}

	return token, nil
}
