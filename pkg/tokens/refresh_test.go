package tokens_test

import (
	"strings"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func TestRefreshToken_Decode_Valid(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, validator := tokens.InitServer(key, "test.domain")

	// issue a valid token
	original, err := issuer.IssueRefreshToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	// decode succeeds and fields match
	decoded := &tokens.RefreshToken{}
	err = decoded.Decode(original.Encoded(), validator)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if decoded.Subject() != original.Subject() {
		t.Errorf("Subject mismatch")
	}
	if decoded.Secret() != original.Secret() {
		t.Errorf("Secret mismatch")
	}
}

func TestRefreshToken_Decode_Expired(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, validator := tokens.InitServer(key, "test.domain")

	// issue token that's already expired
	original, err := issuer.IssueRefreshToken("user", []string{"aud"}, -time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	// decoding expired token fails
	decoded := &tokens.RefreshToken{}
	err = decoded.Decode(original.Encoded(), validator)
	if err == nil {
		t.Error("expected error for expired token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected error about expiration, got %v", err)
	}
}

func TestRefreshToken_HasSecret(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// issue refresh token
	token, err := issuer.IssueRefreshToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	// refresh tokens have CSRF secret
	if token.Secret() == "" {
		t.Error("RefreshToken should have a secret")
	}
}

func TestRefreshToken_UniqueSecrets(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// issue two tokens
	token1, err := issuer.IssueRefreshToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	token2, err := issuer.IssueRefreshToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	// each token has unique secret
	if token1.Secret() == token2.Secret() {
		t.Error("Different tokens should have different secrets")
	}
}

func TestRefreshToken_UniqueEncodings(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// issue two tokens
	token1, err := issuer.IssueRefreshToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	token2, err := issuer.IssueRefreshToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	// each token has unique encoding
	if token1.Encoded() == token2.Encoded() {
		t.Error("Different tokens should have different encodings")
	}
}

func TestRefreshToken_Fields(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// issue token with specific values
	token, err := issuer.IssueRefreshToken("user123", []string{"aud1", "aud2"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	// all fields are accessible and correct
	if token.Subject() != "user123" {
		t.Errorf("Subject = %s, want user123", token.Subject())
	}
	if token.Issuer() != "test.domain" {
		t.Errorf("Issuer = %s, want test.domain", token.Issuer())
	}
	if len(token.Audience()) != 2 {
		t.Errorf("Audience len = %d, want 2", len(token.Audience()))
	}
	if token.Expiration().Before(time.Now()) {
		t.Error("Expiration should be in the future")
	}
	if token.IssuedAt().After(time.Now()) {
		t.Error("IssuedAt should be in the past or now")
	}
	if token.Encoded() == "" {
		t.Error("Encoded should not be empty")
	}
	if token.Secret() == "" {
		t.Error("Secret should not be empty")
	}
}

func TestRefreshToken_SecretPreservedAfterDecode(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, validator := tokens.InitServer(key, "test.domain")

	// issue token
	original, err := issuer.IssueRefreshToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	// decode token
	decoded := &tokens.RefreshToken{}
	err = decoded.Decode(original.Encoded(), validator)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// secret is preserved through encode/decode
	if decoded.Secret() != original.Secret() {
		t.Error("Secret should be preserved after decode")
	}
}
