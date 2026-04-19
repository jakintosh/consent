package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	consentconfig "git.sr.ht/~jakintosh/consent/internal/config"
	"git.sr.ht/~jakintosh/consent/pkg/client"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

const (
	defaultAuthURL      = "http://localhost:9001"
	defaultIssuerDomain = "localhost"
	defaultPort         = "10000"
	defaultServiceName  = "example@localhost"
	defaultConfigDir    = "./config"
)

type config struct {
	AuthURL             string
	IssuerDomain        string
	Port                string
	Service             string
	Audience            string
	VerificationKeyPath string
}

var root = &args.Command{
	Name: "dev-client",
	Help: "Development-only OAuth client playground",
	Config: &args.Config{
		Author: "jakintosh",
		HelpOption: &args.HelpOption{
			Short: 'h',
			Long:  "help",
		},
	},
	Options: []args.Option{
		{
			Short: 'v',
			Long:  "verbose",
			Type:  args.OptionTypeFlag,
			Help:  "Verbose output",
		},
		{
			Long: "auth-url",
			Type: args.OptionTypeParameter,
			Help: "Consent server URL (default: http://localhost:9001)",
		},
		{
			Long: "issuer-domain",
			Type: args.OptionTypeParameter,
			Help: "JWT issuer domain (default: localhost)",
		},
		{
			Long: "port",
			Type: args.OptionTypeParameter,
			Help: "HTTP listen port (default: 10000)",
		},
		{
			Long: "service",
			Type: args.OptionTypeParameter,
			Help: "Service name for consent login (default: example@localhost)",
		},
		{
			Long: "audience",
			Type: args.OptionTypeParameter,
			Help: "JWT audience (default: localhost:<port>)",
		},
		{
			Long: "config-dir",
			Type: args.OptionTypeParameter,
			Help: "Path to consent config directory (default: ./config)",
		},
		{
			Long: "verification-key",
			Type: args.OptionTypeParameter,
			Help: "Path to verification key DER file (default: <config-dir>/secrets/verification_key.der)",
		},
	},
	Handler: func(i *args.Input) error {
		verbose := i.GetFlag("verbose")

		cfg, err := parseConfig(i)
		if err != nil {
			return err
		}

		if verbose {
			log.Println("Starting development OAuth client...")
			log.Printf("  Auth URL: %s", cfg.AuthURL)
			log.Printf("  Issuer domain: %s", cfg.IssuerDomain)
			log.Printf("  Service: %s", cfg.Service)
			log.Printf("  Audience: %s", cfg.Audience)
			log.Printf("  Verification key: %s", cfg.VerificationKeyPath)
			log.Printf("  Port: %s", cfg.Port)
		}

		verificationKeyBytes, err := os.ReadFile(cfg.VerificationKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load verification key %q: %w", cfg.VerificationKeyPath, err)
		}

		verificationKey, err := decodePublicKey(verificationKeyBytes)
		if err != nil {
			return err
		}

		opts := tokens.ClientOptions{
			VerificationKey: verificationKey,
			IssuerDomain:    cfg.IssuerDomain,
			ValidAudience:   cfg.Audience,
		}
		tkValidator := tokens.InitClient(opts)
		authClient := client.Init(tkValidator, cfg.AuthURL)
		authClient.EnableDevelopmentMode()
		if verbose {
			authClient.SetLogLevel(client.LogLevelDebug)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/", homeHandler(authClient, cfg))
		mux.HandleFunc("/api/example", exampleHandler(authClient, cfg))
		mux.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
		mux.HandleFunc("/logout", authClient.HandleLogout())

		if verbose {
			log.Printf("Listening on :%s", cfg.Port)
		}

		if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	},
}

func main() {
	root.Parse()
}

func parseConfig(i *args.Input) (config, error) {
	authURL := optionOrDefault(i, "auth-url", defaultAuthURL)
	authURL, err := normalizeAuthURL(authURL)
	if err != nil {
		return config{}, err
	}

	issuerDomain := optionOrDefault(i, "issuer-domain", defaultIssuerDomain)
	if strings.TrimSpace(issuerDomain) == "" {
		return config{}, fmt.Errorf("--issuer-domain cannot be empty")
	}

	port := optionOrDefault(i, "port", defaultPort)
	if err := validatePort(port); err != nil {
		return config{}, err
	}

	serviceName := optionOrDefault(i, "service", defaultServiceName)
	if strings.TrimSpace(serviceName) == "" {
		return config{}, fmt.Errorf("--service cannot be empty")
	}

	audience := optionOrDefault(i, "audience", fmt.Sprintf("localhost:%s", port))
	if strings.TrimSpace(audience) == "" {
		return config{}, fmt.Errorf("--audience cannot be empty")
	}

	configDir := optionOrDefault(i, "config-dir", defaultConfigDir)
	if strings.TrimSpace(configDir) == "" {
		return config{}, fmt.Errorf("--config-dir cannot be empty")
	}

	roots, err := consentconfig.ResolveRoots(configDir, "")
	if err != nil {
		return config{}, err
	}

	defaultVerificationKeyPath := consentconfig.BuildPaths(roots).VerificationKeyFile
	verificationKeyPath := optionOrDefault(i, "verification-key", defaultVerificationKeyPath)
	if strings.TrimSpace(verificationKeyPath) == "" {
		return config{}, fmt.Errorf("--verification-key cannot be empty")
	}

	return config{
		AuthURL:             authURL,
		IssuerDomain:        issuerDomain,
		Port:                port,
		Service:             serviceName,
		Audience:            audience,
		VerificationKeyPath: verificationKeyPath,
	}, nil
}

func optionOrDefault(i *args.Input, optionName string, defaultValue string) string {
	if param := i.GetParameter(optionName); param != nil {
		return strings.TrimSpace(*param)
	}

	return defaultValue
}

func normalizeAuthURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return "", fmt.Errorf("invalid --auth-url: expected absolute URL with scheme and host")
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid --auth-url: expected absolute URL with scheme and host")
	}

	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("invalid --auth-url: path is not allowed")
	}

	return (&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String(), nil
}

func validatePort(port string) error {
	if strings.TrimSpace(port) == "" {
		return fmt.Errorf("--port cannot be empty")
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid --port %q: expected an integer", port)
	}
	if portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("invalid --port %q: out of range", port)
	}

	return nil
}

func homeHandler(c client.Verifier, cfg config) http.HandlerFunc {
	loginURL := fmt.Sprintf("%s/authorize?service=%s&scope=identity&scope=profile", cfg.AuthURL, url.QueryEscape(cfg.Service))

	return func(w http.ResponseWriter, r *http.Request) {
		page := homePageData{
			Service:              cfg.Service,
			Audience:             cfg.Audience,
			AuthURL:              cfg.AuthURL,
			CurrentOrigin:        requestOrigin(r),
			CurrentHost:          r.Host,
			LoginURL:             loginURL,
			AccessCookiePresent:  cookiePresent(r, "accessToken"),
			RefreshCookiePresent: cookiePresent(r, "refreshToken"),
		}

		accessToken, csrf, err := c.VerifyAuthorizationGetCSRF(w, r)
		if err != nil {
			if !errors.Is(err, client.ErrTokenAbsent) {
				log.Printf("%s: failed to verify authorization: %v", r.RequestURI, err)
				page.AuthError = err.Error()
			}
		}

		if accessToken != nil {
			page.Authenticated = true
			page.CSRF = csrf
			page.Scopes = strings.Join(accessToken.Scopes(), ", ")
			page.Subject = accessToken.Subject()
		} else {
			page.AuthHint = "Complete login in Consent and this page should refresh into the signed-in state."
			if !page.AccessCookiePresent && !page.RefreshCookiePresent {
				page.AuthHint = "No auth cookies are present yet. If you just completed login, the callback may not have stored cookies for this host."
			}
		}

		if err := homeTemplate.Execute(w, page); err != nil {
			log.Printf("%s: failed to render home page: %v", r.RequestURI, err)
		}
	}
}

func exampleHandler(c client.Verifier, cfg config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := examplePageData{
			Service:       cfg.Service,
			CurrentOrigin: requestOrigin(r),
			CurrentHost:   r.Host,
		}

		csrf := r.URL.Query().Get("csrf")
		accessToken, csrf, err := c.VerifyAuthorizationCheckCSRF(w, r, csrf)
		if err != nil {
			log.Printf("%s: failed to verify authorization: %v", r.RequestURI, err)
			page.AuthError = err.Error()
		}

		if accessToken != nil {
			page.Authenticated = true
			page.Handle = fetchProfileHandle(r, cfg.AuthURL, accessToken.Encoded())
			page.CSRF = csrf
			page.Scopes = strings.Join(accessToken.Scopes(), ", ")
			page.Subject = accessToken.Subject()
		} else {
			page.AuthHint = "This route needs a valid session and matching CSRF token from the home page."
		}

		if err := exampleTemplate.Execute(w, page); err != nil {
			log.Printf("%s: failed to render example page: %v", r.RequestURI, err)
		}
	}
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func cookiePresent(r *http.Request, name string) bool {
	_, err := r.Cookie(name)
	return err == nil
}

func fetchProfileHandle(r *http.Request, authURL string, accessToken string) string {
	request, err := http.NewRequest(http.MethodGet, authURL+"/api/v1/me", nil)
	if err != nil {
		log.Printf("failed to create /api/v1/me request: %v", err)
		return ""
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("failed to call /api/v1/me: %v", err)
		return ""
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Printf("/api/v1/me returned status %d", response.StatusCode)
		return ""
	}

	var body struct {
		Data struct {
			Profile *struct {
				Handle string `json:"handle"`
			} `json:"profile"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		log.Printf("failed to decode /api/v1/me response: %v", err)
		return ""
	}
	if body.Data.Profile == nil {
		return ""
	}
	return body.Data.Profile.Handle
}

func decodePublicKey(bytes []byte) (*ecdsa.PublicKey, error) {
	parsedKey, err := x509.ParsePKIXPublicKey(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ecdsa verification key from DER: %w", err)
	}

	ecdsaKey, ok := parsedKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast parsed key as *ecdsa.PublicKey")
	}

	return ecdsaKey, nil
}

type homePageData struct {
	Authenticated        bool
	Service              string
	Audience             string
	AuthURL              string
	CurrentOrigin        string
	CurrentHost          string
	LoginURL             string
	LogoutURL            string
	CSRF                 string
	Subject              string
	Scopes               string
	AuthHint             string
	AuthError            string
	AccessCookiePresent  bool
	RefreshCookiePresent bool
}

type examplePageData struct {
	Authenticated bool
	Service       string
	CurrentOrigin string
	CurrentHost   string
	Handle        string
	Subject       string
	Scopes        string
	CSRF          string
	AuthHint      string
	AuthError     string
}

var homeTemplate = template.Must(template.New("home").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>Mock Client</title>
	<style>
		* { box-sizing: border-box; }
		:root { color-scheme: light; font-family: sans-serif; font-size: 12pt; }
		body { margin: 0; background: #f7f1fb; color: #22132f; }
		header { background: #7521b0; color: #fff; padding: 1.25rem 1rem; }
		header h1, header p { margin: 0; }
		header p { margin-top: 0.35rem; opacity: 0.9; }
		main { max-width: 760px; margin: 0 auto; padding: 1rem; }
		.panel { background: #fff; border: 1px solid #e1d2ee; border-radius: 16px; padding: 1rem; margin: 1rem 0; box-shadow: 0 10px 30px rgba(117, 33, 176, 0.08); }
		h2, h3 { margin-top: 0; color: #7521b0; }
		p { line-height: 1.5; }
		.actions { display: flex; gap: 0.75rem; flex-wrap: wrap; margin-top: 1rem; }
		.button { display: inline-block; padding: 0.8rem 1rem; border-radius: 999px; text-decoration: none; font-weight: 700; }
		.button-primary { background: #7521b0; color: #fff; }
		.button-secondary { background: #f4ecfa; color: #7521b0; }
		.grid { display: grid; gap: 0.75rem; }
		@media (min-width: 640px) { .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
		dl { margin: 0; }
		dt { font-size: 0.9rem; color: #694e80; }
		dd { margin: 0.2rem 0 0; font-family: monospace; overflow-wrap: anywhere; }
		.status { padding: 0.9rem 1rem; border-radius: 12px; }
		.status-ok { background: #eef9f0; color: #235d32; }
		.status-warn { background: #fff5e8; color: #85511c; }
		.status-error { background: #fdecec; color: #8f2332; }
		code { font-family: monospace; }
	</style>
</head>
<body>
	<header>
		<h1>Mock Client Playground</h1>
		<p>Development UI for checking Consent browser login and token cookies.</p>
	</header>
	<main>
		<section class="panel">
			<h2>{{if .Authenticated}}Signed In{{else}}Signed Out{{end}}</h2>
			{{if .Authenticated}}
				<p class="status status-ok">This host has both a valid session and a refresh token. The page verified your session server-side.</p>
			{{else if .AuthError}}
				<p class="status status-error">The client received cookies or tokens it could not validate: <code>{{.AuthError}}</code></p>
			{{else}}
				<p class="status status-warn">{{.AuthHint}}</p>
			{{end}}
			<div class="actions">
				{{if .Authenticated}}
					<a class="button button-primary" href="/api/example?csrf={{.CSRF}}">Call Example API</a>
					<a class="button button-secondary" href="/logout?csrf={{.CSRF}}">Log Out</a>
				{{else}}
					<a class="button button-primary" href="{{.LoginURL}}">Log In with Pollinator</a>
				{{end}}
			</div>
		</section>

		<section class="panel">
			<h3>Service Details</h3>
			<div class="grid">
				<dl><dt>Service</dt><dd>{{.Service}}</dd></dl>
				<dl><dt>Audience</dt><dd>{{.Audience}}</dd></dl>
				<dl><dt>Consent Server</dt><dd>{{.AuthURL}}</dd></dl>
				<dl><dt>Current Origin</dt><dd>{{.CurrentOrigin}}</dd></dl>
				<dl><dt>Current Host</dt><dd>{{.CurrentHost}}</dd></dl>
				{{if .Authenticated}}
					<dl><dt>Opaque Subject</dt><dd>{{.Subject}}</dd></dl>
					<dl><dt>Granted Scopes</dt><dd>{{.Scopes}}</dd></dl>
				{{end}}
			</div>
		</section>

		<section class="panel">
			<h3>Cookie Diagnostics</h3>
			<p>This view checks for incoming HTTP-only token cookies before it asks the shared client library to validate them.</p>
			<div class="grid">
				<dl><dt>accessToken cookie</dt><dd>{{if .AccessCookiePresent}}present{{else}}missing{{end}}</dd></dl>
				<dl><dt>refreshToken cookie</dt><dd>{{if .RefreshCookiePresent}}present{{else}}missing{{end}}</dd></dl>
			</div>
			<p>If you complete Consent login but still return here with both cookies missing, the browser likely did not store cookies for this host.</p>
		</section>

		<section class="panel">
			<h3>Host Notes</h3>
			<p>Mock services are registered on specific hosts like <code>mock1.localhost:9001</code>. The callback and the page you revisit need to use the same host, or the browser will treat them as separate cookie jars.</p>
		</section>
	</main>
</body>
</html>`))

var exampleTemplate = template.Must(template.New("example").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>Mock Client API Result</title>
	<style>
		* { box-sizing: border-box; }
		:root { font-family: sans-serif; font-size: 12pt; }
		body { margin: 0; background: #f7f1fb; color: #22132f; }
		main { max-width: 760px; margin: 0 auto; padding: 1rem; }
		.panel { background: #fff; border: 1px solid #e1d2ee; border-radius: 16px; padding: 1rem; margin: 1rem 0; box-shadow: 0 10px 30px rgba(117, 33, 176, 0.08); }
		h1, h2 { color: #7521b0; margin-top: 0; }
		a { color: #7521b0; text-underline-offset: 0.2em; }
		dl { margin: 0; }
		dt { font-size: 0.9rem; color: #694e80; }
		dd { margin: 0.2rem 0 0.9rem; font-family: monospace; overflow-wrap: anywhere; }
		.status { padding: 0.9rem 1rem; border-radius: 12px; }
		.status-ok { background: #eef9f0; color: #235d32; }
		.status-error { background: #fdecec; color: #8f2332; }
		.status-warn { background: #fff5e8; color: #85511c; }
		code { font-family: monospace; }
	</style>
</head>
<body>
	<main>
		<section class="panel">
			<h1>Example API Result</h1>
			{{if .Authenticated}}
				<p class="status status-ok">Authenticated through Consent and successfully reached the example route.</p>
			{{else if .AuthError}}
				<p class="status status-error">Authorization failed: <code>{{.AuthError}}</code></p>
			{{else}}
				<p class="status status-warn">{{.AuthHint}}</p>
			{{end}}
			<p><a href="/">Back to Home</a>{{if .Authenticated}} | <a href="/api/example?csrf={{.CSRF}}">Repeat Request</a>{{end}}</p>
		</section>
		<section class="panel">
			<h2>Details</h2>
			<dl><dt>Service</dt><dd>{{.Service}}</dd></dl>
			<dl><dt>Current Origin</dt><dd>{{.CurrentOrigin}}</dd></dl>
			<dl><dt>Current Host</dt><dd>{{.CurrentHost}}</dd></dl>
			{{if .Authenticated}}
				<dl><dt>Profile Handle</dt><dd>{{if .Handle}}{{.Handle}}{{else}}profile scope granted but no handle was returned{{end}}</dd></dl>
				<dl><dt>Opaque Subject</dt><dd>{{.Subject}}</dd></dl>
				<dl><dt>Granted Scopes</dt><dd>{{.Scopes}}</dd></dl>
				<dl><dt>Current CSRF</dt><dd>{{.CSRF}}</dd></dl>
			{{end}}
		</section>
	</main>
</body>
</html>`))
