package main

import (
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
	"git.sr.ht/~jakintosh/command-go/pkg/keys"
	"git.sr.ht/~jakintosh/command-go/pkg/version"
	"git.sr.ht/~jakintosh/consent/internal/config"
)

const (
	BIN_AUTHOR       = "jakintosh"
	DEFAULT_CFG_DIR  = "~/.config/consent"
	DEFAULT_DATA_DIR = "~/.local/share/consent"
)

var envsOpts = envs.CommandOptions{
	DefaultConfigDir: DEFAULT_CFG_DIR,
	KeyBackend: keys.EnvBackend{
		CollectionPath: "/api/v1/admin/keys",
	},
}

var runtimeOptions = []args.Option{
	{
		Long: "public-url",
		Type: args.OptionTypeParameter,
		Help: "server public URL",
	},
	{
		Long: "issuer-domain",
		Type: args.OptionTypeParameter,
		Help: "JWT issuer domain",
	},
	{
		Long: "port",
		Type: args.OptionTypeParameter,
		Help: "HTTP listen port",
	},
	{
		Long: "dev-mode",
		Type: args.OptionTypeFlag,
		Help: "dev mode",
	},
	{
		Short: 'v',
		Long:  "verbose",
		Type:  args.OptionTypeFlag,
		Help:  "verbose output",
	},
}

var root = &args.Command{
	Name: "consent",
	Help: "OAuth authorization server",
	Config: &args.Config{
		Author: BIN_AUTHOR,
		HelpOption: &args.HelpOption{
			Short: 'h',
			Long:  "help",
		},
	},
	Options: envs.ConfigOptionsAnd(
		args.Option{
			Long: "data-dir",
			Type: args.OptionTypeParameter,
			Help: "path to data directory",
		},
	),
	Subcommands: []*args.Command{
		apiCmd,
		configCmd,
		initCmd,
		serveCmd,
		envs.Command(envsOpts),
		version.Command(VersionInfo),
	},
}

func main() {
	root.Parse()
}

func resolveOverrides(
	i *args.Input,
) (
	config.Overrides,
	error,
) {
	var overrides config.Overrides

	if value := i.GetParameter("public-url"); value != nil {
		trimmed := strings.TrimSpace(*value)
		overrides.PublicURL = &trimmed
	}

	if value := i.GetParameter("issuer-domain"); value != nil {
		trimmed := strings.TrimSpace(*value)
		overrides.IssuerDomain = &trimmed
	}

	if value := i.GetParameter("port"); value != nil {
		port, err := strconv.Atoi(strings.TrimSpace(*value))
		if err != nil {
			return config.Overrides{}, fmt.Errorf("invalid --port %q: expected integer", *value)
		}
		overrides.Port = &port
	}

	if i.GetFlag("dev-mode") {
		devMode := true
		overrides.DevMode = &devMode
	}

	return overrides, nil
}
