package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func TestInit_AuthURL(t *testing.T) {
	t.Run("normalizes trailing slashes", func(t *testing.T) {
		client, err := Init(nil, "https://consent.example.com/base///")
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		if client.authUrl != "https://consent.example.com/base" {
			t.Fatalf("auth URL = %q, want %q", client.authUrl, "https://consent.example.com/base")
		}
	})

	tests := map[string]string{
		"empty":          "",
		"surrounding WS": " https://consent.example.com",
		"relative":       "consent.example.com",
		"missing host":   "https:///api",
		"unsupported":    "ftp://consent.example.com",
		"unix socket":    "unix:///var/run/consent.sock",
		"credentials":    "https://user:pass@consent.example.com",
		"query":          "https://consent.example.com?debug=true",
		"fragment":       "https://consent.example.com#fragment",
	}

	for name, authURL := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Init(nil, authURL)
			if err == nil {
				t.Fatal("expected invalid auth URL error")
			}
			if !strings.Contains(err.Error(), "invalid auth URL") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRefreshTokens_CanceledContext(t *testing.T) {
	client := testClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := client.RefreshTokens(ctx, "refresh-token")
	if !errors.Is(err, ErrTokenRefresh) {
		t.Fatalf("expected ErrTokenRefresh, got %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestHandleLogout_Success(t *testing.T) {
	refreshToken, c := setupLogoutTestClient(t, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/logout?csrf="+url.QueryEscape(refreshToken.Secret()), nil)
	req.AddCookie(&http.Cookie{Name: "refreshToken", Value: refreshToken.Encoded()})
	rr := httptest.NewRecorder()

	c.HandleLogout()(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if rr.Header().Get("Location") != "/" {
		t.Fatalf("location = %q, want %q", rr.Header().Get("Location"), "/")
	}
	assertCookiesCleared(t, rr)

	if !logoutCalled {
		t.Fatalf("expected logout endpoint to be called")
	}
	if revokedToken != refreshToken.Encoded() {
		t.Fatalf("revoked token mismatch")
	}
}

func TestHandleLogout_InvalidCSRF(t *testing.T) {
	refreshToken, c := setupLogoutTestClient(t, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/logout?csrf=wrong", nil)
	req.AddCookie(&http.Cookie{Name: "refreshToken", Value: refreshToken.Encoded()})
	rr := httptest.NewRecorder()

	c.HandleLogout()(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if logoutCalled {
		t.Fatalf("logout endpoint should not be called when csrf fails")
	}
	if len(rr.Result().Cookies()) != 0 {
		t.Fatalf("cookies should not be cleared on csrf mismatch")
	}
}

func TestHandleLogout_MissingCSRF(t *testing.T) {
	refreshToken, c := setupLogoutTestClient(t, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refreshToken", Value: refreshToken.Encoded()})
	rr := httptest.NewRecorder()

	c.HandleLogout()(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if logoutCalled {
		t.Fatalf("logout endpoint should not be called when csrf is missing")
	}
	if len(rr.Result().Cookies()) != 0 {
		t.Fatalf("cookies should not be cleared when csrf is missing")
	}
}

func TestHandleLogout_MissingRefreshCookie(t *testing.T) {
	refreshToken, c := setupLogoutTestClient(t, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/logout?csrf="+url.QueryEscape(refreshToken.Secret()), nil)
	rr := httptest.NewRecorder()

	c.HandleLogout()(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if rr.Header().Get("Location") != "/" {
		t.Fatalf("location = %q, want %q", rr.Header().Get("Location"), "/")
	}
	if logoutCalled {
		t.Fatalf("logout endpoint should not be called without refresh cookie")
	}
	assertCookiesCleared(t, rr)
}

func TestHandleLogout_RevocationFailureStillClearsCookies(t *testing.T) {
	refreshToken, c := setupLogoutTestClient(t, http.StatusInternalServerError)

	req := httptest.NewRequest(http.MethodGet, "/logout?csrf="+url.QueryEscape(refreshToken.Secret()), nil)
	req.AddCookie(&http.Cookie{Name: "refreshToken", Value: refreshToken.Encoded()})
	rr := httptest.NewRecorder()

	c.HandleLogout()(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if rr.Header().Get("Location") != "/" {
		t.Fatalf("location = %q, want %q", rr.Header().Get("Location"), "/")
	}
	if !logoutCalled {
		t.Fatalf("expected logout endpoint to be called")
	}
	assertCookiesCleared(t, rr)
}

func TestVerifyAuthorization_InvalidRefreshIncludesContext(t *testing.T) {
	c := testClient(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "refreshToken", Value: "invalid-token"})
	rr := httptest.NewRecorder()

	_, err := c.VerifyAuthorization(rr, req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid, got %v", err)
	}
	if !strings.Contains(err.Error(), "token invalid:") {
		t.Fatalf("expected wrapped error context, got %q", err.Error())
	}
}

func TestVerifyAuthorizationCheckCSRF_MissingRefreshIsAbsent(t *testing.T) {
	c := testClient(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	_, _, err := c.VerifyAuthorizationCheckCSRF(rr, req, "csrf")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrTokenAbsent) {
		t.Fatalf("expected ErrTokenAbsent, got %v", err)
	}
}

func TestSetTokenCookies_UsesLaxSameSite(t *testing.T) {
	c := testClient(t)
	accessToken, refreshToken := issueTestTokens(t, "alice", "app.test")
	rr := httptest.NewRecorder()

	c.SetTokenCookies(rr, accessToken, refreshToken)

	assertCookieSameSiteLax(t, rr.Result().Cookies())
}

func TestSetTokenCookies_DefaultsToSecure(t *testing.T) {
	c := testClient(t)
	accessToken, refreshToken := issueTestTokens(t, "alice", "app.test")
	rr := httptest.NewRecorder()

	c.SetTokenCookies(rr, accessToken, refreshToken)

	assertCookieSecure(t, rr.Result().Cookies(), true)
}

func TestSetTokenCookies_InsecureCookiesDisablesSecure(t *testing.T) {
	c := testClient(t)
	c.EnableInsecureCookies()
	accessToken, refreshToken := issueTestTokens(t, "alice", "app.test")
	rr := httptest.NewRecorder()

	c.SetTokenCookies(rr, accessToken, refreshToken)

	assertCookieSecure(t, rr.Result().Cookies(), false)
}

func TestClearTokenCookies_UsesLaxSameSite(t *testing.T) {
	c := testClient(t)
	rr := httptest.NewRecorder()

	c.ClearTokenCookies(rr)

	assertCookieSameSiteLax(t, rr.Result().Cookies())
}

func TestClearTokenCookies_DefaultsToSecure(t *testing.T) {
	c := testClient(t)
	rr := httptest.NewRecorder()

	c.ClearTokenCookies(rr)

	assertCookieSecure(t, rr.Result().Cookies(), true)
}

func TestClearTokenCookies_InsecureCookiesDisablesSecure(t *testing.T) {
	c := testClient(t)
	c.EnableInsecureCookies()
	rr := httptest.NewRecorder()

	c.ClearTokenCookies(rr)

	assertCookieSecure(t, rr.Result().Cookies(), false)
}

func TestFetchUserInfo_SendsBearerTokenAndDecodesResponse(t *testing.T) {
	wantToken := "access.token.value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/userinfo" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer "+wantToken {
			t.Fatalf("authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer "+wantToken)
		}
		if err := json.NewEncoder(w).Encode(struct {
			Data UserInfo `json:"data"`
		}{
			Data: UserInfo{Sub: "subject-alice", Profile: &UserInfoProfile{Handle: "alice"}},
		}); err != nil {
			t.Fatalf("Encode failed: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	c := mustInit(t, nil, server.URL)
	response, err := c.FetchUserInfo(context.Background(), wantToken)
	if err != nil {
		t.Fatalf("FetchUserInfo failed: %v", err)
	}
	if response.Sub != "subject-alice" {
		t.Fatalf("sub = %q, want subject-alice", response.Sub)
	}
	if response.Profile == nil || response.Profile.Handle != "alice" {
		t.Fatalf("profile = %#v, want alice", response.Profile)
	}
}

func TestFetchUserInfo_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	c := mustInit(t, nil, server.URL)
	_, err := c.FetchUserInfo(context.Background(), "access.token.value")
	if err == nil {
		t.Fatal("expected error")
	}
	if !wire.IsStatus(err, http.StatusForbidden) {
		t.Fatalf("expected forbidden wire error, got %v", err)
	}
}

func TestFetchUserInfo_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	t.Cleanup(server.Close)

	c := mustInit(t, nil, server.URL)
	_, err := c.FetchUserInfo(context.Background(), "access.token.value")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "fetch user info") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestFetchUserInfo_CanceledContext(t *testing.T) {
	c := mustInit(t, nil, "https://consent.test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.FetchUserInfo(ctx, "access.token.value")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

var (
	logoutCalled bool
	revokedToken string
)

func setupLogoutTestClient(
	t *testing.T,
	logoutStatus int,
) (
	*RefreshToken,
	*Client,
) {
	t.Helper()

	logoutCalled = false
	revokedToken = ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/logout" {
			http.NotFound(w, r)
			return
		}
		logoutCalled = true

		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Fatalf("content type = %s, want application/json", r.Header.Get("Content-Type"))
		}

		payload := struct {
			RefreshToken string `json:"refreshToken"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload failed: %v", err)
		}
		revokedToken = payload.RefreshToken

		w.WriteHeader(logoutStatus)
	}))
	t.Cleanup(server.Close)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	opts := tokens.ServerOptions{
		SigningKey:   key,
		IssuerDomain: "consent.test",
	}
	issuer, _ := tokens.InitServer(opts)

	refreshToken, err := issuer.IssueRefreshToken("alice", []string{"app.test"}, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.test",
		ValidAudience:   "app.test",
	}
	validator := tokens.InitClient(clientOpts)
	return refreshToken, mustInit(t, validator, server.URL)
}

func assertCookiesCleared(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()

	cookies := rr.Result().Cookies()
	haveAccess := false
	haveRefresh := false

	for _, cookie := range cookies {
		switch cookie.Name {
		case "accessToken":
			haveAccess = true
		case "refreshToken":
			haveRefresh = true
		}
	}

	if !haveAccess || !haveRefresh {
		t.Fatalf("expected both token cookies to be cleared")
	}
}

func assertCookieSameSiteLax(t *testing.T, cookies []*http.Cookie) {
	t.Helper()

	for _, cookie := range cookies {
		switch cookie.Name {
		case "accessToken", "refreshToken":
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Fatalf("cookie %q SameSite = %v, want %v", cookie.Name, cookie.SameSite, http.SameSiteLaxMode)
			}
		}
	}
}

func assertCookieSecure(t *testing.T, cookies []*http.Cookie, want bool) {
	t.Helper()

	for _, cookie := range cookies {
		switch cookie.Name {
		case "accessToken", "refreshToken":
			if cookie.Secure != want {
				t.Fatalf("cookie %q Secure = %t, want %t", cookie.Name, cookie.Secure, want)
			}
		}
	}
}

func issueTestTokens(t *testing.T, subject string, audience string) (*AccessToken, *RefreshToken) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	issuer, _ := tokens.InitServer(tokens.ServerOptions{
		SigningKey:   key,
		IssuerDomain: "consent.test",
	})

	accessToken, err := issuer.IssueAccessToken(subject, []string{audience}, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccessToken failed: %v", err)
	}
	refreshToken, err := issuer.IssueRefreshToken(subject, []string{audience}, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefreshToken failed: %v", err)
	}

	return accessToken, refreshToken
}

func testClient(t *testing.T) *Client {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	clientOpts := tokens.ClientOptions{
		VerificationKey: &key.PublicKey,
		IssuerDomain:    "consent.test",
		ValidAudience:   "app.test",
	}
	validator := tokens.InitClient(clientOpts)
	return mustInit(t, validator, "https://consent.test")
}

func mustInit(
	t *testing.T,
	validator TokenValidator,
	authURL string,
) *Client {
	t.Helper()

	client, err := Init(validator, authURL)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	return client
}
