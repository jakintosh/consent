package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/consent/internal/app"
	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

var serveCmd = &args.Command{
	Name: "serve",
	Help: "Run the OAuth authorization server",
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
			log.Printf("  Port: %s", port)
			log.Printf("  Credentials: %s", credsDir)
		}

		signingKeyRaw := loadCredential("signing_key", credsDir)
		signingKey, err := x509.ParseECPrivateKey(signingKeyRaw)
		if err != nil {
			return fmt.Errorf("failed to parse ecdsa signing key from signing_key: %v", err)
		}

		dbOpts := database.SQLStoreOptions{
			Path: dbPath,
		}
		db, err := database.NewSQLStore(dbOpts)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}

		svcOpts := service.ServiceOptions{
			PasswordMode: service.PasswordModeProduction,
			Store:        db,
			TokenServerOpts: tokens.ServerOptions{
				SigningKey:   signingKey,
				IssuerDomain: issuerDomain,
			},
		}
		svc, err := service.New(svcOpts)
		if err != nil {
			return fmt.Errorf("failed to initialize service: %w", err)
		}

		appOpts := app.AppOptions{
			Service: svc,
		}
		appServer, err := app.New(appOpts)
		if err != nil {
			return fmt.Errorf("failed to initialize app server: %w", err)
		}

		mux := http.NewServeMux()
		mux.Handle("/", appServer.Router())
		mux.Handle("/api/v1/", http.StripPrefix("/api/v1", svc.Router()))

		if verbose {
			log.Printf("Listening on %s", port)
		}

		return http.ListenAndServe(port, mux)
	},
}
