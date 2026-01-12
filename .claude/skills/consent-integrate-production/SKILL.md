---
name: consent-integrate-production
description: Use this skill when integrating an external Go project with a production consent identity server. Triggers include setting up authentication, protecting routes with tokens, handling OAuth authorization codes, configuring ECDSA public keys, implementing CSRF protection, or connecting to a consent server.
---

# Integrate External Project with Production Consent Server

## Overview

The `git.sr.ht/~jakintosh/consent/pkg/client` and `pkg/tokens` packages provide production-ready authentication for backend applications.

## Prerequisites

- Running consent server URL (e.g., `https://consent.example.com`)
- Consent server's ECDSA public key (DER-encoded)
- Application's audience identifier registered with consent server
- Configured redirect URL for authorization code callback

## Integration Steps

### 1. Add Dependency

```bash
go get git.sr.ht/~jakintosh/consent
```

### 2. Load Public Key

```go
import (
    "crypto/ecdsa"
    "crypto/x509"
    "os"
)

func loadPublicKey(path string) (*ecdsa.PublicKey, error) {
    keyBytes, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    parsedKey, err := x509.ParsePKIXPublicKey(keyBytes)
    if err != nil {
        return nil, err
    }
    return parsedKey.(*ecdsa.PublicKey), nil
}
```

### 3. Initialize Client

```go
import (
    "git.sr.ht/~jakintosh/consent/pkg/client"
    "git.sr.ht/~jakintosh/consent/pkg/tokens"
)

publicKey, _ := loadPublicKey("/etc/secrets/verification_key.der")

validator := tokens.InitClient(
    publicKey,              // ECDSA public key
    "consent.example.com",  // Consent server domain (issuer)
    "myapp.example.com",    // Application's audience identifier
)

authClient := client.Init(validator, "https://consent.example.com")
```

### 4. Protect Routes

```go
func protectedHandler(auth client.Verifier) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        accessToken, err := auth.VerifyAuthorization(w, r)
        if err != nil {
            switch err {
            case client.ErrTokenAbsent:
                http.Redirect(w, r, "/login", http.StatusSeeOther)
            default:
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
            }
            return
        }
        username := accessToken.Subject()
        fmt.Fprintf(w, "Hello, %s!", username)
    }
}
```

### 5. Handle Authorization Code Callback

```go
http.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
```

### 6. Create Login Links

```go
loginURL := "https://consent.example.com/login?service=myapp@example.com"
http.Redirect(w, r, loginURL, http.StatusSeeOther)
```

### 7. CSRF Protection

**GET - provide token:**

```go
func showForm(auth client.Verifier) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        accessToken, csrfSecret, _ := auth.VerifyAuthorizationGetCSRF(w, r)
        fmt.Fprintf(w, `<form action="/save?csrf=%s" method="POST">...`, csrfSecret)
    }
}
```

**POST - verify token:**

```go
func handleSubmit(auth client.Verifier) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        csrf := r.URL.Query().Get("csrf")
        accessToken, _, err := auth.VerifyAuthorizationCheckCSRF(w, r, csrf)
        if err == client.ErrCSRFInvalid {
            http.Error(w, "CSRF validation failed", http.StatusForbidden)
            return
        }
        // Process request...
    }
}
```

### 8. Handle Logout

```go
func logoutHandler(auth *client.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        auth.ClearTokenCookies(w)
        http.Redirect(w, r, "/", http.StatusSeeOther)
    }
}
```

## Design for Testability

Depend on `client.Verifier` interface:

```go
type MyApp struct {
    auth client.Verifier  // Not *client.Client
}
```

In production: inject `*client.Client`
In tests: inject `*testing.TestVerifier`

For components needing both verification and auth code callback, use `client.AuthClient`.

## Error Types

| Error | Meaning |
|-------|---------|
| `client.ErrTokenAbsent` | No token cookie - user needs to log in |
| `client.ErrTokenInvalid` | Token malformed, expired, or wrong signature |
| `client.ErrCSRFInvalid` | CSRF secret mismatch |
| `client.ErrNetworkTokenRefresh` | Network error during refresh |

## Checklist

- [ ] ECDSA public key securely deployed
- [ ] Application depends on `client.Verifier` interface
- [ ] Token validator initialized with correct domain and audience
- [ ] Authorization code callback registered
- [ ] Protected routes use `VerifyAuthorization`
- [ ] Error handling covers all `client.Err*` cases
- [ ] Login links include correct service parameter
- [ ] Logout clears cookies with `ClearTokenCookies`
- [ ] CSRF protection for state-changing operations
