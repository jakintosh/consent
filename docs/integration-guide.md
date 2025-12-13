# Consent Integration Guide for Go Applications

## Overview

Consent is a simplified OAuth 2.0 authentication service designed for server-focused applications. It eliminates the complexity of traditional OAuth by using a shared public key instead of per-client secrets.

**Key characteristics:**
- All client applications share one ECDSA public key (no client secrets)
- Tokens stored in HttpOnly cookies (browsers never see cryptographic operations)
- Refresh tokens are single-use and automatically rotated
- Services registered via JSON files (no database registration needed)

---

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Your App      │────▶│  Consent Server │────▶│    SQLite DB    │
│   (client)      │◀────│  (auth server)  │◀────│  (identities)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │
        │    Shared Public Key  │
        └───────────────────────┘
```

---

## Step 1: Register Your Service

Outside of the project's code, an administrator must create a JSON file in the Consent server's services directory:

**`/etc/consent/services/myapp`:**
```json
{
  "display": "My Application",
  "audience": "myapp.example.com",
  "redirect": "https://myapp.example.com/api/authorize"
}
```

| Field | Description |
|-------|-------------|
| `display` | User-friendly name shown on login page |
| `audience` | Unique identifier used in JWT validation |
| `redirect` | Where Consent redirects after login with `?auth_code=TOKEN` |

---

## Step 2: Obtain the Public Verification Key

Get `verification_key.der` from the Consent server administrator. This is the only credential your application needs.

---

## Step 3: Import the Client Library

```go
import (
    "git.sr.ht/~jakintosh/consent/pkg/client"
    "git.sr.ht/~jakintosh/consent/pkg/tokens"
)
```

---

## Step 4: Initialize at Application Startup

```go
func main() {
    // Load the public key
    keyBytes, err := os.ReadFile("/etc/myapp/secrets/verification_key.der")
    if err != nil {
        log.Fatal(err)
    }

    // Parse ECDSA public key
    pubKey, err := x509.ParsePKIXPublicKey(keyBytes)
    if err != nil {
        log.Fatal(err)
    }
    verificationKey := pubKey.(*ecdsa.PublicKey)

    // Create validator for your service
    // Args: public key, issuer domain, your service's audience
    validator := tokens.InitClient(
        verificationKey,
        "auth.example.com",      // Consent server's issuer domain
        "myapp.example.com",     // Must match "audience" in service JSON
    )

    // Initialize the client library
    client.Init(validator, "https://auth.example.com")

    // Set up routes...
}
```

---

## Step 5: Handle the Authorization Redirect

When users complete login, Consent redirects to your `redirect` URL with an `auth_code` parameter. The client library handles this automatically:

```go
http.HandleFunc("/api/authorize", client.HandleAuthorizationCode)
```

This handler:
1. Extracts the `auth_code` query parameter
2. Exchanges it for access + refresh tokens via `/api/refresh`
3. Sets secure HttpOnly cookies
4. Redirects to `/`

---

## Step 6: Protect Your Routes

### Basic Authentication Check

```go
func protectedHandler(w http.ResponseWriter, r *http.Request) {
    accessToken, err := client.VerifyAuthorization(w, r)
    if err != nil {
        // Not authenticated - redirect to login
        http.Redirect(w, r,
            "https://auth.example.com/login?service=myapp",
            http.StatusSeeOther)
        return
    }

    // User is authenticated
    username := accessToken.Subject()
    fmt.Fprintf(w, "Hello, %s!", username)
}
```
	
`VerifyAuthorization` automatically:
- Reads tokens from cookies
- Validates signatures and expiration
- Refreshes expired access tokens using the refresh token
- Updates cookies with new tokens

### With CSRF Protection (for forms)

```go
func formPageHandler(w http.ResponseWriter, r *http.Request) {
    accessToken, csrfSecret, err := client.VerifyAuthorizationGetCSRF(w, r)
    if err != nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    // Include CSRF token in your form
    tmpl.Execute(w, map[string]string{
        "Username":  accessToken.Subject(),
        "CSRFToken": csrfSecret,
    })
}

func formSubmitHandler(w http.ResponseWriter, r *http.Request) {
    csrf := r.FormValue("csrf_token")

    accessToken, _, err := client.VerifyAuthorizationCheckCSRF(w, r, csrf)
    if err != nil {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // CSRF validated, process the form
}
```

---

## Step 7: Handle Logout

```go
func logoutHandler(w http.ResponseWriter, r *http.Request) {
    // Clear token cookies
    client.ClearTokenCookies(w)

    // Optionally invalidate server-side by calling /api/logout
    // (requires POSTing the refresh token)

    http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

---

## Handling Users in Your Application

### The Identity Model

Consent stores minimal user data:
- `handle` (username) - the unique identifier
- `secret` (bcrypt hash) - password

Your application receives the `handle` as `accessToken.Subject()`.

### Linking to Your Application's User Data

**Option 1: Use handle as foreign key**
```go
// Your application's database
type UserProfile struct {
    Handle      string `db:"handle"`  // Links to Consent identity
    DisplayName string
    Email       string
    CreatedAt   time.Time
}

func getOrCreateProfile(handle string) (*UserProfile, error) {
    profile, err := db.GetProfileByHandle(handle)
    if err == sql.ErrNoRows {
        // First login - create profile
        profile = &UserProfile{
            Handle:    handle,
            CreatedAt: time.Now(),
        }
        err = db.CreateProfile(profile)
    }
    return profile, err
}
```

**Option 2: Middleware that enriches requests**
```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        accessToken, err := client.VerifyAuthorization(w, r)
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        handle := accessToken.Subject()
        profile, _ := getOrCreateProfile(handle)

        // Add to context
        ctx := context.WithValue(r.Context(), "user", profile)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Option 3: Lazy profile creation**
```go
func profileHandler(w http.ResponseWriter, r *http.Request) {
    token, _ := client.VerifyAuthorization(w, r)
    handle := token.Subject()

    profile, err := db.GetProfile(handle)
    if err == sql.ErrNoRows {
        // Show profile setup form
        showProfileSetup(w, handle)
        return
    }

    showProfile(w, profile)
}
```

---

## Complete Example Application

```go
package main

import (
    "context"
    "crypto/ecdsa"
    "crypto/x509"
    "html/template"
    "log"
    "net/http"
    "os"

    "git.sr.ht/~jakintosh/consent/pkg/client"
	    "git.sr.ht/~jakintosh/consent/pkg/tokens"
	)

const (
    AuthServerURL   = "https://auth.example.com"
    IssuerDomain    = "auth.example.com"
    ServiceAudience = "myapp.example.com"
    ServiceName     = "myapp"
)

func main() {
    // Initialize Consent client
    initConsent()

    // Public routes
    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/api/authorize", client.HandleAuthorizationCode)

    // Protected routes
    http.HandleFunc("/dashboard", dashboardHandler)
    http.HandleFunc("/logout", logoutHandler)

    log.Println("Server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func initConsent() {
    keyBytes, err := os.ReadFile("/etc/myapp/verification_key.der")
    if err != nil {
        log.Fatal("Failed to load verification key:", err)
    }

    pubKey, err := x509.ParsePKIXPublicKey(keyBytes)
    if err != nil {
        log.Fatal("Failed to parse public key:", err)
    }

    validator := tokens.InitClient(
        pubKey.(*ecdsa.PublicKey),
        IssuerDomain,
        ServiceAudience,
    )

    client.Init(validator, AuthServerURL)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    // Check if already logged in
    token, err := client.VerifyAuthorization(w, r)
    if err == nil {
        http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
        return
    }

    // Show login link
    loginURL := AuthServerURL + "/login?service=" + ServiceName
    tmpl := template.Must(template.New("home").Parse(`
        <h1>Welcome</h1>
        <a href="{{.LoginURL}}">Log In</a>
    `))
    tmpl.Execute(w, map[string]string{"LoginURL": loginURL})
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    token, err := client.VerifyAuthorization(w, r)
    if err != nil {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    username := token.Subject()

    tmpl := template.Must(template.New("dash").Parse(`
        <h1>Dashboard</h1>
        <p>Welcome, {{.Username}}!</p>
        <a href="/logout">Log Out</a>
    `))
    tmpl.Execute(w, map[string]string{"Username": username})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
    client.ClearTokenCookies(w)
    http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

---

## Token Lifetimes

| Token | Lifetime | Purpose |
|-------|----------|---------|
| Authorization Code | 10 seconds | Initial exchange after login redirect |
| Access Token | 30 minutes | API requests / page loads |
| Refresh Token | 72 hours | Getting new access tokens |

The client library handles refresh automatically - you just call `VerifyAuthorization()`.

---

## Security Notes

1. **Cookies are HttpOnly, Secure, SameSite=Strict** - JavaScript cannot access tokens
2. **Refresh tokens are single-use** - Each refresh invalidates the old token
3. **CSRF secrets embedded in refresh tokens** - Use `VerifyAuthorizationCheckCSRF` for state-changing operations
4. **Short authorization code window** - 10 seconds limits exposure during redirect

---

## Quick Reference

| Task | Function |
|------|----------|
| Check authentication | `client.VerifyAuthorization(w, r)` |
| Get CSRF secret | `client.VerifyAuthorizationGetCSRF(w, r)` |
| Validate CSRF | `client.VerifyAuthorizationCheckCSRF(w, r, csrf)` |
| Handle auth redirect | `client.HandleAuthorizationCode` |
| Clear cookies on logout | `client.ClearTokenCookies(w)` |
| Get username | `token.Subject()` |
