# Skill: Integrate External Project with Consent Testing Server

Use this skill when helping an external Go project integrate with the `consent` testing utilities for:
- Testing authenticated routes without running a real consent server
- Setting up local development mode with browser-based login
- Writing tests that verify authorization behavior

## Overview

The `git.sr.ht/~jakintosh/consent/pkg/testing` package provides utilities for testing applications that integrate with consent without needing network access or a running server.

## Key Components

### TestVerifier

`TestVerifier` implements `client.Verifier` and can be injected into application code for testing:

```go
import (
    "git.sr.ht/~jakintosh/consent/pkg/testing"
)

tv := testing.NewTestVerifier("consent.example.com", "my-app")
```

Parameters:
- `domain`: The consent server domain (used in token issuer claim)
- `audience`: The application identifier (used in token audience claim)

### DefaultTestSubject

The constant `testing.DefaultTestSubject` ("alice") is the default user identity for dev/test flows.

## Integration Steps

### Step 1: Add Dependency

Add the consent module to the project's go.mod:

```bash
go get git.sr.ht/~jakintosh/consent
```

### Step 2: Design for Testability

The consuming application should depend on the `client.Verifier` interface rather than a concrete client:

```go
import "git.sr.ht/~jakintosh/consent/pkg/client"

type MyApp struct {
    auth client.Verifier  // Interface, not *client.Client
}

func NewApp(auth client.Verifier) *MyApp {
    return &MyApp{auth: auth}
}
```

### Step 3: Write Tests for Authenticated Routes

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"

    consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func TestProtectedRoute(t *testing.T) {
    // Create test verifier - no network, no real server
    tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")

    // Wire up application with test verifier
    app := myapp.NewApp(tv)
    router := app.Router()

    // Create authenticated request
    req, err := tv.AuthenticatedRequest("GET", "/api/profile", consenttesting.DefaultTestSubject)
    if err != nil {
        t.Fatal(err)
    }

    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rr.Code)
    }
}
```

### Step 4: Test Token Expiration

For testing expired token handling:

```go
func TestExpiredToken(t *testing.T) {
    tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")
    env := tv.TestEnv()

    // Issue an already-expired access token
    accessToken, err := env.IssueAccessToken(consenttesting.DefaultTestSubject, -1*time.Hour)
    if err != nil {
        t.Fatal(err)
    }

    req, _ := http.NewRequest("GET", "/api/profile", nil)
    env.AddAccessTokenCookie(req, accessToken)

    // Test that application handles expired tokens correctly...
}
```

### Step 5: Test CSRF Protection

```go
func TestCSRFProtection(t *testing.T) {
    tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")
    env := tv.TestEnv()

    // Issue tokens to get CSRF secret
    refreshToken, err := env.IssueRefreshToken(consenttesting.DefaultTestSubject, time.Hour)
    if err != nil {
        t.Fatal(err)
    }
    csrfSecret := refreshToken.Secret()

    accessToken, err := env.IssueAccessToken(consenttesting.DefaultTestSubject, time.Hour)
    if err != nil {
        t.Fatal(err)
    }

    // Build request with CSRF
    req, _ := http.NewRequest("POST", "/api/settings?csrf="+csrfSecret, nil)
    env.AddAuthCookies(req, accessToken, refreshToken)

    // Test CSRF-protected endpoint...
}
```

### Step 6: Set Up Development Mode (Optional)

For local browser-based development without running a consent server:

```go
import (
    consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func main() {
    tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")

    // Dev login/logout handlers - only enable in dev mode!
    if devMode {
        http.HandleFunc("/dev/login", tv.HandleDevLogin())
        http.HandleFunc("/dev/logout", tv.HandleDevLogout())
    }

    // Your application routes using tv as the Verifier
    app := NewApp(tv)
    http.Handle("/", app.Router())

    http.ListenAndServe(":8080", nil)
}
```

**Warning**: The testing package uses insecure cookies (Secure=false) to support `http://localhost`. Never use it in production.

## TestEnv Direct Access

For more control over token creation, use `TestEnv` directly:

```go
env := consenttesting.NewTestEnv("consent.example.com", "my-app")

// Issue tokens with custom lifetimes
accessToken, _ := env.IssueAccessToken("bob", 5*time.Minute)
refreshToken, _ := env.IssueRefreshToken("bob", 24*time.Hour)

// Set cookies on response writer
env.SetTokenCookies(w, accessToken, refreshToken)

// Clear cookies
env.ClearTokenCookies(w)
```

## Common Patterns

### Custom Test Subject

```go
req, _ := tv.AuthenticatedRequest("GET", "/api/profile", "custom-user")
```

### Testing Unauthorized Access

```go
func TestUnauthorized(t *testing.T) {
    tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")
    app := myapp.NewApp(tv)

    // Request without auth cookies
    req, _ := http.NewRequest("GET", "/api/profile", nil)
    rr := httptest.NewRecorder()
    app.Router().ServeHTTP(rr, req)

    if rr.Code != http.StatusUnauthorized {
        t.Errorf("expected 401, got %d", rr.Code)
    }
}
```

## Checklist

When integrating a project with consent testing:

- [ ] Application depends on `client.Verifier` interface, not `*client.Client`
- [ ] Tests use `testing.NewTestVerifier()` for authenticated route testing
- [ ] Tests use `tv.AuthenticatedRequest()` to create requests with valid cookies
- [ ] Dev mode handlers are conditionally enabled (not in production builds)
- [ ] CSRF testing uses `TestEnv` to access refresh token secrets
