package tokens_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

var (
	sharedTestKey     *ecdsa.PrivateKey
	sharedTestKeyOnce sync.Once
)

// getSharedTestKey returns a shared ECDSA key for tests that don't need isolation.
// This avoids the overhead of generating a new key for each test.
func getSharedTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	sharedTestKeyOnce.Do(func() {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			panic("failed to generate shared test key: " + err.Error())
		}
		sharedTestKey = key
	})
	return sharedTestKey
}

// generateTestKey creates a new unique key for tests that require key isolation.
func generateTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	return key
}

func TestInitServer(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)

	// server initialization returns issuer and validator
	issuer, validator := tokens.InitServer(key, "test.domain")
	if issuer == nil {
		t.Error("InitServer returned nil issuer")
	}
	if validator == nil {
		t.Error("InitServer returned nil validator")
	}
}

func TestInitClient(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)

	// client initialization returns validator
	validator := tokens.InitClient(&key.PublicKey, "test.domain", "test-audience")
	if validator == nil {
		t.Error("InitClient returned nil validator")
	}
}

func TestRefreshToken_RoundTrip(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, validator := tokens.InitServer(key, "test.domain")

	// issue token
	original, err := issuer.IssueRefreshToken("user123", []string{"aud1"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	// decode token
	decoded := &tokens.RefreshToken{}
	err = decoded.Decode(original.Encoded(), validator)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// all fields are preserved through round-trip
	if decoded.Subject() != "user123" {
		t.Errorf("Subject = %s, want user123", decoded.Subject())
	}
	if decoded.Issuer() != "test.domain" {
		t.Errorf("Issuer = %s, want test.domain", decoded.Issuer())
	}
	if decoded.Secret() != original.Secret() {
		t.Error("Secret mismatch between original and decoded")
	}
}

func TestAccessToken_RoundTrip(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, validator := tokens.InitServer(key, "test.domain")

	// issue token
	original, err := issuer.IssueAccessToken("user123", []string{"aud1"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// decode token
	decoded := &tokens.AccessToken{}
	err = decoded.Decode(original.Encoded(), validator)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// all fields are preserved through round-trip
	if decoded.Subject() != "user123" {
		t.Errorf("Subject = %s, want user123", decoded.Subject())
	}
	if decoded.Issuer() != "test.domain" {
		t.Errorf("Issuer = %s, want test.domain", decoded.Issuer())
	}
}

func TestToken_CrossValidation(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)

	// issue token from server
	issuer, _ := tokens.InitServer(key, "consent.server")
	clientValidator := tokens.InitClient(&key.PublicKey, "consent.server", "my-app")

	token, err := issuer.IssueAccessToken("user", []string{"my-app"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// client can decode server-issued token
	decoded := &tokens.AccessToken{}
	err = decoded.Decode(token.Encoded(), clientValidator)
	if err != nil {
		t.Fatalf("Client decode failed: %v", err)
	}
	if decoded.Subject() != "user" {
		t.Errorf("Subject = %s, want user", decoded.Subject())
	}
}
