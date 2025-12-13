# Testing Guide for Consent

This guide explains how to test applications that integrate with Consent using the provided testing tools.

---

## Overview

Consent provides three testing components:

1. **`pkg/consenttest`** - Token-only helpers for fast unit tests (no network)
2. **`cmd/consent-testserver`** - Real HTTP server for integration/E2E tests (language-agnostic)
3. **`pkg/testharness`** - Go wrapper for easily spawning the test server

---

## When to Use Each Component

| Scenario | Tool | Reason |
|----------|------|--------|
| Unit testing handlers | `pkg/consenttest` | Fast, no network, isolated |
| Integration testing with real auth flow | `cmd/consent-testserver` | Exercises full OAuth flow |
| Go integration tests | `pkg/testharness` | Ergonomic Go API wrapping testserver |
| Non-Go integration tests | `cmd/consent-testserver` | Language-agnostic binary |

---

## `pkg/consenttest` - Token-Only Testing

### When to Use

Use `pkg/consenttest` when:
- Testing individual HTTP handlers
- You need "logged in" request behavior
- You don't need to exercise the actual login flow
- You want fast, isolated unit tests

### API Reference

```go
package consenttest

// Types
type Keys struct {
    SigningKey      *ecdsa.PrivateKey
    VerificationKey *ecdsa.PublicKey
    IssuerDomain    string
}

type Session struct {
    AccessToken      string
    RefreshToken     string
    CSRF             string
    AccessExpiresAt  time.Time
    RefreshExpiresAt time.Time
}

type CookieOptions struct {
    Secure   bool
    SameSite http.SameSite
    Path     string
    MaxAge   int  // if 0, derived from token exp
}

// Create test keys
func NewKeys(issuerDomain string) (*Keys, error)

// Create a test session with tokens
func NewSession(keys *Keys, subject, audience string, accessLifetime, refreshLifetime time.Duration) (*Session, error)

// Generate HTTP cookies from session
func Cookies(sess *Session, opts CookieOptions) (access, refresh *http.Cookie)

// Add cookies directly to a request
func AddCookies(r *http.Request, sess *Session, opts CookieOptions)

// Create a validator for your app
func Validator(keys *Keys, audience string) tokens.Validator
```

### Example: Handler Unit Test

```go
package myapp

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "git.sr.ht/~jakintosh/consent/pkg/client"
    "git.sr.ht/~jakintosh/consent/pkg/consenttest"
)

func TestDashboardHandler(t *testing.T) {
    // Generate test keys once
    keys, err := consenttest.NewKeys("test.example.com")
    if err != nil {
        t.Fatal(err)
    }

    // Initialize client with test validator
    validator := consenttest.Validator(keys, "myapp.example.com")
    client.Init(validator, "http://unused")

    // Create a test session
    session, err := consenttest.NewSession(
        keys,
        "testuser",
        "myapp.example.com",
        30*time.Minute,  // access token lifetime
        72*time.Hour,    // refresh token lifetime
    )
    if err != nil {
        t.Fatal(err)
    }

    // Create test request with cookies
    req := httptest.NewRequest("GET", "/dashboard", nil)
    consenttest.AddCookies(req, session, consenttest.CookieOptions{
        Secure:   false,
        SameSite: http.SameSiteStrictMode,
    })

    // Test the handler
    rr := httptest.NewRecorder()
    DashboardHandler(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", rr.Code)
    }

    // Verify the user was recognized
    body := rr.Body.String()
    if !strings.Contains(body, "testuser") {
        t.Error("expected response to contain username")
    }
}
```

### Example: CSRF-Protected Handler

```go
func TestFormSubmitHandler(t *testing.T) {
    keys, _ := consenttest.NewKeys("test.example.com")
    validator := consenttest.Validator(keys, "myapp.example.com")
    client.Init(validator, "http://unused")

    session, _ := consenttest.NewSession(
        keys, "testuser", "myapp.example.com",
        30*time.Minute, 72*time.Hour,
    )

    // Create POST request with CSRF token
    form := url.Values{}
    form.Add("csrf_token", session.CSRF)
    form.Add("data", "test data")

    req := httptest.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    consenttest.AddCookies(req, session, consenttest.CookieOptions{})

    rr := httptest.NewRecorder()
    FormSubmitHandler(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", rr.Code)
    }
}
```

---

## `cmd/consent-testserver` - Real Server Harness

### When to Use

Use `consent-testserver` when:
- You need to test the full OAuth authorization code flow
- You're testing from a non-Go language
- You want to test against a real Consent instance
- You need browser-like testing with cookie jars

### CLI Reference

```bash
consent-testserver [flags]
```

**Required Flags:**
- `--service-redirect` - URL where Consent redirects after login (your app's `/api/authorize` endpoint)

**Optional Flags:**
- `--listen` - Listen address (default: `127.0.0.1:0` for ephemeral port)
- `--issuer-domain` - JWT issuer domain (default: `consent.test`)
- `--service-name` - Service identifier (default: `test-service`)
- `--service-display` - Display name shown on login page (default: `Test Service`)
- `--service-audience` - JWT audience (default: `test-audience`)
- `--user` - User credentials as `handle:password` (repeatable, default: `test:test`)
- `--data-dir` - Data directory (uses temp dir if not set)
- `--keep` - Keep data directory after exit (default: false)
- `--quiet` - Suppress log output (default: false)

### JSON Output Contract

On startup, `consent-testserver` emits a single JSON line to stdout:

```json
{
  "base_url": "http://127.0.0.1:51234",
  "issuer_domain": "consent.test",
  "paths": {
    "data_dir": "/tmp/consent-testserver-xyz",
    "db_path": "/tmp/consent-testserver-xyz/db.sqlite",
    "services_dir": "/tmp/consent-testserver-xyz/services",
    "credentials_dir": "/tmp/consent-testserver-xyz/credentials",
    "verification_key_path": "/tmp/consent-testserver-xyz/credentials/verification_key.der"
  },
  "service": {
    "name": "test-service",
    "display": "Test Service",
    "audience": "test-audience",
    "redirect": "http://127.0.0.1:8080/api/authorize"
  },
  "users": [
    { "handle": "test", "password": "test" }
  ],
  "keys": {
    "verification_key_der_base64": "..."
  }
}
```

### Example: Manual Usage

```bash
# Start your app on port 8080
./myapp &

# Start consent-testserver pointing to your app
consent-testserver \
  --service-redirect http://localhost:8080/api/authorize \
  --service-audience myapp.local \
  --user alice:password123 \
  --user bob:password456

# Output:
# {"base_url":"http://127.0.0.1:54321",...}
```

Configure your app to use the `base_url` from the JSON output, then run your tests.

### Example: CI Integration (Python)

```python
import subprocess
import json

def start_consent_testserver(redirect_url):
    proc = subprocess.Popen(
        [
            "consent-testserver",
            "--service-redirect", redirect_url,
            "--quiet"
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )

    # Read first line (JSON contract)
    line = proc.stdout.readline()
    contract = json.loads(line)

    return proc, contract

# In your test setup
app_proc = start_app(port=8080)
consent_proc, contract = start_consent_testserver("http://localhost:8080/api/authorize")

# Configure your app
os.environ["AUTH_URL"] = contract["base_url"]
os.environ["ISSUER_DOMAIN"] = contract["issuer_domain"]
# ... copy verification key, etc.

# Run tests
run_integration_tests()

# Cleanup
consent_proc.terminate()
app_proc.terminate()
```

---

## `pkg/testharness` - Go Wrapper

### When to Use

Use `pkg/testharness` for Go integration tests where you want a convenient API without manually managing subprocesses.

### API Reference

```go
package testharness

type Config struct {
    RedirectURL     string // required
    ServiceName     string
    ServiceAudience string
    ServiceDisplay  string
    IssuerDomain    string
    Users           []User
    ListenAddr      string
    DataDir         string
    Keep            bool
    BinaryPath      string
    Quiet           bool
}

type User struct {
    Handle   string
    Password string
}

type Harness struct {
    BaseURL             string
    IssuerDomain        string
    VerificationKeyPath string
    VerificationKeyDER  []byte
    ServiceName         string
    ServiceAudience     string
    ServiceRedirect     string
    Users               []User
}

func Start(t *testing.T, cfg Config) *Harness
func (h *Harness) Close() error
```

### Example: Integration Test

```go
package myapp_test

import (
    "crypto/x509"
    "net/http/httptest"
    "testing"

    "git.sr.ht/~jakintosh/consent/pkg/client"
    "git.sr.ht/~jakintosh/consent/pkg/testharness"
    "git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func TestFullAuthFlow(t *testing.T) {
    // Start test app
    app := NewTestApp()
    server := httptest.NewServer(app)
    defer server.Close()

    // Start consent-testserver
    harness := testharness.Start(t, testharness.Config{
        RedirectURL: server.URL + "/api/authorize",
        ServiceAudience: "test-app",
    })

    // Configure app to use test harness
    pubKey, err := x509.ParsePKIXPublicKey(harness.VerificationKeyDER)
    if err != nil {
        t.Fatal(err)
    }

    validator := tokens.InitClient(
        pubKey.(*ecdsa.PublicKey),
        harness.IssuerDomain,
        harness.ServiceAudience,
    )

    client.Init(validator, harness.BaseURL)
    client.SetCookieOptions(client.CookieOptions{
        Secure:   false, // HTTP testing
        SameSite: http.SameSiteStrictMode,
        Path:     "/",
    })

    // Run tests that exercise the full login flow
    // ... (use harness.Users[0].Handle, harness.Users[0].Password to log in)
}
```

---

## Insecure Cookies Option

### Why It Exists

Consent's client library sets `Secure: true` on cookies by default, which is correct for production. However, this prevents cookies from being sent over plain HTTP connections during local development and testing.

### How to Enable (For Testing Only)

```go
import "git.sr.ht/~jakintosh/consent/pkg/client"

client.SetCookieOptions(client.CookieOptions{
    Secure:   false, // ⚠️ NEVER use in production
    SameSite: http.SameSiteStrictMode,
    Path:     "/",
})
```

When enabled, a prominent warning is logged:
```
WARNING: Insecure cookies enabled. This must NOT be used in production.
```

### Safety Notes

- This option should ONLY be used in tests or local development
- Never deploy code to production with `Secure: false`
- The warning is emitted even if logging is disabled to prevent accidental misuse

---

## Binary Resolution (testharness)

The `pkg/testharness` package finds the `consent-testserver` binary in this order:

1. `Config.BinaryPath` (if set)
2. `CONSENT_TESTSERVER_BIN` environment variable
3. `PATH` lookup

### Example: CI Setup

```bash
# Install testserver to a known location
go install git.sr.ht/~jakintosh/consent/cmd/consent-testserver@latest

# Point to it explicitly
export CONSENT_TESTSERVER_BIN=$(which consent-testserver)

# Or let it find via PATH
go test ./...
```

---

## Complete Example: Testing Workflow

### 1. Unit Tests (pkg/consenttest)

```go
// myapp/handlers_test.go
func TestProtectedHandler(t *testing.T) {
    keys, _ := consenttest.NewKeys("test.local")
    validator := consenttest.Validator(keys, "myapp")
    client.Init(validator, "http://unused")

    session, _ := consenttest.NewSession(keys, "alice", "myapp", 30*time.Minute, 72*time.Hour)

    req := httptest.NewRequest("GET", "/protected", nil)
    consenttest.AddCookies(req, session, consenttest.CookieOptions{})

    rr := httptest.NewRecorder()
    ProtectedHandler(rr, req)

    // Assertions...
}
```

### 2. Integration Tests (pkg/testharness)

```go
// myapp/integration_test.go
func TestLoginFlow(t *testing.T) {
    server := httptest.NewServer(myapp.NewRouter())
    defer server.Close()

    h := testharness.Start(t, testharness.Config{
        RedirectURL: server.URL + "/api/authorize",
        ServiceAudience: "myapp",
    })

    // Configure client for HTTP testing
    client.SetCookieOptions(client.CookieOptions{Secure: false})

    // Test login flow...
}
```

### 3. Non-Go Integration Tests

```python
# tests/test_auth.py
def test_auth_flow():
    consent = start_consent_testserver(
        redirect_url="http://localhost:8080/callback"
    )

    # Use requests library with session (cookie jar)
    session = requests.Session()

    # Navigate to login
    resp = session.get(f"{consent['base_url']}/login?service=test-service")

    # Submit credentials
    resp = session.post(
        f"{consent['base_url']}/api/login",
        json={
            "username": consent["users"][0]["handle"],
            "password": consent["users"][0]["password"],
            "service": "test-service"
        }
    )

    # Verify redirect to app with auth_code
    assert resp.history[0].status_code == 303
```

---

## Troubleshooting

### "consent-testserver binary not found"

**Solution:** Ensure `consent-testserver` is installed and in your `PATH`, or set `CONSENT_TESTSERVER_BIN`.

```bash
go install git.sr.ht/~jakintosh/consent/cmd/consent-testserver@latest
```

### "Cookies not being sent in tests"

**Solution:** For HTTP testing, enable insecure cookies:

```go
client.SetCookieOptions(client.CookieOptions{
    Secure: false,
    SameSite: http.SameSiteStrictMode,
    Path: "/",
})
```

### "Token validation fails with wrong audience"

**Solution:** Ensure the `audience` parameter matches across:
- `tokens.InitClient(key, issuer, audience)` in your app
- `--service-audience` flag to `consent-testserver`
- Service JSON `audience` field

---

## Summary

| Component | Use Case | Key Benefit |
|-----------|----------|-------------|
| `pkg/consenttest` | Unit tests | Fast, no network, isolated |
| `cmd/consent-testserver` | Integration tests | Real server, language-agnostic |
| `pkg/testharness` | Go integration tests | Ergonomic subprocess management |

Choose the tool that matches your testing needs. For most projects, use `pkg/consenttest` for unit tests and `pkg/testharness` for integration tests.
