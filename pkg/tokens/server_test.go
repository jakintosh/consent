package tokens_test

import (
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func TestServer_IssueRefreshToken(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// issuing refresh token succeeds with correct fields
	token, err := issuer.IssueRefreshToken("subject", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}
	if token.Subject() != "subject" {
		t.Errorf("Subject = %s, want subject", token.Subject())
	}
	if token.Encoded() == "" {
		t.Error("Encoded token is empty")
	}
	if token.Secret() == "" {
		t.Error("Secret (CSRF) is empty")
	}
	if token.Issuer() != "test.domain" {
		t.Errorf("Issuer = %s, want test.domain", token.Issuer())
	}
}

func TestServer_IssueAccessToken(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// issuing access token succeeds with correct fields
	token, err := issuer.IssueAccessToken("subject", []string{"aud"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}
	if token.Subject() != "subject" {
		t.Errorf("Subject = %s, want subject", token.Subject())
	}
	if token.Encoded() == "" {
		t.Error("Encoded token is empty")
	}
	if token.Issuer() != "test.domain" {
		t.Errorf("Issuer = %s, want test.domain", token.Issuer())
	}
}

func TestServer_SignHash(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// create a 32-byte hash (SHA256)
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i)
	}

	// signing succeeds and produces non-empty signature
	sig, err := issuer.SignHash(hash)
	if err != nil {
		t.Fatalf("SignHash failed: %v", err)
	}
	if sig == "" {
		t.Error("signature is empty")
	}
}

func TestServer_ValidateDomain(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	_, validator := tokens.InitServer(key, "test.domain")

	// matching domain returns true
	if !validator.ValidateDomain("test.domain") {
		t.Error("ValidateDomain should return true for matching domain")
	}

	// non-matching domain returns false
	if validator.ValidateDomain("other.domain") {
		t.Error("ValidateDomain should return false for non-matching domain")
	}
}

func TestServer_ShouldValidateAudience(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	_, validator := tokens.InitServer(key, "test.domain")

	// server-side validator does not require audience validation
	if validator.ShouldValidateAudience() {
		t.Error("Server validator should not require audience validation")
	}
}

func TestServer_TokenExpiration(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// issue token with 1 hour lifetime
	token, err := issuer.IssueAccessToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// expiration is approximately 1 hour from now
	expectedExp := time.Now().Add(time.Hour)
	actualExp := token.Expiration()
	diff := actualExp.Sub(expectedExp)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("Expiration off by more than 1 second: got %v, want ~%v", actualExp, expectedExp)
	}
}

func TestServer_MultipleAudiences(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, validator := tokens.InitServer(key, "test.domain")

	// issue token with multiple audiences
	audiences := []string{"app1", "app2", "app3"}
	token, err := issuer.IssueAccessToken("user", audiences, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// decode preserves all audiences
	decoded := &tokens.AccessToken{}
	if err := decoded.Decode(token.Encoded(), validator); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if len(decoded.Audience()) != 3 {
		t.Errorf("Audience len = %d, want 3", len(decoded.Audience()))
	}
}
