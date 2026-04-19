package main

import (
	"fmt"
	"os"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/consent/internal/config"
	"gopkg.in/yaml.v3"
)

var configCmd = &args.Command{
	Name: "config",
	Help: "Manage consent configuration",
	Subcommands: []*args.Command{
		configInitCmd,
		configShowCmd,
	},
}

var configInitCmd = &args.Command{
	Name: "init",
	Help: "Generate baseline config, secrets, and directories",
	Options: append([]args.Option{
		{
			Long: "force",
			Type: args.OptionTypeFlag,
			Help: "overwrite existing generated files",
		},
	}, runtimeOptions...),
	Handler: func(i *args.Input) error {
		overrides, err := resolveOverrides(i)
		if err != nil {
			return err
		}

		opts := config.InitOptions{
			ConfigDir: i.GetParameterOr("config-dir", DEFAULT_CFG_DIR),
			DataDir:   i.GetParameterOr("data-dir", DEFAULT_DATA_DIR),
			Force:     i.GetFlag("force"),
			Overrides: overrides,
		}
		result, err := config.Init(opts)
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
	Options: append([]args.Option{
		{
			Long: "resolved",
			Type: args.OptionTypeFlag,
			Help: "show resolved runtime config",
		},
	}, runtimeOptions...),
	Handler: func(i *args.Input) error {

		resolved := i.GetFlag("resolved")
		cfgDir := i.GetParameterOr("config-dir", DEFAULT_CFG_DIR)
		dataDir := i.GetParameterOr("data-dir", DEFAULT_DATA_DIR)

		var cfgYaml any
		if resolved {
			overrides, err := resolveOverrides(i)
			if err != nil {
				return err
			}

			opts := config.ResolveOptions{
				Overrides:              overrides,
				ConfigDir:              cfgDir,
				DataDir:                dataDir,
				RequireSigningKey:      false,
				RequireBootstrapAPIKey: false,
			}
			runtime, err := config.Resolve(opts)
			if err != nil {
				return err
			}

			cfgYaml = runtime.View()
		} else {
			roots, err := config.ResolveRoots(cfgDir, dataDir)
			if err != nil {
				return err
			}

			paths := config.BuildPaths(roots)
			cfg, err := config.Load(paths)
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
