package main

import (
	"git.sr.ht/~jakintosh/command-go/pkg/args"
	keys "git.sr.ht/~jakintosh/command-go/pkg/keys/cmd"
	"git.sr.ht/~jakintosh/command-go/pkg/wire"
)

var apiCmd = &args.Command{
	Name:    "api",
	Help:    "API utilities",
	Options: wire.ClientOptions,
	Subcommands: []*args.Command{
		servicesCmd,
		keys.Command(DEFAULT_CFG_DIR, "/api/v1/admin"),
	},
}
