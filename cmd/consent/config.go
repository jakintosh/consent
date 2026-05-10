package main

import (
	"fmt"
	"os"
	"time"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/consent/internal/config"
	"gopkg.in/yaml.v3"
)

var configCmd = &args.Command{
	Name:    "config",
	Help:    "manage consent configuration",
	Options: runtimeOptions,
	Subcommands: []*args.Command{
		configInitCmd,
		configShowCmd,
	},
}

var configInitCmd = &args.Command{
	Name: "init",
	Help: "generate baseline config, secrets, and directories",
	Options: []args.Option{
		{
			Long: "force",
			Type: args.OptionTypeFlag,
			Help: "overwrite existing generated files",
		},
		{
			Long: "auth-code-ttl",
			Type: args.OptionTypeParameter,
			Help: "authorization code token lifetime",
		},
		{
			Long: "access-ttl",
			Type: args.OptionTypeParameter,
			Help: "access token lifetime",
		},
		{
			Long: "refresh-ttl",
			Type: args.OptionTypeParameter,
			Help: "refresh token lifetime",
		},
	},
	Handler: func(i *args.Input) error {

		cfgDir := i.GetParameterOr("config-dir", "")
		dataDir := i.GetParameterOr("data-dir", "")
		overrides, err := resolveOverrides(i)
		if err != nil {
			return err
		}
		tokenOverrides, err := resolveTokenTTLOverrides(i)
		if err != nil {
			return err
		}
		overrides.AuthCodeTTL = tokenOverrides.AuthCodeTTL
		overrides.AccessTTL = tokenOverrides.AccessTTL
		overrides.RefreshTTL = tokenOverrides.RefreshTTL

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

func resolveTokenTTLOverrides(
	i *args.Input,
) (
	config.Overrides,
	error,
) {
	var overrides config.Overrides

	if value := i.GetParameter("auth-code-ttl"); value != nil {
		ttl, err := parseDurationOption("auth-code-ttl", *value)
		if err != nil {
			return config.Overrides{}, err
		}
		overrides.AuthCodeTTL = &ttl
	}
	if value := i.GetParameter("access-ttl"); value != nil {
		ttl, err := parseDurationOption("access-ttl", *value)
		if err != nil {
			return config.Overrides{}, err
		}
		overrides.AccessTTL = &ttl
	}
	if value := i.GetParameter("refresh-ttl"); value != nil {
		ttl, err := parseDurationOption("refresh-ttl", *value)
		if err != nil {
			return config.Overrides{}, err
		}
		overrides.RefreshTTL = &ttl
	}

	return overrides, nil
}

func parseDurationOption(
	name string,
	value string,
) (
	time.Duration,
	error,
) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --%s %q: expected duration like 10s, 30m, or 72h", name, value)
	}
	return duration, nil
}

var configShowCmd = &args.Command{
	Name: "show",
	Help: "show authored or resolved config",
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
