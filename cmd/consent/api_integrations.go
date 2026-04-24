package main

import (
	"encoding/json"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/config"
)

var integrationsCmd = &args.Command{
	Name: "integrations",
	Help: "Manage integrations",
	Subcommands: []*args.Command{
		integrationsListCmd,
		integrationsGetCmd,
		integrationsCreateCmd,
		integrationsUpdateCmd,
		integrationsDeleteCmd,
	},
}

var integrationsListCmd = &args.Command{
	Name: "list",
	Help: "List integrations",
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		var integrations []api.Integration
		if err := client.Get("/admin/integrations", &integrations); err != nil {
			return err
		}

		return printJSON(integrations)
	},
}

var integrationsGetCmd = &args.Command{
	Name: "get",
	Help: "Get an integration",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Integration name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("integration name is required")
		}

		var integration api.Integration
		if err := client.Get("/admin/integrations/"+name, &integration); err != nil {
			return err
		}

		return printJSON(integration)
	},
}

var integrationsCreateCmd = &args.Command{
	Name: "create",
	Help: "Create an integration",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Integration name",
		},
	},
	Options: []args.Option{
		{
			Long: "display",
			Type: args.OptionTypeParameter,
			Help: "Integration display name",
		},
		{
			Long: "audience",
			Type: args.OptionTypeParameter,
			Help: "Integration audience",
		},
		{
			Long: "redirect",
			Type: args.OptionTypeParameter,
			Help: "Redirect URL",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("integration name is required")
		}

		display := i.GetParameter("display")
		audience := i.GetParameter("audience")
		redirect := i.GetParameter("redirect")
		if display == nil || audience == nil || redirect == nil {
			return fmt.Errorf("--display, --audience, and --redirect are required")
		}

		payload := api.Integration{
			Name:     name,
			Display:  *display,
			Audience: *audience,
			Redirect: *redirect,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Post("/admin/integrations", body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var integrationsUpdateCmd = &args.Command{
	Name: "update",
	Help: "Update an integration",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Integration name",
		},
	},
	Options: []args.Option{
		{
			Long: "display",
			Type: args.OptionTypeParameter,
			Help: "Integration display name",
		},
		{
			Long: "audience",
			Type: args.OptionTypeParameter,
			Help: "Integration audience",
		},
		{
			Long: "redirect",
			Type: args.OptionTypeParameter,
			Help: "Redirect URL",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("integration name is required")
		}

		display := i.GetParameter("display")
		audience := i.GetParameter("audience")
		redirect := i.GetParameter("redirect")
		if display == nil && audience == nil && redirect == nil {
			return fmt.Errorf("at least one of --display, --audience, or --redirect is required")
		}

		payload := api.UpdateIntegrationRequest{
			Display:  display,
			Audience: audience,
			Redirect: redirect,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Patch("/admin/integrations/"+name, body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var integrationsDeleteCmd = &args.Command{
	Name: "delete",
	Help: "Delete an integration",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Integration name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, config.DefaultConfigDir(), "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("integration name is required")
		}

		if err := client.Delete("/admin/integrations/"+name, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

func printJSON(value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(payload))
	return nil
}
