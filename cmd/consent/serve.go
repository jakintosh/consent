package main

import (
	"log"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/consent/internal/config"
	"git.sr.ht/~jakintosh/consent/internal/server"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

var serveCmd = &args.Command{
	Name: "serve",
	Help: "run the OAuth authorization server",
	Options: append(
		runtimeOptions,
		args.Option{
			Long: "insecure-cookies",
			Type: args.OptionTypeFlag,
			Help: "emit Secure=false auth cookies",
		},
	),
	Handler: func(i *args.Input) error {
		cfgDir := i.GetParameterOr("config-dir", "")
		dataDir := i.GetParameterOr("data-dir", "")
		insecureCookies := i.GetFlag("insecure-cookies")
		verbose := i.GetFlag("verbose")

		overrides, err := resolveOverrides(i)
		if err != nil {
			return err
		}

		runtimeOpts := config.RuntimeOptions{
			Overrides:              overrides,
			RequireSigningKey:      true,
			RequireBootstrapAPIKey: false,
		}
		runtime, err := config.Resolve(cfgDir, dataDir, runtimeOpts)
		if err != nil {
			return err
		}

		if verbose {
			log.Printf("Starting consent server")
			log.Printf("  Config dir: %s", runtime.Paths.ConfigDir)
			log.Printf("  Data dir: %s", runtime.Paths.DataDir)
			log.Printf("  Database: %s", runtime.Paths.DatabaseFile)
			log.Printf("  Public URL: %s", runtime.Server.PublicURL)
			log.Printf("  Authority: %s", runtime.Server.AuthorityDomain)
			log.Printf("  Listen: %s", runtime.Server.ListenAddress)
			log.Printf("  Dev mode: %t", runtime.Server.DevMode)
			log.Printf("  Insecure cookies: %t", insecureCookies)
		}

		serverOpts := server.Options{
			Runtime:         runtime,
			InsecureCookies: insecureCookies,
			PasswordMode:    service.PasswordModeProduction,
		}
		return server.Serve(serverOpts)
	},
}
