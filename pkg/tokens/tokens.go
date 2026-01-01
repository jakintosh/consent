package tokens

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type validateError struct {
	context string
	err     error
}

func (t *validateError) Context() string {
	return t.context
}
func (t *validateError) Error() string {
	return fmt.Sprintf("%v", t.err)
}

var (
	errTokenMalformed       = errors.New("token malformed")
	errTokenBadSignature    = errors.New("token bad signature")
	errTokenInvalidAudience = errors.New("token invalid audience")
	errTokenInvalidIssuer   = errors.New("token invalid issuer")
	errTokenExpired         = errors.New("token expired")
	errTokenNotIssued       = errors.New("token not issued yet")
)

// ErrTokenMalformed returns an error indicating the token structure is invalid or cannot be parsed.
func ErrTokenMalformed() error { return errTokenMalformed }

// ErrTokenBadSignature returns an error indicating the token's signature verification failed.
func ErrTokenBadSignature() error { return errTokenBadSignature }

// ErrTokenInvalidAudience returns an error indicating the token's audience claim doesn't match the expected value.
func ErrTokenInvalidAudience() error { return errTokenInvalidAudience }

// ErrTokenInvalidIssuer returns an error indicating the token's issuer claim doesn't match the expected value.
func ErrTokenInvalidIssuer() error { return errTokenInvalidIssuer }

// ErrTokenExpired returns an error indicating the token has passed its expiration time.
func ErrTokenExpired() error { return errTokenExpired }

// ErrTokenNotIssued returns an error indicating the token's issued-at time is in the future.
func ErrTokenNotIssued() error { return errTokenNotIssued }

// Issuer can issue new tokens by signing them with a private key.
// This interface is implemented by Server, which has access to the signing key.
type Issuer interface {
	SignHash([]byte) (string, error)
	IssueRefreshToken(string, []string, time.Duration) (*RefreshToken, error)
	IssueAccessToken(string, []string, time.Duration) (*AccessToken, error)
}

// Validator can validate tokens by verifying signatures with a public key.
// This interface is implemented by both Server and Client.
// Server validates tokens without checking audience (since it issued them).
// Client validates tokens and enforces audience matching.
type Validator interface {
	ShouldValidateAudience() bool
	ValidateDomain(string) bool
	ValidateAudiences(string) bool
	VerifySignature(string, string, string) error
}

// InitServer creates a token issuer and validator for the consent auth server.
// The returned Issuer can create new tokens signed with the private key.
// The returned Validator can verify tokens but does not enforce audience checks.
//
// Parameters:
//   - signingKey: ECDSA private key used to sign tokens
//   - issuerDomain: Domain name of the consent server (e.g., "consent.example.com")
//
// Returns both an Issuer and Validator interface backed by the same Server instance.
func InitServer(
	signingKey *ecdsa.PrivateKey,
	issuerDomain string,
) (
	Issuer,
	Validator,
) {
	server := &Server{
		signingKey:      signingKey,
		verificationKey: &signingKey.PublicKey,
		issuerDomain:    issuerDomain,
	}
	return server, server
}

// InitClient creates a token validator for backend applications integrating with consent.
// The returned Validator can verify token signatures and enforces audience matching.
//
// Parameters:
//   - verificationKey: ECDSA public key from the consent server (for signature verification)
//   - issuerDomain: Expected issuer domain (must match tokens' "iss" claim)
//   - validAudience: Your application's identifier (must be in tokens' "aud" claim)
//
// Returns a Validator that rejects tokens not intended for this application.
func InitClient(
	verificationKey *ecdsa.PublicKey,
	issuerDomain string,
	validAudience string,
) Validator {
	return &Client{
		verificationKey: verificationKey,
		issuerDomain:    issuerDomain,
		validAudience:   validAudience,
	}
}

type JWTHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type claims interface {
	validate(Validator) error
	comparable
}

func newES256JWTHeader() JWTHeader {
	return JWTHeader{
		Algorithm: "ES256",
		Type:      "JWT",
	}
}

func generateCSRFCode() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random CSRF bytes: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func buildMessage(encHeader string, encClaims string) string {
	return fmt.Sprintf("%s.%s", encHeader, encClaims)
}

func hashMessage(message string) []byte {
	hash := sha256.Sum256([]byte(message))
	return hash[:]
}

func encodeSignature(r *big.Int, s *big.Int) (string, error) {
	signature := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	// Right-align r in first 32 bytes (padding with zeros on the left)
	copy(signature[32-len(rBytes):32], rBytes)
	// Right-align s in second 32 bytes (padding with zeros on the left)
	copy(signature[64-len(sBytes):64], sBytes)
	encSignature := base64.RawURLEncoding.EncodeToString(signature)
	return encSignature, nil
}

func decodeSignature(signature []byte) (*big.Int, *big.Int, error) {
	if len(signature) != 64 {
		return nil, nil, fmt.Errorf("invalid signature length")
	}
	r := new(big.Int).SetBytes(signature[00:32])
	s := new(big.Int).SetBytes(signature[32:64])
	return r, s, nil
}

func encodeJWTSection[T comparable](section T) (string, error) {
	sectionJSON, err := json.Marshal(section)
	if err != nil {
		return "", fmt.Errorf("json marshal failure: %v", err)
	}
	encodedSection := base64.RawURLEncoding.EncodeToString([]byte(sectionJSON))
	return encodedSection, nil
}

func encodeMessage[T comparable](claims T) (string, error) {
	encHeader, err := encodeJWTSection(newES256JWTHeader())
	if err != nil {
		return "", fmt.Errorf("failed to encode header: %v", err)
	}
	encClaims, err := encodeJWTSection(claims)
	if err != nil {
		return "", fmt.Errorf("failed to encode claims: %v", err)
	}
	return buildMessage(encHeader, encClaims), nil
}

func encodeToken[T comparable](claims T, issuer Issuer) (string, error) {
	message, err := encodeMessage(claims)
	if err != nil {
		return "", err
	}
	encSignature, err := issuer.SignHash(hashMessage(message))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", message, encSignature), nil
}

func decodeJWTSection[T comparable](str string, value *T) error {
	bytes, err := base64.RawURLEncoding.DecodeString(str)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %v", err)
	}
	err = json.Unmarshal(bytes, &value)
	if err != nil {
		return fmt.Errorf("not valid JSON: %v", err)
	}
	return nil
}

func validateStructure(tokenStr string) (
	header string,
	claims string,
	signature string,
	err error,
) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		err = fmt.Errorf("JWT expected three parts, found %d", len(parts))
		return
	}
	header = parts[0]
	claims = parts[1]
	signature = parts[2]
	return
}

func verifyHeader(header *JWTHeader) error {
	switch header.Type {
	case "JWT":
		break
	default:
		return fmt.Errorf("illegal type: %s", header.Type)
	}

	switch header.Algorithm {
	case "ES256":
		break
	default:
		return fmt.Errorf("illegal algorithm: %s", header.Algorithm)
	}

	return nil
}

func verifySignature(
	encHeader string,
	encClaims string,
	encSignature string,
	verificationKey *ecdsa.PublicKey,
) error {
	signature, err := base64.RawURLEncoding.DecodeString(encSignature)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %v", err)
	}

	r, s, err := decodeSignature(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %v", err)
	}

	hash := hashMessage(buildMessage(encHeader, encClaims))

	if valid := ecdsa.Verify(verificationKey, hash, r, s); !valid {
		return fmt.Errorf("verification failed")
	}

	return nil
}

func decodeToken[T claims](tokenStr string, validator Validator) (*T, *validateError) {
	encHeader, encClaims, encSignature, err := validateStructure(tokenStr)
	if err != nil {
		return nil, &validateError{
			context: fmt.Sprintf("token malformed: %v", err),
			err:     errTokenMalformed,
		}
	}

	header := JWTHeader{}
	if err := decodeJWTSection(encHeader, &header); err != nil {
		return nil, &validateError{
			context: fmt.Sprintf("token header malformed: %v", err),
			err:     errTokenMalformed,
		}
	}

	if err := verifyHeader(&header); err != nil {
		return nil, &validateError{
			context: fmt.Sprintf("token header illegal: %v", err),
			err:     errTokenBadSignature,
		}
	}

	if err := validator.VerifySignature(encHeader, encClaims, encSignature); err != nil {
		return nil, &validateError{
			context: fmt.Sprintf("token signature illegal: %v", err),
			err:     errTokenBadSignature,
		}
	}

	claims := new(T)
	if err := decodeJWTSection(encClaims, &claims); err != nil {
		return nil, &validateError{
			context: fmt.Sprintf("token claims malformed: %v", err),
			err:     errTokenMalformed,
		}
	}
	if err = (*claims).validate(validator); err != nil {
		return nil, &validateError{
			context: fmt.Sprintf("token claims invalid: %v", err),
			err:     err,
		}
	}

	return claims, nil
}
