# Consent: Simplified OAuth for Server Applications

**Consent** is a streamlined authentication service that distills OAuth 2.0's authorization code flow into a server-focused solution. By simplifying OAuth's per-client secret management while maintaining cryptographic security, Consent provides secure authentication specifically tailored for backend server applications that *only target a browser-based front-end*.

## Core Architecture

The system consists of three main components: an authentication server, a client library for backend integration, and persistent storage. The **authentication server** hosts both a web interface for user login and a RESTful API for token operations. Services register themselves through JSON configuration files that define display names, audiences, and redirect URLs, enabling dynamic service discovery without server restarts. These text files are expected to be deployed through simple tools like `rsync`, and rely on the admin to make smart choices about access and deployment. `consent` watches the service directory and updates its internal service registry whenever changes are made to the directory.

The **client library** provides server-side functionality for backend applications, including automatic authorization code handling, token validation with automatic refresh, and built-in CSRF protection. This library runs entirely on the application backend—browsers never see cryptographic operations, only cookies and redirects.

**Data persistence** uses SQLite with two core tables: `identity` for user credentials with bcrypt-hashed passwords, and `refresh` for tracking active refresh tokens. This simple schema supports the essential authentication operations without unnecessary complexity.

## Authentication Flow

The authorization process mirrors OAuth's security model while simplifying implementation. When users access a protected service, they're redirected to the authentication server with a service identifier. After successful authentication via web form or JSON API, the server issues a short-lived refresh token (10 seconds) as an authorization code.

The user is then redirected back to the service with this code, which the client application backend automatically exchanges for long-lived access and refresh tokens through the `/api/refresh` endpoint. This maintains OAuth's security benefits—the authorization code prevents long-lived token exposure in browser history—while streamlining the developer experience.

## Key Design Decisions

**Simplified Secret Management**: Unlike OAuth's per-client secrets, Consent uses a single ECDSA key pair distributed to all client backend servers that integrate with a particular Consent instance. The auth server holds the private signing key while client backends share the public verification key. This eliminates per-client registration complexity while maintaining cryptographic security through server-to-server communication. **A primary intended use case that this supports is where a sysadmin deploys multiple consent-enabled services on the same node, making key sharing between clients simple through symoblic links**.

**Integrated CSRF Protection**: Refresh tokens include cryptographic secrets that serve double duty as CSRF tokens, providing protection against cross-site request forgery without additional infrastructure.

**Token Rotation**: Refresh tokens are single-use and replaced on every refresh operation, limiting the damage from token compromise while maintaining session continuity.

**Backend-Only Cryptography**: All token operations happen server-side. Browsers interact only through secure cookies and redirects, never seeing cryptographic keys or performing validation logic.

## Operational Benefits

**Simplified Deployment**: Client applications are "pre-authorized" by virtue of having the verification key. No per-client registration process is required—just distribute the key pair and configure service definitions.

**Easier Key Management**: Single key pair per Consent instance instead of managing individual client secrets. Key rotation affects all clients uniformly.

**Reduced Implementation Complexity**: The client library handles all token lifecycle management automatically. Applications simply use verification functions to protect routes without implementing OAuth flows manually.

## Security Model

Consent maintains OAuth's proven security approach:
- No credentials traverse the browser directly
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
