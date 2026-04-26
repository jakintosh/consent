package main

import (
	"encoding/json"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/config"
)

var usersCmd = &args.Command{
	Name: "users",
	Help: "manage users",
	Subcommands: []*args.Command{
		usersListCmd,
		usersGetCmd,
		usersCreateCmd,
		usersUpdateCmd,
		usersDeleteCmd,
	},
}

var usersListCmd = &args.Command{
	Name: "list",
	Help: "list users",
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), config.APIUrlPrefix)
		if err != nil {
			return err
		}

		var users []api.User
		if err := client.Get("/admin/users", &users); err != nil {
			return err
		}

		return printJSON(users)
	},
}

var usersGetCmd = &args.Command{
	Name: "get",
	Help: "get a user",
	Operands: []args.Operand{
		{
			Name: "subject",
			Help: "user subject",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), config.APIUrlPrefix)
		if err != nil {
			return err
		}

		subject := i.GetOperand("subject")
		if subject == "" {
			return fmt.Errorf("user subject is required")
		}

		var user api.User
		if err := client.Get("/admin/users/"+subject, &user); err != nil {
			return err
		}

		return printJSON(user)
	},
}

var usersCreateCmd = &args.Command{
	Name: "create",
	Help: "create a user",
	Operands: []args.Operand{
		{
			Name: "handle",
			Help: "user handle",
		},
	},
	Options: []args.Option{
		{
			Long: "password",
			Type: args.OptionTypeParameter,
			Help: "user password",
		},
		{
			Long: "role",
			Type: args.OptionTypeArray,
			Help: "user role",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), config.APIUrlPrefix)
		if err != nil {
			return err
		}

		handle := i.GetOperand("handle")
		if handle == "" {
			return fmt.Errorf("user handle is required")
		}

		password := i.GetParameter("password")
		if password == nil {
			return fmt.Errorf("--password is required")
		}

		roles := i.GetArray("role")
		if len(roles) == 0 {
			return fmt.Errorf("--role is required")
		}

		payload := api.CreateUserRequest{
			Handle:   handle,
			Password: *password,
			Roles:    roles,
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

var usersUpdateCmd = &args.Command{
	Name: "update",
	Help: "update a user",
	Operands: []args.Operand{
		{
			Name: "subject",
			Help: "user subject",
		},
	},
	Options: []args.Option{
		{
			Long: "handle",
			Type: args.OptionTypeParameter,
			Help: "new user handle",
		},
		{
			Long: "role",
			Type: args.OptionTypeArray,
			Help: "user role",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), config.APIUrlPrefix)
		if err != nil {
			return err
		}

		subject := i.GetOperand("subject")
		if subject == "" {
			return fmt.Errorf("user subject is required")
		}

		handle := i.GetParameter("handle")
		roles := i.GetArray("role")
		if handle == nil && len(roles) == 0 {
			return fmt.Errorf("at least one of --handle or --role is required")
		}

		payload := api.UpdateUserRequest{
			Handle: handle,
		}
		if len(roles) > 0 {
			payload.Roles = &roles
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Patch("/admin/users/"+subject, body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var usersDeleteCmd = &args.Command{
	Name: "delete",
	Help: "delete a user",
	Operands: []args.Operand{
		{
			Name: "subject",
			Help: "user subject",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), config.APIUrlPrefix)
		if err != nil {
			return err
		}

		subject := i.GetOperand("subject")
		if subject == "" {
			return fmt.Errorf("user subject is required")
		}

		if err := client.Delete("/admin/users/"+subject, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}
