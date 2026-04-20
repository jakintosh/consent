package main

import (
	"fmt"
	"os"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/consent/internal/config"
	"gopkg.in/yaml.v3"
)

var configCmd = &args.Command{
	Name:    "config",
	Help:    "Manage consent configuration",
	Options: runtimeOptions,
	Subcommands: []*args.Command{
		configInitCmd,
		configShowCmd,
	},
}

var configInitCmd = &args.Command{
	Name: "init",
	Help: "Generate baseline config, secrets, and directories",
	Options: []args.Option{
		{
			Long: "force",
			Type: args.OptionTypeFlag,
			Help: "overwrite existing generated files",
		},
	},
	Handler: func(i *args.Input) error {

		cfgDir := i.GetParameterOr("config-dir", "")
		dataDir := i.GetParameterOr("data-dir", "")
		overrides, err := resolveOverrides(i)
		if err != nil {
			return err
		}

		opts := config.InitOptions{
			Overrides: overrides,
			Force:     i.GetFlag("force"),
		}
		result, err := config.Init(cfgDir, dataDir, opts)
		if err != nil {
			return err
		}

		fmt.Printf("config: %s\n", result.Paths.ConfigFile)
		fmt.Printf("data: %s\n", result.Paths.DataDir)
		fmt.Printf("signing key: %s\n", result.Paths.SigningKeyFile)
		fmt.Printf("verification key: %s\n", result.Paths.VerificationKeyFile)
		fmt.Printf("bootstrap api key: %s\n", result.Paths.BootstrapAPIKeyFile)

		return nil
	},
}

var configShowCmd = &args.Command{
	Name: "show",
	Help: "Show authored or resolved config",
	Options: []args.Option{
		{
			Long: "resolved",
			Type: args.OptionTypeFlag,
			Help: "show resolved runtime config",
		},
	},
	Handler: func(i *args.Input) error {

		resolved := i.GetFlag("resolved")
		cfgDir := i.GetParameterOr("config-dir", "")
		dataDir := i.GetParameterOr("data-dir", "")

		var cfgYaml any
		if resolved {
			overrides, err := resolveOverrides(i)
			if err != nil {
				return err
			}

			opts := config.RuntimeOptions{
				Overrides:              overrides,
				RequireSigningKey:      false,
				RequireBootstrapAPIKey: false,
			}
			runtime, err := config.Resolve(cfgDir, dataDir, opts)
			if err != nil {
				return err
			}

			cfgYaml = runtime.View()
		} else {
			cfg, err := config.Load(cfgDir, dataDir)
			if err != nil {
				return err
			}
			cfgYaml = cfg
		}

		data, err := yaml.Marshal(cfgYaml)
		if err != nil {
			return err
		}

		_, err = os.Stdout.Write(data)
		return err
	},
}
