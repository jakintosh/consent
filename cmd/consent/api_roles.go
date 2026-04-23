package main

import (
	"encoding/json"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/config"
)

var rolesCmd = &args.Command{
	Name: "roles",
	Help: "Manage roles",
	Subcommands: []*args.Command{
		rolesListCmd,
		rolesGetCmd,
		rolesCreateCmd,
		rolesUpdateCmd,
		rolesDeleteCmd,
	},
}

var rolesListCmd = &args.Command{
	Name: "list",
	Help: "List roles",
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		var roles []api.Role
		if err := client.Get("/admin/roles", &roles); err != nil {
			return err
		}

		return printJSON(roles)
	},
}

var rolesGetCmd = &args.Command{
	Name: "get",
	Help: "Get a role",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Role name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("role name is required")
		}

		var role api.Role
		if err := client.Get("/admin/roles/"+name, &role); err != nil {
			return err
		}

		return printJSON(role)
	},
}

var rolesCreateCmd = &args.Command{
	Name: "create",
	Help: "Create a role",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Role name",
		},
	},
	Options: []args.Option{
		{
			Long: "display",
			Type: args.OptionTypeParameter,
			Help: "Role display name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("role name is required")
		}

		display := i.GetParameter("display")
		if display == nil {
			return fmt.Errorf("--display is required")
		}

		payload := api.Role{
			Name:    name,
			Display: *display,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Post("/admin/roles", body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var rolesUpdateCmd = &args.Command{
	Name: "update",
	Help: "Update a role",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Role name",
		},
	},
	Options: []args.Option{
		{
			Long: "display",
			Type: args.OptionTypeParameter,
			Help: "New display name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("role name is required")
		}

		display := i.GetParameter("display")
		if display == nil {
			return fmt.Errorf("--display is required")
		}

		payload := api.UpdateRoleRequest{
			Display: display,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Put("/admin/roles/"+name, body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var rolesDeleteCmd = &args.Command{
	Name: "delete",
	Help: "Delete a role",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Role name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("role name is required")
		}

		if err := client.Delete("/admin/roles/"+name, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}
