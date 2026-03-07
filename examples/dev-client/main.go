package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/consent/pkg/client"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

const (
	defaultAuthURL             = "http://localhost:9001"
	defaultIssuerDomain        = "localhost"
	defaultPort                = "10000"
	defaultServiceName         = "example@localhost"
	defaultVerificationKeyPath = "./secrets/verification_key.der"
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
			Long: "verification-key",
			Type: args.OptionTypeParameter,
			Help: "Path to verification key DER file (default: ./secrets/verification_key.der)",
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

		validator := tokens.InitClient(verificationKey, cfg.IssuerDomain, cfg.Audience)
		authClient := client.Init(validator, cfg.AuthURL)
		authClient.EnableDevelopmentMode()

		mux := http.NewServeMux()
		mux.HandleFunc("/", homeHandler(authClient, cfg))
		mux.HandleFunc("/api/example", exampleHandler(authClient))
		mux.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())

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
	loginURL := fmt.Sprintf("%s/login?service=%s", cfg.AuthURL, url.QueryEscape(cfg.Service))

	return func(w http.ResponseWriter, r *http.Request) {
		accessToken, csrf, err := c.VerifyAuthorizationGetCSRF(w, r)
		if err != nil {
			if !errors.Is(err, client.ErrTokenAbsent) {
				log.Printf("%s: failed to verify authorization: %v", r.RequestURI, err)
			}
		}

		if accessToken != nil {
			w.Write(fmt.Appendf(nil, homeAuth, csrf))
		} else {
			w.Write(fmt.Appendf(nil, homeUnauth, loginURL))
		}
	}
}

func exampleHandler(c client.Verifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		csrf := r.URL.Query().Get("csrf")
		accessToken, csrf, err := c.VerifyAuthorizationCheckCSRF(w, r, csrf)
		if err != nil {
			log.Printf("%s: failed to verify authorization: %v", r.RequestURI, err)
		}

		if accessToken != nil {
			w.Write(fmt.Appendf(nil, exampleAuth, accessToken.Subject(), csrf))
		} else {
			w.Write([]byte(exampleUnauth))
		}
	}
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

const homeAuth string = `<!DOCTYPE html>
<html>
<body>
<a href="/api/example?csrf=%s">Example API Call</a>
</body>
</html>`

const homeUnauth string = `<!DOCTYPE html>
<html>
<body>
<a href="%s">Log In with Pollinator</a>
</body>
</html>`

const exampleAuth string = `<!DOCTYPE html>
<html>
<body>
<p>Secret logged in page for %s!</p>
<form>
	<input hidden value="%s"/>
</form>
</body>
</html>`

const exampleUnauth string = `<!DOCTYPE html>
<html>
<body>
<p>You are not logged in.</p>
</body>
</html>`
