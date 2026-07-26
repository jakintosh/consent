# Consent: Simplified OAuth for Server Applications

**Consent** is a streamlined authentication service that distills OAuth 2.0's authorization code flow into a server-focused solution. By simplifying OAuth's per-client secret management while maintaining cryptographic security, Consent provides secure authentication specifically tailored for backend server applications that *only target a browser-based front-end*.

## Core Architecture

The system consists of three main components: an authentication server, a client library for backend integration, and persistent storage. The **authentication server** hosts both a web interface for user login and a RESTful API for token operations. Core server runtime configuration lives under a config directory with a generated `config.yaml`, file-backed secrets, and operator environment metadata managed through the CLI. Mutable runtime state such as SQLite data lives under a separate data directory. Service registrations themselves are API-managed records stored in SQLite.

The **client library** provides server-side functionality for backend applications, including automatic authorization code handling, token validation with automatic refresh, and built-in CSRF protection. This library runs entirely on the application backend—browsers never see cryptographic operations, only cookies and redirects.

**Data persistence** uses SQLite tables for `identity`, durable per-user per-service `grant` records, and active `refresh` tokens. This keeps Consent small while separating Consent login from third-party authorization.

## Authentication Flow

The authorization process mirrors OAuth's security model while simplifying implementation. When users access a protected service, they're redirected to Consent's `/authorize` endpoint with a service identifier and one or more scopes. Consent first ensures the user has its own Consent session, then reuses or records durable grants before issuing a short-lived refresh token (10 seconds) as an authorization code.

The user is then redirected back to the service with this code, which the client application backend automatically exchanges for long-lived access and refresh tokens through the `/api/v1/auth/refresh` endpoint. This maintains OAuth's security benefits—the authorization code prevents long-lived token exposure in browser history—while streamlining the developer experience.

## Key Design Decisions

**Simplified Secret Management**: Unlike OAuth's per-client secrets, Consent uses a single ECDSA key pair distributed to all client backend servers that integrate with a particular Consent instance. The auth server holds the private signing key while client backends share the public verification key. This eliminates per-client secret management while maintaining cryptographic security through server-to-server communication. **A primary intended use case that this supports is where a sysadmin deploys multiple consent-enabled services on the same node, making key sharing between clients simple through symbolic links**.

**Integrated CSRF Protection**: Refresh tokens include cryptographic secrets that serve double duty as CSRF tokens, providing protection against cross-site request forgery without additional infrastructure.

**Token Rotation**: Refresh tokens are single-use and replaced on every refresh operation, limiting the damage from token compromise while maintaining session continuity.

**Backend-Only Cryptography**: All token operations happen server-side. Browsers interact only through secure cookies and redirects, never seeing cryptographic keys or performing validation logic.

## Operational Benefits

**Simplified Deployment**: Client applications do not need their own OAuth client secret. Deploy the Consent verification key to each backend and register each app as an integration so Consent can validate its audience, redirect URL, homepage, logo, and optional role requirements.

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

Consent client integration has two halves:

1. The consuming backend mounts Consent auth handlers and validates Consent-issued cookies with the shared verification key.
2. The Consent operator registers that backend as an integration, either from the admin UI by importing `/.well-known/consent-integration` or with `consent api integrations create`.

The integration record is not an OAuth client secret. It is public metadata that tells Consent which app is requesting authorization, which JWT audience to issue, where authorization codes may be redirected, where users can launch the app, which logo to show, and which Consent roles are allowed to authorize it.

### Production Integration

```go
import (
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"git.sr.ht/~jakintosh/consent/pkg/client"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

const (
	consentURL      = "https://consent.example.com"
	consentIssuer   = "consent.example.com"
	publicURL       = "https://myapp.example.com"
	integrationName = "myapp"
)

var requestedScopes = []string{"identity", "profile"}

func main() {
	authClient, loginURL, manifestHandler, err := buildAuth()
	if err != nil {
		log.Fatal(err)
	}

	router := http.NewServeMux()
	router.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
	router.HandleFunc("/auth/logout", authClient.HandleLogout())
	router.HandleFunc(client.IntegrationManifestPath, manifestHandler)

	router.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		accessToken, err := authClient.VerifyAuthorization(w, r)
		if err != nil {
			http.Redirect(w, r, loginURL, http.StatusSeeOther)
			return
		}

		// Use accessToken.Subject() as the stable opaque Consent user key.
		// Do not key local data by handle; handles can change.
		_ = accessToken.Subject()
	})
}

func buildAuth() (*client.Client, string, http.HandlerFunc, error) {
	publicKey, err := loadVerificationKey("./config/verification_key")
	if err != nil {
		return nil, "", nil, err
	}

	appURL, err := url.Parse(publicURL)
	if err != nil {
		return nil, "", nil, err
	}
	audience := appURL.Host

	validator := tokens.InitClient(tokens.ClientOptions{
		VerificationKey: publicKey,
		IssuerDomain:    consentIssuer,
		ValidAudience:   audience,
	})
	authClient, err := client.Init(validator, consentURL)
	if err != nil {
		return nil, "", nil, err
	}

	loginURL := buildAuthorizeURL(consentURL, integrationName, requestedScopes)
	manifestHandler := client.HandleIntegrationManifest(client.IntegrationManifest{
		Name:           integrationName,
		Display:        "My App",
		Audience:       audience,
		Redirect:       publicURL + "/auth/callback",
		Homepage:       publicURL,
		Logo:           publicURL + "/static/consent-logo.png",
		ConsentIssuer:  consentIssuer,
		ConsentBaseURL: consentURL,
	})
	return authClient, loginURL, manifestHandler, nil
}

func loadVerificationKey(path string) (*ecdsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, err
	}
	ecdsaKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("verification key is not ECDSA")
	}
	return ecdsaKey, nil
}

func buildAuthorizeURL(consentURL, integration string, scopes []string) string {
	u, _ := url.Parse(strings.TrimRight(consentURL, "/") + "/authorize")
	q := u.Query()
	q.Set("integration", integration)
	for _, scope := range scopes {
		q.Add("scope", scope)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
```

For state-changing routes, read the CSRF secret when rendering a page, then require it when processing the POST:

```go
accessToken, csrf, err := authClient.VerifyAuthorizationGetCSRF(w, r)
// render csrf into a hidden form field or same-site request URL

accessToken, newCSRF, err := authClient.VerifyAuthorizationCheckCSRF(w, r, r.FormValue("csrf"))
// if the access token was refreshed, newCSRF may differ from the submitted value
```

If the app needs display/user profile data, request the `profile` scope along with `identity` and call Consent's userinfo endpoint through the client:

```go
userInfo, err := authClient.FetchUserInfo(r.Context(), accessToken.Encoded())
if err != nil {
	// Treat this as account setup or profile refresh failure.
	return
}

subject := userInfo.Sub
handle := userInfo.Profile.Handle
```

Compass follows this pattern in production: it uses Consent's `sub` as the durable foreign key in its own account table, caches the Consent handle for URLs like `/alice/`, refreshes that profile data periodically from `/api/v1/auth/userinfo`, and treats missing profile permission as an account setup error.

Available scopes are:

- `identity`: required for every authorization request; exposes the stable `sub` claim and permits use of the token as a user identity.
- `profile`: requires `identity`; exposes the user's Consent handle from `/api/v1/auth/userinfo`.

### Registering a Client App

The app must be registered with Consent before users can authorize it. From the admin UI, sign in as an admin, go to **Integrations**, choose **New Integration**, and import the app root URL. Consent fetches:

```text
https://myapp.example.com/.well-known/consent-integration
```

The manifest import requires the manifest version, name, display, audience, redirect, homepage, and logo fields. The redirect and homepage hosts must match the audience. After import, the admin can assign required roles to limit which Consent users may authorize the app.

The same registration can be done from the CLI:

```sh
consent api integrations create myapp \
  --config-dir ./config \
  --display "My App" \
  --audience myapp.example.com \
  --redirect https://myapp.example.com/auth/callback \
  --homepage https://myapp.example.com \
  --logo https://myapp.example.com/static/consent-logo.png
```

Add one or more `--required-roles role-name` flags if the integration should only be available to users with those Consent roles. Users and roles are managed separately:

```sh
consent api roles create editor --display "Editor" --config-dir ./config
consent api users create alice --password password123 --role editor --config-dir ./config
```

The consuming app should receive the Consent verification key out of band, typically by mounting or copying Consent's generated `verification_key.der` from the server config secrets directory. The app can name the mounted file however it wants; Compass, for example, expects `verification_key` in its own config directory.

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
consent api users create alice --password password123 --role admin --config-dir ./config
```

### Mock Deployment

Run a full local mock deployment with one real consent server login flow and three mock browser clients:

```sh
make mock-deployment
```

This target resets `./mock`, creates a consent config for `http://localhost:9000` using the default production mode, starts a temporary consent server with `--insecure-cookies` to seed an admin demo user and register three mock integrations through the API, and then starts:

- `http://localhost:9000` for the consent server
- `http://mock1.localhost:9001`
- `http://mock2.localhost:9002`
- `http://mock3.localhost:9003`
- `http://mock4.localhost:9004` as an unregistered app for testing admin manifest import

The mock deployment keeps the real login flow enabled while relaxing auth cookie security for local HTTP so Safari and other stricter browsers will store them on localhost.

The default demo credentials are:

```text
alice / alice123
```

To prepare the mock environment without starting the long-running processes, use:

```sh
make mock-deployment-init
```

### Docker Deployment

Build the production image from the repository root:

```sh
docker build -t consent:local .
```

The image uses the `consent` CLI as its entrypoint. Runtime config and secrets live under `/config`, and mutable SQLite data lives under `/data`; both paths should be mounted from the host.

For a first deployment, create host directories for those mounts:

```sh
mkdir -p deploy/config deploy/data
```

On Linux hosts, the container runs as UID/GID `10001`, so bind-mounted directories may need matching ownership:

```sh
sudo chown -R 10001:10001 deploy
```

Generate the config and file-backed secrets with an explicit one-off container run. Replace the URL and authority domain with the public values for your deployment:

```sh
docker run --rm \
  -v "$PWD/config:/config" \
  -v "$PWD/data:/data" \
  consent:local config init \
  --config-dir /config \
  --data-dir /data \
  --public-url https://consent.example.com \
  --authority-domain consent.example.com \
  --port 8000
```

Review and edit `./deploy/config/config.yaml`, then initialize the SQLite data directory:

```sh
docker run --rm \
  -v "$PWD/config:/config" \
  -v "$PWD/data:/data" \
  consent:local init --config-dir /config --data-dir /data
```

Start the server. This example binds to `8000` for a same-host reverse proxy such as Caddy:

```sh
docker run -d \
  --name consent \
  --restart unless-stopped \
  -p 8000:8000 \
  -v "$PWD/config:/config:ro" \
  -v "$PWD/data:/data" \
  consent:local
```

The repository ships a Dockerfile but no default Compose file. If you deploy through Compose, Dokploy, or another container platform, use the same image, mount persistent storage at `/config` and `/data`, run `config init` and `init` as intentional one-off commands, then run the default `serve` command for the long-lived container.

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
