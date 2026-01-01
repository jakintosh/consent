package tokens_test

import (
	"strings"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func TestAccessToken_Decode_Valid(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, validator := tokens.InitServer(key, "test.domain")

	// issue a valid token
	original, err := issuer.IssueAccessToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// decode succeeds and fields match
	decoded := &tokens.AccessToken{}
	err = decoded.Decode(original.Encoded(), validator)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if decoded.Subject() != original.Subject() {
		t.Errorf("Subject mismatch: %s != %s", decoded.Subject(), original.Subject())
	}
	if decoded.Issuer() != "test.domain" {
		t.Errorf("Issuer = %s, want test.domain", decoded.Issuer())
	}
}

func TestAccessToken_Decode_Expired(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, validator := tokens.InitServer(key, "test.domain")

	// issue token that's already expired
	original, err := issuer.IssueAccessToken("user", []string{"aud"}, -time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// decoding expired token fails
	decoded := &tokens.AccessToken{}
	err = decoded.Decode(original.Encoded(), validator)
	if err == nil {
		t.Error("expected error for expired token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected error about expiration, got %v", err)
	}
}

func TestAccessToken_Decode_WrongIssuer(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)

	// issue from one domain, validate with another
	issuer, _ := tokens.InitServer(key, "wrong.domain")
	_, validator := tokens.InitServer(key, "correct.domain")

	original, err := issuer.IssueAccessToken("user", []string{"aud"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	// decoding with wrong issuer fails
	decoded := &tokens.AccessToken{}
	err = decoded.Decode(original.Encoded(), validator)
	if err == nil {
		t.Error("expected error for wrong issuer")
	}
	if !strings.Contains(err.Error(), "issuer") {
		t.Errorf("expected error about issuer, got %v", err)
	}
}

func TestAccessToken_Decode_Malformed(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	_, validator := tokens.InitServer(key, "test.domain")

	decoded := &tokens.AccessToken{}

	// table-driven test for malformed tokens
	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"single part", "abc"},
		{"two parts", "abc.def"},
		{"four parts", "abc.def.ghi.jkl"},
		{"invalid base64", "!!!.@@@.###"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := decoded.Decode(tt.token, validator)
			if err == nil {
				t.Error("expected error for malformed token")
			}
		})
	}
}

func TestAccessToken_Fields(t *testing.T) {
	t.Parallel()
	key := getSharedTestKey(t)
	issuer, _ := tokens.InitServer(key, "test.domain")

	// issue token with specific values
	token, err := issuer.IssueAccessToken("user123", []string{"aud1", "aud2"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
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
}
