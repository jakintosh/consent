package main

import (
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/consent/internal/config"
	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

var initCmd = &args.Command{
	Name:    "init",
	Help:    "Initialize mutable runtime state from resolved config",
	Options: runtimeOptions,
	Handler: func(i *args.Input) error {
		cfgDir := i.GetParameterOr("config-dir", DEFAULT_CFG_DIR)
		dataDir := i.GetParameterOr("data-dir", DEFAULT_DATA_DIR)

		overrides, err := resolveOverrides(i)
		if err != nil {
			return err
		}

		resolveOpts := config.ResolveOptions{
			Overrides:              overrides,
			ConfigDir:              cfgDir,
			DataDir:                dataDir,
			RequireSigningKey:      false,
			RequireBootstrapAPIKey: true,
		}
		runtime, err := config.Resolve(resolveOpts)
		if err != nil {
			return err
		}

		dbOpts := database.Options{
			Path: runtime.Paths.DatabaseFile,
		}
		db, err := database.Open(dbOpts)
		if err != nil {
			return err
		}
		defer db.Close()

		initOpts := service.InitOptions{
			Store:          db,
			KeysStore:      db.KeysStore,
			PublicURL:      runtime.Server.PublicURL,
			BootstrapToken: runtime.Secrets.BootstrapAPIKey,
		}
		if err := service.Init(initOpts); err != nil {
			return err
		}

		fmt.Printf("database: %s\n", runtime.Paths.DatabaseFile)
		fmt.Printf("bootstrap api key: %s\n", runtime.Paths.BootstrapAPIKeyFile)
		fmt.Printf("system service: %s\n", service.InternalServiceName)
		return nil
	},
}
