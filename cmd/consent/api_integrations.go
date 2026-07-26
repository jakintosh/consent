package main

import (
	"encoding/json"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/consent/internal/api"
)

var integrationsCmd = &args.Command{
	Name: "integrations",
	Help: "manage integrations",
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
	Help: "list integrations",
	Handler: func(i *args.Input) error {
		client, err := buildClient(i)
		if err != nil {
			return err
		}

		var integrations []api.Integration
		if err := client.Get(i.Context(), "/admin/integrations", &integrations); err != nil {
			return err
		}

		return printJSON(integrations)
	},
}

var integrationsGetCmd = &args.Command{
	Name: "get",
	Help: "get an integration",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "integration name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := buildClient(i)
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("integration name is required")
		}

		var integration api.Integration
		if err := client.Get(i.Context(), "/admin/integrations/"+name, &integration); err != nil {
			return err
		}

		return printJSON(integration)
	},
}

var integrationsCreateCmd = &args.Command{
	Name: "create",
	Help: "create an integration",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "integration name",
		},
	},
	Options: []args.Option{
		{
			Long: "display",
			Type: args.OptionTypeParameter,
			Help: "integration display name",
		},
		{
			Long: "audience",
			Type: args.OptionTypeParameter,
			Help: "integration audience",
		},
		{
			Long: "redirect",
			Type: args.OptionTypeParameter,
			Help: "redirect URL",
		},
		{
			Long: "homepage",
			Type: args.OptionTypeParameter,
			Help: "integration homepage URL",
		},
		{
			Long: "logo",
			Type: args.OptionTypeParameter,
			Help: "integration logo URL",
		},
		{
			Long: "required-roles",
			Type: args.OptionTypeArray,
			Help: "required role to access this integration (can be specified multiple times)",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := buildClient(i)
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
		homepage := i.GetParameter("homepage")
		logo := i.GetParameter("logo")
		requiredRoles := i.GetArray("required-roles")
		if display == nil || audience == nil || redirect == nil || homepage == nil || logo == nil {
			return fmt.Errorf("--display, --audience, --redirect, --homepage, and --logo are required")
		}

		payload := api.Integration{
			Name:          name,
			Display:       *display,
			Audience:      *audience,
			Redirect:      *redirect,
			Homepage:      *homepage,
			Logo:          *logo,
			RequiredRoles: requiredRoles,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Post(i.Context(), "/admin/integrations", body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var integrationsUpdateCmd = &args.Command{
	Name: "update",
	Help: "update an integration",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "integration name",
		},
	},
	Options: []args.Option{
		{
			Long: "display",
			Type: args.OptionTypeParameter,
			Help: "integration display name",
		},
		{
			Long: "audience",
			Type: args.OptionTypeParameter,
			Help: "integration audience",
		},
		{
			Long: "redirect",
			Type: args.OptionTypeParameter,
			Help: "redirect URL",
		},
		{
			Long: "homepage",
			Type: args.OptionTypeParameter,
			Help: "integration homepage URL",
		},
		{
			Long: "logo",
			Type: args.OptionTypeParameter,
			Help: "integration logo URL",
		},
		{
			Long: "required-roles",
			Type: args.OptionTypeArray,
			Help: "required role to access this integration (can be specified multiple times)",
		},
		{
			Long: "clear-required-roles",
			Type: args.OptionTypeFlag,
			Help: "clear all required roles for this integration",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := buildClient(i)
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
		homepage := i.GetParameter("homepage")
		logo := i.GetParameter("logo")
		requiredRoles := i.GetArray("required-roles")
		clearRequiredRoles := i.GetFlag("clear-required-roles")
		hasRequiredRoles := len(requiredRoles) > 0
		if hasRequiredRoles && clearRequiredRoles {
			return fmt.Errorf("--required-roles and --clear-required-roles cannot be used together")
		}
		if display == nil && audience == nil && redirect == nil && homepage == nil && logo == nil && !hasRequiredRoles && !clearRequiredRoles {
			return fmt.Errorf("at least one of --display, --audience, --redirect, --homepage, --logo, --required-roles, or --clear-required-roles is required")
		}

		payload := api.UpdateIntegrationRequest{
			Display:  display,
			Audience: audience,
			Redirect: redirect,
			Homepage: homepage,
			Logo:     logo,
		}
		if hasRequiredRoles {
			payload.RequiredRoles = &requiredRoles
		} else if clearRequiredRoles {
			emptyRoles := []string{}
			payload.RequiredRoles = &emptyRoles
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Patch(i.Context(), "/admin/integrations/"+name, body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var integrationsDeleteCmd = &args.Command{
	Name: "delete",
	Help: "delete an integration",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "integration name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := buildClient(i)
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("integration name is required")
		}

		if err := client.Delete(i.Context(), "/admin/integrations/"+name, nil); err != nil {
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
