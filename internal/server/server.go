package server

import (
	"fmt"
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/app"
	"git.sr.ht/~jakintosh/consent/internal/config"
	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/pkg/client"
	"git.sr.ht/~jakintosh/consent/pkg/testing"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

type Options struct {
	Runtime         config.Runtime
	InsecureCookies bool
	PasswordMode    service.PasswordMode
}

func Serve(
	options Options,
) error {
	if options.Runtime.Secrets.SigningKey == nil {
		return fmt.Errorf("failed to initialize service: signing key required")
	}

	// build database
	dbOpts := database.Options{
		Path: options.Runtime.Paths.DatabaseFile,
	}
	db, err := database.Open(dbOpts)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// build service
	svcOpts := service.Options{
		PasswordMode: options.PasswordMode,
		Store:        db,
		TokenServerOpts: tokens.ServerOptions{
			SigningKey:   options.Runtime.Secrets.SigningKey,
			IssuerDomain: options.Runtime.Server.AuthorityDomain,
		},
		ResourceTokenClientOpts: tokens.ClientOptions{
			VerificationKey: &options.Runtime.Secrets.SigningKey.PublicKey,
			IssuerDomain:    options.Runtime.Server.AuthorityDomain,
			ValidAudience:   options.Runtime.Server.AuthorityDomain,
		},
	}
	svc, err := service.New(svcOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// build api
	apiOpts := api.Options{
		Service:   svc,
		KeysStore: db.KeysStore,
	}
	apiServer, err := api.New(apiOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize api server: %w", err)
	}

	// build app
	var authConfig app.AuthConfig
	if options.Runtime.Server.DevMode {
		authConfig = buildDevAuthConfig(options)
	} else {
		authConfig = buildProdAuthConfig(options)
	}
	appOpts := app.Options{
		Service: svc,
		Auth:    authConfig,
	}
	appServer, err := app.New(appOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize app server: %w", err)
	}

	// build router
	mux := http.NewServeMux()
	wire.Subrouter(mux, "/", appServer.Router())
	wire.Subrouter(mux, "/api/v1", apiServer.Router())

	//serve
	return http.ListenAndServe(options.Runtime.Server.ListenAddress, mux)
}

func buildProdAuthConfig(
	options Options,
) app.AuthConfig {
	prodClientOpts := tokens.ClientOptions{
		VerificationKey: &options.Runtime.Secrets.SigningKey.PublicKey,
		IssuerDomain:    options.Runtime.Server.AuthorityDomain,
		ValidAudience:   options.Runtime.Server.PublicHost,
	}
	tkValidator := tokens.InitClient(prodClientOpts)
	prodClient := client.Init(tkValidator, options.Runtime.Server.PublicBaseURL)
	if options.InsecureCookies {
		prodClient.EnableInsecureCookies()
	}
	return app.AuthConfig{
		Verifier:  prodClient,
		LoginURL:  "/login",
		LogoutURL: "/logout",
		Routes: map[string]http.HandlerFunc{
			"/auth/callback": prodClient.HandleAuthorizationCode(),
			"/logout":        prodClient.HandleLogout(),
		},
	}
}

func buildDevAuthConfig(
	options Options,
) app.AuthConfig {
	devClient := testing.NewTestVerifier(
		options.Runtime.Server.AuthorityDomain,
		options.Runtime.Server.PublicHost,
	)
	return app.AuthConfig{
		Verifier:  devClient,
		LoginURL:  "/dev/login",
		LogoutURL: "/dev/logout",
		Routes: map[string]http.HandlerFunc{
			"/dev/login":  devClient.HandleDevLogin(),
			"/dev/logout": devClient.HandleDevLogout(),
		},
	}
}
