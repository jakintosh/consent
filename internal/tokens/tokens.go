package tokens

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

var issuerDomain string
var signingKey *ecdsa.PrivateKey
var verificationKey *ecdsa.PublicKey

func Init(privateKey *ecdsa.PrivateKey, issuer string) {
	issuerDomain = issuer
	signingKey = privateKey
	verificationKey = &privateKey.PublicKey
}

func InitPublic(publicKey *ecdsa.PublicKey, issuer string) {
	issuerDomain = issuer
	verificationKey = publicKey
}

type JWTHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
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
		return "", fmt.Errorf("Failed to generate random CSRF bytes: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func buildMessageHash(encHeader string, encClaims string) (string, []byte) {
	message := fmt.Sprintf("%s.%s", encHeader, encClaims)
	hash := sha256.Sum256([]byte(message))
	return message, hash[:]
}

func encodeSignature(r *big.Int, s *big.Int) ([]byte, error) {
	signature := append(r.Bytes(), s.Bytes()...)
	if len(signature) != 64 {
		return nil, fmt.Errorf("invalid signature length")
	}
	return signature, nil
}

func decodeSignature(signature []byte) (*big.Int, *big.Int, error) {
	if len(signature) != 64 {
		return nil, nil, fmt.Errorf("")
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

func encodeToken[Claims comparable](claims Claims) (string, error) {
	encHeader, err := encodeJWTSection(newES256JWTHeader())
	if err != nil {
		return "", fmt.Errorf("failed to encode header: %v", err)
	}
	encClaims, err := encodeJWTSection(claims)
	if err != nil {
		return "", fmt.Errorf("failed to encode claims: %v", err)
	}
	message, hash := buildMessageHash(encHeader, encClaims)

	// sign message
	r, s, err := ecdsa.Sign(rand.Reader, signingKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %v", err)
	}

	signature, err := encodeSignature(r, s)
	if err != nil {
		return "", fmt.Errorf("failed to encode signature: %v", err)
	}

	// build final token
	encSignature := base64.RawURLEncoding.EncodeToString(signature)
	token := fmt.Sprintf("%s.%s", message, encSignature)
	return token, nil
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

func validateParts(tokenStr string) (string, string, string, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("JWT expected three parts, found %d", len(parts))
	}
	return parts[0], parts[1], parts[2], nil
}

func validateHeader(header *JWTHeader) error {
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

func validateSignature(encHeader string, encClaims string, encSignature string) error {
	signature, err := base64.RawURLEncoding.DecodeString(encSignature)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %v", err)
	}

	r, s, err := decodeSignature(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %v", err)
	}

	_, hash := buildMessageHash(encHeader, encClaims)
	if valid := ecdsa.Verify(verificationKey, hash[:], r, s); !valid {
		return fmt.Errorf("verification failed")
	}

	return nil
}

func validateToken[Claims comparable](tokenStr string, claims *Claims) error {
	encHeader, encClaims, encSignature, err := validateParts(tokenStr)
	if err != nil {
		return fmt.Errorf("token malformed: %v", err)
	}

	header := JWTHeader{}
	err = decodeJWTSection(encHeader, &header)
	if err != nil {
		return fmt.Errorf("token header malformed: %v", err)
	}

	err = validateHeader(&header)
	if err != nil {
		return fmt.Errorf("token header invalid: %v", err)
	}

	err = validateSignature(encHeader, encClaims, encSignature)
	if err != nil {
		return fmt.Errorf("token signature invalid: %v", err)
	}

	err = decodeJWTSection(encClaims, claims)
	if err != nil {
		return fmt.Errorf("token claims malformed: %v", err)
	}

	return nil
}
