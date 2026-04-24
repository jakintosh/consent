package main

import (
	"git.sr.ht/~jakintosh/command-go/pkg/args"
	keys "git.sr.ht/~jakintosh/command-go/pkg/keys/cmd"
	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/config"
)

var apiCmd = &args.Command{
	Name:    "api",
	Help:    "API utilities",
	Options: wire.ClientOptions,
	Subcommands: []*args.Command{
		registerCmd,
		integrationsCmd,
		rolesCmd,
		keys.Command(config.DefaultConfigDir(), "/api/v1/admin/keys"),
	},
}
