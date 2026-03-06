package testing

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.sr.ht/~jakintosh/consent/pkg/client"
)

func TestVerifyAuthorizationCheckCSRF_MissingRefreshIsAbsent(t *testing.T) {
	tv := NewTestVerifier("consent.test", "app.test")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	_, _, err := tv.VerifyAuthorizationCheckCSRF(rr, req, "csrf")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, client.ErrTokenAbsent) {
		t.Fatalf("expected ErrTokenAbsent, got %v", err)
	}
}
