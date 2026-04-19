package main

import (
	"fmt"
	"log"
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/app"
	"git.sr.ht/~jakintosh/consent/internal/config"
	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/pkg/client"
	"git.sr.ht/~jakintosh/consent/pkg/testing"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

var serveCmd = &args.Command{
	Name: "serve",
	Help: "Run the OAuth authorization server",
	Options: append(
		runtimeOptions,
		args.Option{
			Long: "insecure-cookies",
			Type: args.OptionTypeFlag,
			Help: "emit Secure=false auth cookies",
		},
	),
	Handler: func(i *args.Input) error {
		verbose := i.GetFlag("verbose")
		insecureCookies := i.GetFlag("insecure-cookies")
		cfgDir := i.GetParameterOr("config-dir", DEFAULT_CFG_DIR)
		dataDir := i.GetParameterOr("data-dir", DEFAULT_DATA_DIR)

		overrides, err := resolveOverrides(i)
		if err != nil {
			return err
		}

		opts := config.ResolveOptions{
			Overrides:              overrides,
			ConfigDir:              cfgDir,
			DataDir:                dataDir,
			RequireSigningKey:      true,
			RequireBootstrapAPIKey: false,
		}
		runtime, err := config.Resolve(opts)
		if err != nil {
			return err
		}

		if verbose {
			log.Printf("Starting consent server")
			log.Printf("  Config dir: %s", runtime.Paths.ConfigDir)
			log.Printf("  Data dir: %s", runtime.Paths.DataDir)
			log.Printf("  Database: %s", runtime.Paths.DatabaseFile)
			log.Printf("  Public URL: %s", runtime.Server.PublicURL)
			log.Printf("  Issuer: %s", runtime.Server.IssuerDomain)
			log.Printf("  Listen: %s", runtime.Server.ListenAddress)
			log.Printf("  Dev mode: %t", runtime.Server.DevMode)
			log.Printf("  Insecure cookies: %t", insecureCookies)
		}

		dbOpts := database.Options{
			Path: runtime.Paths.DatabaseFile,
		}
		db, err := database.Open(dbOpts)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		svcOpts := service.Options{
			PasswordMode: service.PasswordModeProduction,
			Store:        db,
			KeysStore:    db.KeysStore,
			TokenServerOpts: tokens.ServerOptions{
				SigningKey:   runtime.Secrets.SigningKey,
				IssuerDomain: runtime.Server.IssuerDomain,
			},
		}
		svc, err := service.New(svcOpts)
		if err != nil {
			return fmt.Errorf("failed to initialize service: %w", err)
		}

		var authConfig app.AuthConfig
		if runtime.Server.DevMode {
			tv := testing.NewTestVerifier(runtime.Server.IssuerDomain, runtime.Server.PublicHost)
			authConfig = app.AuthConfig{
				Verifier:  tv,
				LoginURL:  "/dev/login",
				LogoutURL: "/dev/logout",
				Routes: map[string]http.HandlerFunc{
					"/dev/login":  tv.HandleDevLogin(),
					"/dev/logout": tv.HandleDevLogout(),
				},
			}
		} else {
			opts := tokens.ClientOptions{
				VerificationKey: &runtime.Secrets.SigningKey.PublicKey,
				IssuerDomain:    runtime.Server.IssuerDomain,
				ValidAudience:   runtime.Server.PublicHost,
			}
			tkValidator := tokens.InitClient(opts)
			consentClient := client.Init(tkValidator, runtime.Server.PublicBaseURL)
			if insecureCookies {
				consentClient.EnableInsecureCookies()
			}

			authConfig = app.AuthConfig{
				Verifier:  consentClient,
				LoginURL:  "/login",
				LogoutURL: "/logout",
				Routes: map[string]http.HandlerFunc{
					"/auth/callback": consentClient.HandleAuthorizationCode(),
					"/logout":        consentClient.HandleLogout(),
				},
			}
		}

		appOpts := app.AppOptions{
			Service: svc,
			Auth:    authConfig,
		}
		appServer, err := app.New(appOpts)
		if err != nil {
			return fmt.Errorf("failed to initialize app server: %w", err)
		}

		mux := http.NewServeMux()
		mux.Handle("/", appServer.Router())
		wire.Subrouter(mux, "/api/v1", svc.Router())

		if verbose {
			log.Printf("Listening on %s", runtime.Server.ListenAddress)
		}

		return http.ListenAndServe(runtime.Server.ListenAddress, mux)
	},
}
