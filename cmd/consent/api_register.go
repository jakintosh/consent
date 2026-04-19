package main

import (
	"encoding/json"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
	"git.sr.ht/~jakintosh/consent/internal/service"
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
		client, err := envs.ResolveClient(i, DEFAULT_CFG_DIR, "/api/v1")
		if err != nil {
			return err
		}

		payload := service.RegistrationRequest{
			Handle:   i.GetOperand("handle"),
			Password: i.GetOperand("password"),
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Post("/register", body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}
