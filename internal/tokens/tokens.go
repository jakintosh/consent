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
)

type ctxError struct {
	context string
	err     error
}

func (t *ctxError) Push(ctx string) { t.context = fmt.Sprintf("%s: %s", ctx, t.context) }
func (t *ctxError) Set(err error)   { t.err = err }
func (t *ctxError) Error() string   { return fmt.Sprintf("%v", t.err) }
func (t *ctxError) Context() string { return t.context }

var (
	errTokenInvalid   = errors.New("token invalid")
	errTokenIllegal   = errors.New("token illegal")
	errTokenMalformed = errors.New("token malformed")
)

func ErrTokenInvalid() error   { return errTokenInvalid }
func ErrTokenIllegal() error   { return errTokenIllegal }
func ErrTokenMalformed() error { return errTokenMalformed }

var _issuerDomain string
var _signingKey *ecdsa.PrivateKey
var _verificationKey *ecdsa.PublicKey
var _validAudience *string = nil

func InitServer(signingKey *ecdsa.PrivateKey, issuerDomain string) {
	_signingKey = signingKey
	_verificationKey = &signingKey.PublicKey
	_issuerDomain = issuerDomain
}

func InitClient(verificationKey *ecdsa.PublicKey, issuerDomain string, validAudience string) {
	_verificationKey = verificationKey
	_issuerDomain = issuerDomain

	_validAudience = new(string)
	*_validAudience = validAudience
}

type JWTHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type claims interface {
	validate() error
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
	signature := append(r.Bytes(), s.Bytes()...)
	if len(signature) != 64 {
		return "", fmt.Errorf("invalid signature length")
	}
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

func signHash(hash []byte) (string, error) {
	r, s, err := ecdsa.Sign(rand.Reader, _signingKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %v", err)
	}
	encSignature, err := encodeSignature(r, s)
	if err != nil {
		return "", fmt.Errorf("failed to encode signature: %v", err)
	}
	return encSignature, nil
}

func encodeToken[T comparable](claims T) (string, error) {
	message, err := encodeMessage(claims)
	if err != nil {
		return "", err
	}
	encSignature, err := signHash(hashMessage(message))
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

func verifySignature(encHeader string, encClaims string, encSignature string) error {
	signature, err := base64.RawURLEncoding.DecodeString(encSignature)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %v", err)
	}

	r, s, err := decodeSignature(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %v", err)
	}

	hash := hashMessage(buildMessage(encHeader, encClaims))

	if valid := ecdsa.Verify(_verificationKey, hash, r, s); !valid {
		return fmt.Errorf("verification failed")
	}

	return nil
}

func validateToken[Claims claims](tokenStr string, claims Claims) *ctxError {
	encHeader, encClaims, encSignature, err := validateStructure(tokenStr)
	if err != nil {
		return &ctxError{
			context: fmt.Sprintf("token malformed: %v", err),
			err:     errTokenMalformed,
		}
	}

	header := JWTHeader{}
	if err := decodeJWTSection(encHeader, &header); err != nil {
		return &ctxError{
			context: fmt.Sprintf("token header malformed: %v", err),
			err:     errTokenMalformed,
		}
	}

	if err := verifyHeader(&header); err != nil {
		return &ctxError{
			context: fmt.Sprintf("token header illegal: %v", err),
			err:     errTokenIllegal,
		}
	}

	if err := verifySignature(encHeader, encClaims, encSignature); err != nil {
		return &ctxError{
			context: fmt.Sprintf("token signature illegal: %v", err),
			err:     errTokenIllegal,
		}
	}

	if err := decodeJWTSection(encClaims, &claims); err != nil {
		return &ctxError{
			context: fmt.Sprintf("token claims malformed: %v", err),
			err:     errTokenMalformed,
		}
	}

	if err = claims.validate(); err != nil {
		return &ctxError{
			context: fmt.Sprintf("token claims invalid: %v", err),
			err:     errTokenInvalid,
		}
	}

	return nil
}
