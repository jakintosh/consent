package integration_test

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/service"
	consentclient "git.sr.ht/~jakintosh/consent/pkg/client"
	consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

const (
	testIssuerDomain  = "consent.test.local"
	testAppAudience   = "example-app.local"
	testServiceName   = "example-app"
	testBootstrapKey  = "test.0123456789abcdef"
	testUserHandle    = "alice"
	testUserPassword  = "password123"
	testServiceNameUI = "Example App"
)

type apiCounters struct {
	refreshCalls atomic.Int32
	logoutCalls  atomic.Int32
}

type e2eHarness struct {
	consentServer *httptest.Server
	appServer     *httptest.Server
	db            *database.SQLStore

	signingKey *ecdsa.PrivateKey
	validator  tokens.Validator
	counters   apiCounters
}

func TestAuthFlow_E2E(t *testing.T) {
	h := newE2EHarness(t)
	defer h.close()

	loginBody := map[string]string{
		"handle":  testUserHandle,
		"secret":  testUserPassword,
		"service": testServiceName,
	}
	loginResp := postJSONNoRedirect(t, h.consentServer.Client(), h.consentServer.URL+"/api/v1/login", loginBody)
	if loginResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status = %d, want %d", loginResp.StatusCode, http.StatusSeeOther)
	}
	authCodeRedirect := loginResp.Header.Get("Location")
	if !strings.Contains(authCodeRedirect, "auth_code=") {
		t.Fatalf("login redirect missing auth_code: %q", authCodeRedirect)
	}
	loginResp.Body.Close()

	callbackResp := getNoRedirectWithCookies(t, h.appServer.Client(), authCodeRedirect)
	if callbackResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("callback status = %d, want %d", callbackResp.StatusCode, http.StatusSeeOther)
	}
	accessCookie := cookieByName(callbackResp.Cookies(), "accessToken")
	refreshCookie := cookieByName(callbackResp.Cookies(), "refreshToken")
	if accessCookie == nil || refreshCookie == nil {
		t.Fatalf("callback should set accessToken and refreshToken cookies")
	}
	callbackResp.Body.Close()

	protectedResp := getNoRedirectWithCookies(t, h.appServer.Client(), h.appServer.URL+"/protected", accessCookie, refreshCookie)
	if protectedResp.StatusCode != http.StatusOK {
		t.Fatalf("protected status = %d, want %d", protectedResp.StatusCode, http.StatusOK)
	}
	protectedResp.Body.Close()

	refreshBefore := h.counters.refreshCalls.Load()
	refreshPathResp := getNoRedirectWithCookies(t, h.appServer.Client(), h.appServer.URL+"/protected", refreshCookie)
	if refreshPathResp.StatusCode != http.StatusOK {
		t.Fatalf("refresh-path protected status = %d, want %d", refreshPathResp.StatusCode, http.StatusOK)
	}
	if h.counters.refreshCalls.Load() <= refreshBefore {
		t.Fatalf("expected refresh endpoint to be called during protected refresh")
	}
	rotatedRefreshCookie := cookieByName(refreshPathResp.Cookies(), "refreshToken")
	if rotatedRefreshCookie == nil {
		rotatedRefreshCookie = refreshCookie
	}
	refreshPathResp.Body.Close()

	logoutBefore := h.counters.logoutCalls.Load()
	csrf := decodeRefreshCSRF(t, rotatedRefreshCookie.Value, h.validator)
	logoutURL := h.appServer.URL + "/logout?csrf=" + url.QueryEscape(csrf)
	logoutResp := getNoRedirectWithCookies(t, h.appServer.Client(), logoutURL, rotatedRefreshCookie)
	if logoutResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want %d", logoutResp.StatusCode, http.StatusSeeOther)
	}
	if h.counters.logoutCalls.Load() <= logoutBefore {
		t.Fatalf("expected logout endpoint to be called")
	}
	logoutResp.Body.Close()

	afterLogoutResp := getNoRedirectWithCookies(t, h.appServer.Client(), h.appServer.URL+"/protected", rotatedRefreshCookie)
	if afterLogoutResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout protected status = %d, want %d", afterLogoutResp.StatusCode, http.StatusUnauthorized)
	}
	afterLogoutResp.Body.Close()
}

func newE2EHarness(t *testing.T) *e2eHarness {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "consent.sqlite")
	db, err := database.NewSQLStore(database.SQLStoreOptions{Path: dbPath})
	if err != nil {
		t.Fatalf("NewSQLStore failed: %v", err)
	}

	signingKey := consenttesting.SharedTestKey()
	svc, err := service.New(service.ServiceOptions{
		PasswordMode: service.PasswordModeTesting,
		Store:        db,
		PublicURL:    "https://consent.test",
		TokenServerOpts: tokens.ServerOptions{
			SigningKey:   signingKey,
			IssuerDomain: testIssuerDomain,
		},
		KeysOptions: keys.Options{
			Store:          db.KeysStore,
			BootstrapToken: testBootstrapKey,
		},
	})
	if err != nil {
		t.Fatalf("service.New failed: %v", err)
	}

	if err := svc.Register(testUserHandle, testUserPassword); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	h := &e2eHarness{
		db:         db,
		signingKey: signingKey,
		validator:  tokens.InitClient(&signingKey.PublicKey, testIssuerDomain, testAppAudience),
	}

	apiMux := http.NewServeMux()
	apiMux.Handle("/api/v1/", http.StripPrefix("/api/v1", svc.Router()))
	h.consentServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			switch r.URL.Path {
			case "/api/v1/refresh":
				h.counters.refreshCalls.Add(1)
			case "/api/v1/logout":
				h.counters.logoutCalls.Add(1)
			}
		}

		apiMux.ServeHTTP(w, r)
	}))

	authClient := consentclient.Init(h.validator, h.consentServer.URL)
	appMux := http.NewServeMux()
	appMux.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
	appMux.HandleFunc("/logout", authClient.HandleLogout())
	appMux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		token, err := authClient.VerifyAuthorization(w, r)
		if err != nil || token == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(token.Subject()))
	})
	h.appServer = httptest.NewTLSServer(appMux)

	err = svc.CreateService(
		testServiceName,
		testServiceNameUI,
		testAppAudience,
		h.appServer.URL+"/auth/callback",
	)
	if err != nil {
		t.Fatalf("CreateService failed: %v", err)
	}

	return h
}

func (h *e2eHarness) close() {
	if h.appServer != nil {
		h.appServer.Close()
	}
	if h.consentServer != nil {
		h.consentServer.Close()
	}
	if h.db != nil {
		_ = h.db.Close()
	}
}

func postJSONNoRedirect(
	t *testing.T,
	baseClient *http.Client,
	endpoint string,
	body any,
) *http.Response {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := noRedirectClient(baseClient).Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", endpoint, err)
	}

	return resp
}

func getNoRedirectWithCookies(
	t *testing.T,
	baseClient *http.Client,
	endpoint string,
	cookies ...*http.Cookie,
) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	for _, c := range cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}

	resp, err := noRedirectClient(baseClient).Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", endpoint, err)
	}

	return resp
}

func noRedirectClient(
	base *http.Client,
) *http.Client {
	clientCopy := *base
	clientCopy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &clientCopy
}

func cookieByName(
	cookies []*http.Cookie,
	name string,
) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

func decodeRefreshCSRF(
	t *testing.T,
	encoded string,
	validator tokens.Validator,
) string {
	t.Helper()

	var refreshToken tokens.RefreshToken
	if err := refreshToken.Decode(encoded, validator); err != nil {
		t.Fatalf("RefreshToken.Decode failed: %v", err)
	}

	return refreshToken.Secret()
}
