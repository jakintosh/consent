package main

import (
	"encoding/json"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/config"
)

var registerCmd = &args.Command{
	Name: "register",
	Help: "Register a local user through the API",
	Operands: []args.Operand{
		{
			Name: "handle",
			Help: "User handle",
		},
		{
			Name: "password",
			Help: "User password",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		payload := api.CreateUserRequest{
			Handle:   i.GetOperand("handle"),
			Password: i.GetOperand("password"),
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Post("/admin/users", body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}
