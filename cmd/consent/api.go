package main

import (
	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/wire"
)

var apiCmd = &args.Command{
	Name:    "api",
	Help:    "API utilities",
	Options: wire.ClientOptions,
	Subcommands: []*args.Command{
		servicesCmd,
	},
}
