---
name: integrate-testing
description: Use this skill when integrating an external Go project with consent testing utilities. Triggers include testing authenticated routes, setting up dev mode login, writing tests for authorization, using TestVerifier, creating authenticated test requests, or local development without a real consent server.
---

# Integrate External Project with Consent Testing

## Overview

The `git.sr.ht/~jakintosh/consent/pkg/testing` package provides utilities for testing applications that integrate with consent without needing network access or a running server.

## Key Components

### TestVerifier

Implements `client.Verifier` for testing:

```go
import consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"

tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")
```

Parameters:
- `domain`: Consent server domain (token issuer claim)
- `audience`: Application identifier (token audience claim)

### DefaultTestSubject

`consenttesting.DefaultTestSubject` ("alice") is the default user identity for dev/test flows.

## Integration Steps

### 1. Add Dependency

```bash
go get git.sr.ht/~jakintosh/consent
```

### 2. Design for Testability

Depend on `client.Verifier` interface, not concrete types:

```go
import "git.sr.ht/~jakintosh/consent/pkg/client"

type MyApp struct {
    auth client.Verifier  // Interface for testability
}
```

### 3. Write Tests

```go
import (
    "net/http/httptest"
    "testing"
    consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func TestProtectedRoute(t *testing.T) {
    tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")
    app := myapp.NewApp(tv)

    req, _ := tv.AuthenticatedRequest("GET", "/api/profile", consenttesting.DefaultTestSubject)
    rr := httptest.NewRecorder()
    app.Router().ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rr.Code)
    }
}
```

### 4. Test Token Expiration

```go
func TestExpiredToken(t *testing.T) {
    tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")
    env := tv.TestEnv()

    accessToken, _ := env.IssueAccessToken(consenttesting.DefaultTestSubject, -1*time.Hour)
    req, _ := http.NewRequest("GET", "/api/profile", nil)
    env.AddAccessTokenCookie(req, accessToken)
    // Test expired token handling...
}
```

### 5. Test CSRF Protection

```go
func TestCSRFProtection(t *testing.T) {
    tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")
    env := tv.TestEnv()

    refreshToken, _ := env.IssueRefreshToken(consenttesting.DefaultTestSubject, time.Hour)
    accessToken, _ := env.IssueAccessToken(consenttesting.DefaultTestSubject, time.Hour)

    req, _ := http.NewRequest("POST", "/api/settings?csrf="+refreshToken.Secret(), nil)
    env.AddAuthCookies(req, accessToken, refreshToken)
    // Test CSRF-protected endpoint...
}
```

### 6. Development Mode (Optional)

For local browser-based development without a consent server:

```go
tv := consenttesting.NewTestVerifier("consent.example.com", "my-app")

if devMode {
    http.HandleFunc("/dev/login", tv.HandleDevLogin())
    http.HandleFunc("/dev/logout", tv.HandleDevLogout())
}
```

**Warning**: Testing package uses insecure cookies (Secure=false) for localhost. Never use in production.

## TestEnv Direct Access

For more control:

```go
env := consenttesting.NewTestEnv("consent.example.com", "my-app")

accessToken, _ := env.IssueAccessToken("bob", 5*time.Minute)
refreshToken, _ := env.IssueRefreshToken("bob", 24*time.Hour)

env.SetTokenCookies(w, accessToken, refreshToken)
env.ClearTokenCookies(w)
```

## Checklist

- [ ] Application depends on `client.Verifier` interface
- [ ] Tests use `consenttesting.NewTestVerifier()`
- [ ] Tests use `tv.AuthenticatedRequest()` for authenticated requests
- [ ] Dev mode handlers conditionally enabled (not in production)
- [ ] CSRF testing uses `TestEnv` to access refresh token secrets
