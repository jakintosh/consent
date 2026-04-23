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

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/app"
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
	testState         = "test-state"
)

type apiCounters struct {
	refreshCalls atomic.Int32
	logoutCalls  atomic.Int32
}

type e2eHarness struct {
	consentServer *httptest.Server
	appServer     *httptest.Server
	db            *database.DB

	signingKey *ecdsa.PrivateKey
	validator  tokens.Validator
	counters   apiCounters
}

func TestAuthFlow_E2E(t *testing.T) {
	h := newE2EHarness(t)
	defer h.close()

	authorizeURL := h.consentServer.URL + "/authorize?service=" + url.QueryEscape(testServiceName) + "&scope=identity&scope=profile&state=" + url.QueryEscape(testState)
	authorizeResp := getNoRedirectWithCookies(t, h.consentServer.Client(), authorizeURL)
	if authorizeResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("authorize status = %d, want %d", authorizeResp.StatusCode, http.StatusSeeOther)
	}
	loginRedirect := authorizeResp.Header.Get("Location")
	if !strings.Contains(loginRedirect, "/login?") || !strings.Contains(loginRedirect, "return_to=") {
		t.Fatalf("authorize redirect = %q, want login redirect with return_to", loginRedirect)
	}
	authorizeResp.Body.Close()

	loginBody := url.Values{
		"handle":    []string{testUserHandle},
		"secret":    []string{testUserPassword},
		"service":   []string{service.InternalServiceName},
		"return_to": []string{mustURL(t, authorizeURL).RequestURI()},
	}
	loginResp := postFormNoRedirect(t, h.consentServer.Client(), h.consentServer.URL+"/api/v1/auth/login", loginBody)
	if loginResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status = %d, want %d", loginResp.StatusCode, http.StatusSeeOther)
	}
	loginCallback := loginResp.Header.Get("Location")
	if !strings.Contains(loginCallback, "/auth/callback") || !strings.Contains(loginCallback, "auth_code=") {
		t.Fatalf("login redirect = %q, want internal auth callback with auth_code", loginCallback)
	}
	loginResp.Body.Close()

	consentCallbackResp := getNoRedirectWithCookies(t, h.consentServer.Client(), loginCallback)
	if consentCallbackResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("consent callback status = %d, want %d", consentCallbackResp.StatusCode, http.StatusSeeOther)
	}
	consentAccessCookie := cookieByName(consentCallbackResp.Cookies(), "accessToken")
	consentRefreshCookie := cookieByName(consentCallbackResp.Cookies(), "refreshToken")
	if consentAccessCookie == nil || consentRefreshCookie == nil {
		t.Fatalf("consent callback should set accessToken and refreshToken cookies")
	}
	returnTo := consentCallbackResp.Header.Get("Location")
	if !strings.Contains(returnTo, "/authorize?") {
		t.Fatalf("consent callback redirect = %q, want original authorize request", returnTo)
	}
	consentCallbackResp.Body.Close()

	approvalResp := getNoRedirectWithCookies(t, h.consentServer.Client(), h.consentServer.URL+returnTo, consentAccessCookie, consentRefreshCookie)
	if approvalResp.StatusCode != http.StatusOK {
		t.Fatalf("approval page status = %d, want %d", approvalResp.StatusCode, http.StatusOK)
	}
	approvalBody := readBody(t, approvalResp)
	if !strings.Contains(approvalBody, "Authorize "+testServiceNameUI) {
		t.Fatalf("approval page missing service display: %q", approvalBody)
	}
	approvalResp.Body.Close()

	clientOpts := tokens.ClientOptions{
		VerificationKey: &h.signingKey.PublicKey,
		IssuerDomain:    testIssuerDomain,
		ValidAudience:   mustURL(t, h.consentServer.URL).Host,
	}
	tkValidator := tokens.InitClient(clientOpts)
	csrf := decodeRefreshCSRF(t, consentRefreshCookie.Value, tkValidator)
	approveBody := url.Values{
		"service": []string{testServiceName},
		"scope":   []string{"identity", "profile"},
		"state":   []string{testState},
		"csrf":    []string{csrf},
		"action":  []string{"approve"},
	}
	approveResp := postFormWithCookiesNoRedirect(t, h.consentServer.Client(), h.consentServer.URL+"/authorize", approveBody, consentAccessCookie, consentRefreshCookie)
	if approveResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("approve status = %d, want %d", approveResp.StatusCode, http.StatusSeeOther)
	}
	authCodeRedirect := approveResp.Header.Get("Location")
	if !strings.Contains(authCodeRedirect, "auth_code=") || !strings.Contains(authCodeRedirect, "state="+testState) {
		t.Fatalf("approve redirect missing auth_code/state: %q", authCodeRedirect)
	}
	approveResp.Body.Close()

	callbackResp := getNoRedirectWithCookies(t, h.appServer.Client(), authCodeRedirect)
	if callbackResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("app callback status = %d, want %d", callbackResp.StatusCode, http.StatusSeeOther)
	}
	accessCookie := cookieByName(callbackResp.Cookies(), "accessToken")
	refreshCookie := cookieByName(callbackResp.Cookies(), "refreshToken")
	if accessCookie == nil || refreshCookie == nil {
		t.Fatalf("app callback should set accessToken and refreshToken cookies")
	}
	callbackResp.Body.Close()

	protectedResp := getNoRedirectWithCookies(t, h.appServer.Client(), h.appServer.URL+"/protected", accessCookie, refreshCookie)
	if protectedResp.StatusCode != http.StatusOK {
		t.Fatalf("protected status = %d, want %d", protectedResp.StatusCode, http.StatusOK)
	}
	protectedResp.Body.Close()

	var appAccessToken tokens.AccessToken
	if err := appAccessToken.Decode(accessCookie.Value, h.validator); err != nil {
		t.Fatalf("AccessToken.Decode failed: %v", err)
	}
	if appAccessToken.Subject() == testUserHandle {
		t.Fatalf("expected opaque sub, got handle %q", appAccessToken.Subject())
	}

	meResp := getBearerNoRedirect(t, h.consentServer.Client(), h.consentServer.URL+"/api/v1/auth/me", accessCookie.Value)
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("/api/v1/auth/me status = %d, want %d", meResp.StatusCode, http.StatusOK)
	}
	var meBody struct {
		Profile struct {
			Handle string `json:"handle"`
		} `json:"profile"`
	}
	decodeWireData(t, meResp, &meBody)
	if meBody.Profile.Handle != testUserHandle {
		t.Fatalf("profile.handle = %q, want %q", meBody.Profile.Handle, testUserHandle)
	}
	meResp.Body.Close()

	identityOnlyURL := h.consentServer.URL + "/authorize?service=" + url.QueryEscape(testServiceName) + "&scope=identity&state=identity-only"
	identityOnlyResp := getNoRedirectWithCookies(t, h.consentServer.Client(), identityOnlyURL, consentAccessCookie, consentRefreshCookie)
	if identityOnlyResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("identity-only authorize status = %d, want %d", identityOnlyResp.StatusCode, http.StatusSeeOther)
	}
	identityOnlyRedirect := identityOnlyResp.Header.Get("Location")
	if !strings.Contains(identityOnlyRedirect, "auth_code=") || !strings.Contains(identityOnlyRedirect, "state=identity-only") {
		t.Fatalf("identity-only redirect missing auth_code/state: %q", identityOnlyRedirect)
	}
	identityOnlyResp.Body.Close()

	identityCallbackResp := getNoRedirectWithCookies(t, h.appServer.Client(), identityOnlyRedirect)
	if identityCallbackResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("identity-only callback status = %d, want %d", identityCallbackResp.StatusCode, http.StatusSeeOther)
	}
	identityAccessCookie := cookieByName(identityCallbackResp.Cookies(), "accessToken")
	if identityAccessCookie == nil {
		t.Fatalf("identity-only callback should set access token cookie")
	}
	identityCallbackResp.Body.Close()

	identityMeResp := getBearerNoRedirect(t, h.consentServer.Client(), h.consentServer.URL+"/api/v1/auth/me", identityAccessCookie.Value)
	if identityMeResp.StatusCode != http.StatusOK {
		t.Fatalf("identity-only /api/v1/auth/me status = %d, want %d", identityMeResp.StatusCode, http.StatusOK)
	}
	if strings.Contains(readBody(t, identityMeResp), "profile") {
		t.Fatalf("identity-only /api/v1/auth/me should not include profile data")
	}
	identityMeResp.Body.Close()
}

func newE2EHarness(t *testing.T) *e2eHarness {
	t.Helper()

	dbOpts := database.Options{
		Path: filepath.Join(t.TempDir(), "consent.sqlite"),
	}
	db, err := database.Open(dbOpts)
	if err != nil {
		t.Fatalf("database.Open failed: %v", err)
	}

	signingKey := consenttesting.SharedTestKey()
	h := &e2eHarness{db: db, signingKey: signingKey}

	var svc *service.Service
	var appHandler http.Handler
	apiMux := http.NewServeMux()
	h.consentServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			if r.Method == http.MethodPost {
				switch r.URL.Path {
				case "/api/v1/auth/refresh":
					h.counters.refreshCalls.Add(1)
				case "/api/v1/auth/logout":
					h.counters.logoutCalls.Add(1)
				}
			}
			apiMux.ServeHTTP(w, r)
			return
		}
		appHandler.ServeHTTP(w, r)
	}))

	initOpts := service.InitOptions{
		Store:          db,
		KeysStore:      db.KeysStore,
		PublicURL:      h.consentServer.URL,
		BootstrapToken: testBootstrapKey,
	}
	if err := service.Init(initOpts); err != nil {
		t.Fatalf("service.Init failed: %v", err)
	}

	svc, err = service.New(service.Options{
		PasswordMode: service.PasswordModeTesting,
		Store:        db,
		TokenServerOpts: tokens.ServerOptions{
			SigningKey:   signingKey,
			IssuerDomain: testIssuerDomain,
		},
		ResourceTokenClientOpts: tokens.ClientOptions{
			VerificationKey: &signingKey.PublicKey,
			IssuerDomain:    testIssuerDomain,
			ValidAudience:   testIssuerDomain,
		},
	})
	if err != nil {
		t.Fatalf("service.New failed: %v", err)
	}
	apiServer, err := api.New(api.Options{
		Service:   svc,
		KeysStore: db.KeysStore,
	})
	if err != nil {
		t.Fatalf("api.New failed: %v", err)
	}
	wire.Subrouter(apiMux, "/api/v1", apiServer.Router())

	clientOpts := tokens.ClientOptions{
		VerificationKey: &signingKey.PublicKey,
		IssuerDomain:    testIssuerDomain,
		ValidAudience:   mustURL(t, h.consentServer.URL).Host,
	}
	tkValidator := tokens.InitClient(clientOpts)
	consentClient := consentclient.Init(tkValidator, h.consentServer.URL)
	appServer, err := app.New(app.Options{
		Service: svc,
		Auth: app.AuthConfig{
			Verifier:  consentClient,
			LoginURL:  "/login",
			LogoutURL: "/logout",
			Routes: map[string]http.HandlerFunc{
				"/auth/callback": consentClient.HandleAuthorizationCode(),
				"/logout":        consentClient.HandleLogout(),
			},
		},
	})
	if err != nil {
		t.Fatalf("app.New failed: %v", err)
	}
	appHandler = appServer.Router()

	if err := svc.Register(testUserHandle, testUserPassword); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := svc.CreateService(testServiceName, testServiceNameUI, testAppAudience, h.appServerURL()+"/auth/callback"); err != nil {
		t.Fatalf("CreateService failed: %v", err)
	}

	clientOpts = tokens.ClientOptions{
		VerificationKey: &signingKey.PublicKey,
		IssuerDomain:    testIssuerDomain,
		ValidAudience:   testAppAudience,
	}
	h.validator = tokens.InitClient(clientOpts)

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

	if err := svc.UpdateService(testServiceName, nil, nil, stringPtr(h.appServer.URL+"/auth/callback")); err != nil {
		t.Fatalf("UpdateService redirect failed: %v", err)
	}

	return h
}

func (h *e2eHarness) appServerURL() string {
	if h.appServer != nil {
		return h.appServer.URL
	}
	return "https://placeholder.invalid"
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

func postFormNoRedirect(t *testing.T, baseClient *http.Client, endpoint string, body url.Values) *http.Response {
	t.Helper()
	return postFormWithCookiesNoRedirect(t, baseClient, endpoint, body)
}

func postFormWithCookiesNoRedirect(t *testing.T, baseClient *http.Client, endpoint string, body url.Values, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}
	resp, err := noRedirectClient(baseClient).Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", endpoint, err)
	}
	return resp
}

func getBearerNoRedirect(t *testing.T, baseClient *http.Client, endpoint string, accessToken string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := noRedirectClient(baseClient).Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", endpoint, err)
	}
	return resp
}

func getNoRedirectWithCookies(t *testing.T, baseClient *http.Client, endpoint string, cookies ...*http.Cookie) *http.Response {
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

func noRedirectClient(base *http.Client) *http.Client {
	clientCopy := *base
	clientCopy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &clientCopy
}

func cookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func decodeRefreshCSRF(t *testing.T, encoded string, validator tokens.Validator) string {
	t.Helper()
	var refreshToken tokens.RefreshToken
	if err := refreshToken.Decode(encoded, validator); err != nil {
		t.Fatalf("RefreshToken.Decode failed: %v", err)
	}
	return refreshToken.Secret()
}

func decodeWireData(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("json unmarshal data failed: %v", err)
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	return buf.String()
}

func mustURL(t *testing.T, value string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatalf("url.Parse failed: %v", err)
	}
	return parsed
}

func stringPtr(value string) *string {
	return &value
}
