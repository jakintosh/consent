# Consent: Simplified OAuth for Server Applications

**Consent** is a streamlined authentication service that distills OAuth 2.0's authorization code flow into a server-focused solution. By simplifying OAuth's per-client secret management while maintaining cryptographic security, Consent provides secure authentication specifically tailored for backend server applications that *only target a browser-based front-end*.

## Core Architecture

The system consists of three main components: an authentication server, a client library for backend integration, and persistent storage. The **authentication server** hosts both a web interface for user login and a RESTful API for token operations. Core server runtime configuration lives under a config directory with a generated `config.yaml`, file-backed secrets, and operator environment metadata managed through the CLI. Mutable runtime state such as SQLite data lives under a separate data directory. Service registrations themselves are API-managed records stored in SQLite.

The **client library** provides server-side functionality for backend applications, including automatic authorization code handling, token validation with automatic refresh, and built-in CSRF protection. This library runs entirely on the application backend—browsers never see cryptographic operations, only cookies and redirects.

**Data persistence** uses SQLite tables for `identity`, durable per-user per-service `grant` records, and active `refresh` tokens. This keeps Consent small while separating Consent login from third-party authorization.

## Authentication Flow

The authorization process mirrors OAuth's security model while simplifying implementation. When users access a protected service, they're redirected to Consent's `/authorize` endpoint with a service identifier and one or more scopes. Consent first ensures the user has its own Consent session, then reuses or records durable grants before issuing a short-lived refresh token (10 seconds) as an authorization code.

The user is then redirected back to the service with this code, which the client application backend automatically exchanges for long-lived access and refresh tokens through the `/api/v1/refresh` endpoint. This maintains OAuth's security benefits—the authorization code prevents long-lived token exposure in browser history—while streamlining the developer experience.

## Key Design Decisions

**Simplified Secret Management**: Unlike OAuth's per-client secrets, Consent uses a single ECDSA key pair distributed to all client backend servers that integrate with a particular Consent instance. The auth server holds the private signing key while client backends share the public verification key. This eliminates per-client registration complexity while maintaining cryptographic security through server-to-server communication. **A primary intended use case that this supports is where a sysadmin deploys multiple consent-enabled services on the same node, making key sharing between clients simple through symoblic links**.

**Integrated CSRF Protection**: Refresh tokens include cryptographic secrets that serve double duty as CSRF tokens, providing protection against cross-site request forgery without additional infrastructure.

**Token Rotation**: Refresh tokens are single-use and replaced on every refresh operation, limiting the damage from token compromise while maintaining session continuity.

**Backend-Only Cryptography**: All token operations happen server-side. Browsers interact only through secure cookies and redirects, never seeing cryptographic keys or performing validation logic.

## Operational Benefits

**Simplified Deployment**: Client applications are "pre-authorized" by virtue of having the verification key. No per-client registration process is required—just distribute the key pair and register service definitions through the API.

**Easier Key Management**: Single key pair per Consent instance instead of managing individual client secrets. Key rotation affects all clients uniformly.

**Reduced Implementation Complexity**: The client library handles all token lifecycle management automatically. Applications simply use verification functions to protect routes without implementing OAuth flows manually.

## Security Model

Consent maintains OAuth's proven security approach:
- Third-party services never receive user credentials directly
- Tokens have limited lifetimes with automatic refresh
- ECDSA signatures prevent token tampering
- Authorization codes are short-lived (10 seconds) to minimize exposure window
- HttpOnly, Secure, SameSite cookies prevent XSS-based token theft

The server-to-server architecture ensures that cryptographic operations remain secure while eliminating the complexity that often leads to implementation vulnerabilities in OAuth deployments.

## Use Cases

Consent is ideal for:
- Multiple backend services needing shared authentication
- Microservice architectures requiring lightweight auth
- Organizations wanting OAuth-level security without OAuth complexity
- Applications where simplified key distribution outweighs per-client secret isolation

In sum, Consent is designed to provide easy "login with facebook" style authentication for small, open source, community-scale software projects.

This approach reduces OAuth implementation time from weeks to hours while maintaining the security guarantees that make OAuth suitable for production authentication systems.

## Package Overview

The `pkg/` directory contains public packages for consuming projects:

- **`pkg/client`**: Client library for backend applications integrating with a consent server. Provides the `Verifier` interface for protecting routes, automatic token refresh, and CSRF protection.
- **`pkg/tokens`**: JWT token utilities including `InitClient` for creating token validators with ECDSA public keys.
- **`pkg/testing`**: Test utilities for consuming projects. Provides `TestVerifier` (implements `client.Verifier`) for testing authenticated routes without a real consent server, plus dev login handlers for local browser-based development.

The `cmd/` directory also includes development-focused binaries:

- **`cmd/dev-client`**: A local integration playground for testing how a service integrates with consent. This command always enables client development mode and is not intended for real world usage.

## Integration Guide

### Production Integration

```go
import (
    "git.sr.ht/~jakintosh/consent/pkg/client"
    "git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// Initialize with consent server's public key
clientOpts := tokens.ClientOptions{
    VerificationKey: publicKey,
    IssuerDomain:    "consent.example.com",
    ValidAudience:   "myapp.example.com",
}
validator := tokens.InitClient(clientOpts)
authClient := client.Init(validator, "https://consent.example.com")

// Protect routes
func protectedHandler(w http.ResponseWriter, r *http.Request) {
    accessToken, err := authClient.VerifyAuthorization(w, r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    // Use accessToken.Subject() as a stable opaque user key
    // Call Consent's /api/v1/me endpoint for scoped profile data
}

// Handle authorization code callback
http.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
```

### Testing Integration

```go
import (
    "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func TestProtectedRoute(t *testing.T) {
    // TestVerifier implements client.Verifier - no network required
    tv := testing.NewTestVerifier("consent.example.com", "my-app")

    router := myapp.NewRouter(tv)  // Inject as Verifier interface

    req, _ := tv.AuthenticatedRequest("GET", "/api/profile", testing.DefaultTestSubject)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rr.Code)
    }
}
```

### Development Mode

For local browser-based development without running a consent server:

```go
tv := testing.NewTestVerifier("consent.example.com", "my-app")

http.HandleFunc("/dev/login", tv.HandleDevLogin())
http.HandleFunc("/dev/logout", tv.HandleDevLogout())
```

### Local Integration Workflow

Bootstrap a local consent instance with the public CLI and Makefile:

```sh
make init
make run-local
```

`make init` builds the binary, generates baseline config and secrets under `./config`, initializes mutable runtime state under `./data`, and stores a matching local operator environment with `consent env create`. The verification key is written to `./config/secrets/verification_key.der`.

The generated `./config/config.yaml` uses production defaults. `make init` passes `--dev-mode`, so it looks like this for a local dev setup:

```yaml
server:
  publicURL: http://localhost:9001
  issuerDomain: localhost
  port: 9001
  devMode: true
```

That authored config file is only part of the runtime layout. `consent config init` also creates the signing key, verification key, bootstrap API key, and the directories used by the server.

Run the local dev client against that generated config with:

```sh
go run ./cmd/dev-client --config-dir ./config
```

After that, start the server with:

```sh
make run-local
```

Useful config commands:

```sh
consent config show --config-dir ./config
consent config show --resolved --config-dir ./config --data-dir ./data
```

Create a local user through the API with:

```sh
consent api register alice password123 --config-dir ./config
```

### Mock Deployment

Run a full local mock deployment with one real consent server login flow and three mock browser clients:

```sh
make mock-deployment
```

This target resets `./mock`, creates a consent config for `http://localhost:9000` using the default production mode, starts a temporary consent server with `--insecure-cookies` to seed a demo user and register three mock services through the API, and then starts:

- `http://localhost:9000` for the consent server
- `http://mock1.localhost:9001`
- `http://mock2.localhost:9002`
- `http://mock3.localhost:9003`

The mock deployment keeps the real login flow enabled while relaxing auth cookie security for local HTTP so Safari and other stricter browsers will store them on localhost.

The default demo credentials are:

```text
alice / alice123
```

To prepare the mock environment without starting the long-running processes, use:

```sh
make mock-deployment-init
```

## Interface Design

For testability, depend on the `client.Verifier` interface rather than `*client.Client`:

```go
type MyApp struct {
    auth client.Verifier  // Not *client.Client
}
```

In production, pass a `*client.Client` (which implements `Verifier`).
In tests, pass a `*testing.TestVerifier`.

If your component needs both verification and the auth code callback, use `client.AuthClient` (combines `Verifier` + `AuthorizationCodeHandler`).
