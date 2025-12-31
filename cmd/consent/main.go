package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/version"
	"git.sr.ht/~jakintosh/consent/internal/app"
	"git.sr.ht/~jakintosh/consent/pkg/api"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
	"github.com/gorilla/mux"
)

var root = &args.Command{
	Name: "consent",
	Help: "OAuth authorization server",
	Config: &args.Config{
		Author: "jakintosh",
		HelpOption: &args.HelpOption{
			Short: 'h',
			Long:  "help",
		},
	},
	Options: []args.Option{
		{
			Long: "db-path",
			Type: args.OptionTypeParameter,
			Help: "SQLite database path (env: DB_PATH)",
		},
		{
			Long: "issuer-domain",
			Type: args.OptionTypeParameter,
			Help: "JWT issuer domain (env: ISSUER_DOMAIN)",
		},
		{
			Long: "templates-path",
			Type: args.OptionTypeParameter,
			Help: "HTML templates directory (env: TEMPLATES_PATH)",
		},
		{
			Long: "services-path",
			Type: args.OptionTypeParameter,
			Help: "Services configuration directory (env: SERVICES_PATH)",
		},
		{
			Long: "port",
			Type: args.OptionTypeParameter,
			Help: "HTTP listen port (env: PORT)",
		},
		{
			Long: "credentials-dir",
			Type: args.OptionTypeParameter,
			Help: "Directory containing signing_key (env: CREDENTIALS_DIRECTORY)",
		},
		{
			Short: 'v',
			Long:  "verbose",
			Type:  args.OptionTypeFlag,
			Help:  "Verbose output",
		},
	},
	Subcommands: []*args.Command{
		version.Command(VersionInfo),
	},
	Handler: func(i *args.Input) error {
		verbose := i.GetFlag("verbose")

		dbPath := resolveOption(i, "db-path", "DB_PATH", "")
		if dbPath == "" {
			return fmt.Errorf("--db-path or DB_PATH is required")
		}

		issuerDomain := resolveOption(i, "issuer-domain", "ISSUER_DOMAIN", "")
		if issuerDomain == "" {
			return fmt.Errorf("--issuer-domain or ISSUER_DOMAIN is required")
		}

		templatesPath := resolveOption(i, "templates-path", "TEMPLATES_PATH", "")
		if templatesPath == "" {
			return fmt.Errorf("--templates-path or TEMPLATES_PATH is required")
		}

		servicesPath := resolveOption(i, "services-path", "SERVICES_PATH", "")
		if servicesPath == "" {
			return fmt.Errorf("--services-path or SERVICES_PATH is required")
		}

		portStr := resolveOption(i, "port", "PORT", "")
		if portStr == "" {
			return fmt.Errorf("--port or PORT is required")
		}
		port := fmt.Sprintf(":%s", portStr)

		credsDir := resolveOption(i, "credentials-dir", "CREDENTIALS_DIRECTORY", "")
		if credsDir == "" {
			return fmt.Errorf("--credentials-dir or CREDENTIALS_DIRECTORY is required")
		}

		if verbose {
			log.Printf("Starting consent server...")
			log.Printf("  Database: %s", dbPath)
			log.Printf("  Issuer: %s", issuerDomain)
			log.Printf("  Templates: %s", templatesPath)
			log.Printf("  Services: %s", servicesPath)
			log.Printf("  Port: %s", port)
			log.Printf("  Credentials: %s", credsDir)
		}

		// Load credentials
		signingKeyRaw := loadCredential("signing_key", credsDir)
		signingKey, err := x509.ParseECPrivateKey(signingKeyRaw)
		if err != nil {
			return fmt.Errorf("failed to parse ecdsa signing key from signing_key: %v", err)
		}

		// Init program services
		services := api.NewServices(servicesPath)
		templates := app.NewTemplates(templatesPath)
		issuer, validator := tokens.InitServer(signingKey, issuerDomain)

		// Init endpoints
		app.Init(services, templates)
		authApi := api.New(issuer, validator, services, dbPath)

		// Config and serve router
		r := mux.NewRouter()
		r.HandleFunc("/", app.Home)
		r.HandleFunc("/login", app.Login)

		// API subrouter
		apiRouter := r.PathPrefix("/api").Subrouter()
		authApi.BuildRouter(apiRouter)

		if verbose {
			log.Printf("Listening on %s", port)
		}

		err = http.ListenAndServe(port, r)
		if err != nil {
			return fmt.Errorf("server error: %v", err)
		}

		return nil
	},
}

func main() {
	root.Parse()
}

func resolveOption(
	i *args.Input,
	optionName string,
	envVarName string,
	defaultValue string,
) string {
	// Check CLI option first
	if param := i.GetParameter(optionName); param != nil {
		return *param
	}

	// Check environment variable
	if envVal := os.Getenv(envVarName); envVal != "" {
		return envVal
	}

	// Return default
	return defaultValue
}

func loadCredential(
	name string,
	credsDir string,
) []byte {
	credPath := filepath.Join(credsDir, name)
	cred, err := os.ReadFile(credPath)
	if err != nil {
		log.Fatalf("failed to load required credential '%s': %v\n", name, err)
	}
	return cred
}
