package tokens_test

import (
	"strings"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func TestDecodeLogger_DefaultSilent(t *testing.T) {
	tokens.SetDecodeLogger(nil)

	_, validator := newTestServer(t, "test.domain")
	decoded := &tokens.AccessToken{}
	if err := decoded.Decode("malformed", validator); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestDecodeLogger_ReceivesValidationContext(t *testing.T) {
	tokens.SetDecodeLogger(nil)
	t.Cleanup(func() { tokens.SetDecodeLogger(nil) })

	var logged []string
	tokens.SetDecodeLogger(func(context string) {
		logged = append(logged, context)
	})

	issuer, validator := newTestServer(t, "test.domain")
	token, err := issuer.IssueAccessToken("alice", []string{"app.test"}, nil, -time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}

	decoded := &tokens.AccessToken{}
	if err := decoded.Decode(token.Encoded(), validator); err == nil {
		t.Fatal("expected decode error")
	}
	if len(logged) != 1 {
		t.Fatalf("logged contexts = %d, want 1", len(logged))
	}
	if !strings.Contains(logged[0], "token claims invalid") {
		t.Fatalf("logged context = %q, want validation context", logged[0])
	}
}
