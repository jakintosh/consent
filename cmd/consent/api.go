package main

import (
	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
	keys "git.sr.ht/~jakintosh/command-go/pkg/keys/cmd"
	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/config"
)

var apiCmd = &args.Command{
	Name:    "api",
	Help:    "API utilities",
	Options: wire.ClientArgs,
	Subcommands: []*args.Command{
		usersCmd,
		integrationsCmd,
		rolesCmd,
		keys.Command(buildKeysClient),
	},
}

func buildClient(
	i *args.Input,
) (
	wire.Client,
	error,
) {
	resolution, err := envs.Resolve(i, config.DefaultConfigDir())
	if err != nil {
		return wire.Client{}, err
	}

	client, err := resolution.Client()
	if err != nil {
		return wire.Client{}, err
	}

	return client.WithBasePath(config.APIUrlPrefix), nil
}

func buildKeysClient(
	i *args.Input,
) (
	wire.Client,
	error,
) {
	client, err := buildClient(i)
	if err != nil {
		return wire.Client{}, err
	}

	return client.WithBasePath("/admin/keys"), nil
}
