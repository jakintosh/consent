package tokens_test

import (
	"strings"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func TestClient_ValidateDomain(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.domain",
		ValidAudience:   "my-app",
	}
	validator := tokens.InitClient(clientOpts)

	// matching domain returns true
	if !validator.ValidateDomain("consent.domain") {
		t.Error("ValidateDomain should return true for matching domain")
	}

	// non-matching domain returns false
	if validator.ValidateDomain("other.domain") {
		t.Error("ValidateDomain should return false for non-matching domain")
	}
}

func TestClient_ShouldValidateAudience(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.domain",
		ValidAudience:   "my-app",
	}
	validator := tokens.InitClient(clientOpts)

	// client-side validator requires audience validation
	if !validator.ShouldValidateAudience() {
		t.Error("Client validator should require audience validation")
	}
}

func TestClient_ValidateAudiences_Single(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.domain",
		ValidAudience:   "my-app",
	}
	validator := tokens.InitClient(clientOpts)

	// matching audience returns true
	if !validator.ValidateAudiences("my-app") {
		t.Error("ValidateAudiences should return true for matching audience")
	}

	// non-matching audience returns false
	if validator.ValidateAudiences("other-app") {
		t.Error("ValidateAudiences should return false for non-matching audience")
	}
}

func TestClient_ValidateAudiences_Multiple(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.domain",
		ValidAudience:   "my-app",
	}
	validator := tokens.InitClient(clientOpts)

	// target audience in list returns true
	if !validator.ValidateAudiences("other-app my-app another-app") {
		t.Error("ValidateAudiences should return true when target is in list")
	}

	// target audience not in list returns false
	if validator.ValidateAudiences("other-app another-app") {
		t.Error("ValidateAudiences should return false when target is not in list")
	}
}

func TestClient_VerifySignature_Valid(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := newTestServerWithKey(t, key, "consent.domain")
	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.domain",
		ValidAudience:   "my-app",
	}
	clientValidator := tokens.InitClient(clientOpts)

	// issue a token
	token, err := issuer.IssueAccessToken("user", []string{"my-app"}, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// parse JWT parts
	parts := strings.Split(token.Encoded(), ".")
	if len(parts) != 3 {
		t.Fatal("invalid JWT format")
	}

	// signature verification succeeds
	err = clientValidator.VerifySignature(parts[0], parts[1], parts[2])
	if err != nil {
		t.Errorf("VerifySignature failed: %v", err)
	}
}

func TestClient_VerifySignature_WrongKey(t *testing.T) {
	t.Parallel()
	key1 := generateTestKey(t)
	key2 := generateTestKey(t)

	// issue with one key, verify with another
	issuer, _ := newTestServerWithKey(t, key1, "consent.domain")
	clientOpts := tokens.ClientOptions{
		VerificationKey: &key2.PublicKey,
		IssuerDomain:    "consent.domain",
		ValidAudience:   "my-app",
	}
	clientValidator := tokens.InitClient(clientOpts)

	token, err := issuer.IssueAccessToken("user", []string{"my-app"}, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	parts := strings.Split(token.Encoded(), ".")

	// signature verification fails with wrong key
	err = clientValidator.VerifySignature(parts[0], parts[1], parts[2])
	if err == nil {
		t.Error("VerifySignature should fail with wrong key")
	}
}

func TestClient_DecodeToken_WrongAudience(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := newTestServer(t, "consent.domain")
	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.domain",
		ValidAudience:   "my-app",
	}
	clientValidator := tokens.InitClient(clientOpts)

	// issue token with different audience
	token, err := issuer.IssueAccessToken("user", []string{"other-app"}, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// decode fails with wrong audience
	decoded := &tokens.AccessToken{}
	err = decoded.Decode(token.Encoded(), clientValidator)
	if err == nil {
		t.Error("Decode should fail with wrong audience")
	}
}

func TestClient_DecodeToken_WrongIssuer(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := newTestServer(t, "wrong.domain")
	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.domain",
		ValidAudience:   "my-app",
	}
	clientValidator := tokens.InitClient(clientOpts)

	// issue token with wrong issuer
	token, err := issuer.IssueAccessToken("user", []string{"my-app"}, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// decode fails with wrong issuer
	decoded := &tokens.AccessToken{}
	err = decoded.Decode(token.Encoded(), clientValidator)
	if err == nil {
		t.Error("Decode should fail with wrong issuer")
	}
}
