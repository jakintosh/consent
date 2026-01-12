# Skill: Integrate External Project with Production Consent Server

Use this skill when helping an external Go project integrate with a production consent identity server for:
- Protecting routes with authentication
- Handling OAuth authorization code flow
- Managing tokens and CSRF protection

## Overview

The `git.sr.ht/~jakintosh/consent/pkg/client` and `git.sr.ht/~jakintosh/consent/pkg/tokens` packages provide production-ready authentication for backend applications.

## Prerequisites

Before integrating, you need:
1. A running consent server URL (e.g., `https://consent.example.com`)
2. The consent server's ECDSA public key (DER-encoded)
3. Your application's audience identifier registered with the consent server
4. A configured redirect URL for the authorization code callback

## Integration Steps

### Step 1: Add Dependency

```bash
go get git.sr.ht/~jakintosh/consent
```

### Step 2: Load the Public Key

The consent server's public key must be distributed to your application. Load it at startup:

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

    ecdsaKey, ok := parsedKey.(*ecdsa.PublicKey)
    if !ok {
        return nil, fmt.Errorf("key is not ECDSA")
    }

    return ecdsaKey, nil
}
```

### Step 3: Initialize the Client

```go
import (
    "git.sr.ht/~jakintosh/consent/pkg/client"
    "git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func main() {
    // Load public key from file
    publicKey, err := loadPublicKey("/etc/secrets/verification_key.der")
    if err != nil {
        log.Fatal(err)
    }

    // Create token validator
    validator := tokens.InitClient(
        publicKey,              // ECDSA public key
        "consent.example.com",  // Consent server domain (issuer)
        "myapp.example.com",    // Your application's audience identifier
    )

    // Initialize consent client
    authClient := client.Init(validator, "https://consent.example.com")

    // Use authClient for route protection...
}
```

### Step 4: Protect Routes

Use `VerifyAuthorization` to protect your API routes:

```go
func protectedHandler(authClient client.Verifier) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        accessToken, err := authClient.VerifyAuthorization(w, r)
        if err != nil {
            switch err {
            case client.ErrTokenAbsent:
                // No token - redirect to login
                http.Redirect(w, r, "/login", http.StatusSeeOther)
            case client.ErrTokenInvalid:
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
            case client.ErrNetworkTokenRefresh:
                http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
            default:
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
            }
            return
        }

        // User is authenticated
        username := accessToken.Subject()
        fmt.Fprintf(w, "Hello, %s!", username)
    }
}
```

### Step 5: Handle Authorization Code Callback

Register the authorization code handler at your redirect URL:

```go
// This should match the redirect URL registered with the consent server
http.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
```

When users complete login at the consent server, they're redirected back with `?auth_code=...`. The handler:
1. Exchanges the code for access and refresh tokens
2. Sets secure cookies
3. Redirects to `/`

### Step 6: Create Login Links

Direct users to the consent server with your service identifier:

```go
func loginHandler(w http.ResponseWriter, r *http.Request) {
    // service parameter should match your registered service identifier
    loginURL := "https://consent.example.com/login?service=myapp@example.com"
    http.Redirect(w, r, loginURL, http.StatusSeeOther)
}
```

### Step 7: Implement CSRF Protection

For state-changing operations:

**GET request - provide CSRF token to client:**

```go
func showSettingsForm(authClient client.Verifier) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        accessToken, csrfSecret, err := authClient.VerifyAuthorizationGetCSRF(w, r)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Include csrfSecret in form as hidden field or query param
        fmt.Fprintf(w, `<form action="/settings?csrf=%s" method="POST">
            <button type="submit">Save</button>
        </form>`, csrfSecret)
    }
}
```

**POST request - verify CSRF token:**

```go
func updateSettings(authClient client.Verifier) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        csrfFromRequest := r.URL.Query().Get("csrf")

        accessToken, _, err := authClient.VerifyAuthorizationCheckCSRF(w, r, csrfFromRequest)
        if err == client.ErrCSRFInvalid {
            http.Error(w, "CSRF validation failed", http.StatusForbidden)
            return
        }
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Process the settings update...
    }
}
```

### Step 8: Handle Logout

Clear cookies on logout:

```go
func logoutHandler(authClient *client.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        authClient.ClearTokenCookies(w)
        http.Redirect(w, r, "/", http.StatusSeeOther)
    }
}
```

## Design for Testability

Depend on the `client.Verifier` interface rather than `*client.Client`:

```go
type MyApp struct {
    auth client.Verifier  // Interface for testability
}

func NewApp(auth client.Verifier) *MyApp {
    return &MyApp{auth: auth}
}

func (app *MyApp) ProtectedHandler(w http.ResponseWriter, r *http.Request) {
    accessToken, err := app.auth.VerifyAuthorization(w, r)
    // ...
}
```

In production, inject `*client.Client`. In tests, inject `*testing.TestVerifier`.

### When Using Authorization Code Handler

If a component needs both verification and the auth code callback, use `client.AuthClient`:

```go
type MyApp struct {
    auth client.AuthClient  // Verifier + AuthorizationCodeHandler
}
```

## Error Handling

The client package defines these error types:

| Error | Meaning |
|-------|---------|
| `client.ErrTokenAbsent` | No token cookie found - user needs to log in |
| `client.ErrTokenInvalid` | Token malformed, expired (refresh failed), or wrong signature |
| `client.ErrCSRFInvalid` | CSRF secret doesn't match (from VerifyAuthorizationCheckCSRF) |
| `client.ErrNetworkTokenRefresh` | Network error during refresh with consent server |

## Complete Example

```go
package main

import (
    "crypto/ecdsa"
    "crypto/x509"
    "fmt"
    "log"
    "net/http"
    "os"

    "git.sr.ht/~jakintosh/consent/pkg/client"
    "git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func main() {
    // Configuration
    consentURL := os.Getenv("CONSENT_URL")           // https://consent.example.com
    issuerDomain := os.Getenv("CONSENT_DOMAIN")      // consent.example.com
    audience := os.Getenv("APP_AUDIENCE")            // myapp.example.com
    keyPath := os.Getenv("VERIFICATION_KEY_PATH")    // /etc/secrets/verification_key.der

    // Load public key
    keyBytes, err := os.ReadFile(keyPath)
    if err != nil {
        log.Fatalf("Failed to read key: %v", err)
    }
    parsedKey, err := x509.ParsePKIXPublicKey(keyBytes)
    if err != nil {
        log.Fatalf("Failed to parse key: %v", err)
    }
    publicKey := parsedKey.(*ecdsa.PublicKey)

    // Initialize client
    validator := tokens.InitClient(publicKey, issuerDomain, audience)
    authClient := client.Init(validator, consentURL)

    // Routes
    http.HandleFunc("/", homeHandler(authClient))
    http.HandleFunc("/protected", protectedHandler(authClient))
    http.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
    http.HandleFunc("/logout", logoutHandler(authClient))

    log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(auth client.Verifier) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        accessToken, _ := auth.VerifyAuthorization(w, r)
        if accessToken != nil {
            fmt.Fprintf(w, "Welcome, %s!", accessToken.Subject())
        } else {
            fmt.Fprint(w, `<a href="https://consent.example.com/login?service=myapp">Login</a>`)
        }
    }
}

func protectedHandler(auth client.Verifier) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        accessToken, err := auth.VerifyAuthorization(w, r)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        fmt.Fprintf(w, "Secret data for %s", accessToken.Subject())
    }
}

func logoutHandler(auth *client.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        auth.ClearTokenCookies(w)
        http.Redirect(w, r, "/", http.StatusSeeOther)
    }
}
```

## Checklist

When integrating a project with a production consent server:

- [ ] ECDSA public key is securely deployed to the application
- [ ] Application depends on `client.Verifier` interface for testability
- [ ] Token validator initialized with correct domain and audience
- [ ] Authorization code callback registered at the redirect URL
- [ ] Protected routes use `VerifyAuthorization` or CSRF variants
- [ ] Error handling covers all `client.Err*` cases
- [ ] Login links point to consent server with correct service parameter
- [ ] Logout clears cookies with `ClearTokenCookies`
- [ ] CSRF protection implemented for state-changing operations
