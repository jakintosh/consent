package main

import (
	"encoding/json"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
)

type serviceDefinition struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

type updateServiceRequest struct {
	Display  *string `json:"display,omitempty"`
	Audience *string `json:"audience,omitempty"`
	Redirect *string `json:"redirect,omitempty"`
}

var servicesCmd = &args.Command{
	Name: "services",
	Help: "Manage services",
	Subcommands: []*args.Command{
		servicesListCmd,
		servicesGetCmd,
		servicesCreateCmd,
		servicesUpdateCmd,
		servicesDeleteCmd,
	},
}

var servicesListCmd = &args.Command{
	Name: "list",
	Help: "List services",
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, DEFAULT_CFG_DIR, "/api/v1")
		if err != nil {
			return err
		}

		var services []serviceDefinition
		if err := client.Get("/services", &services); err != nil {
			return err
		}

		return printJSON(services)
	},
}

var servicesGetCmd = &args.Command{
	Name: "get",
	Help: "Get a service",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Service name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, DEFAULT_CFG_DIR, "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("service name is required")
		}

		var service serviceDefinition
		if err := client.Get("/services/"+name, &service); err != nil {
			return err
		}

		return printJSON(service)
	},
}

var servicesCreateCmd = &args.Command{
	Name: "create",
	Help: "Create a service",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Service name",
		},
	},
	Options: []args.Option{
		{
			Long: "display",
			Type: args.OptionTypeParameter,
			Help: "Service display name",
		},
		{
			Long: "audience",
			Type: args.OptionTypeParameter,
			Help: "Service audience",
		},
		{
			Long: "redirect",
			Type: args.OptionTypeParameter,
			Help: "Redirect URL",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, DEFAULT_CFG_DIR, "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("service name is required")
		}

		display := i.GetParameter("display")
		audience := i.GetParameter("audience")
		redirect := i.GetParameter("redirect")
		if display == nil || audience == nil || redirect == nil {
			return fmt.Errorf("--display, --audience, and --redirect are required")
		}

		payload := serviceDefinition{
			Name:     name,
			Display:  *display,
			Audience: *audience,
			Redirect: *redirect,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Post("/services", body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var servicesUpdateCmd = &args.Command{
	Name: "update",
	Help: "Update a service",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Service name",
		},
	},
	Options: []args.Option{
		{
			Long: "display",
			Type: args.OptionTypeParameter,
			Help: "Service display name",
		},
		{
			Long: "audience",
			Type: args.OptionTypeParameter,
			Help: "Service audience",
		},
		{
			Long: "redirect",
			Type: args.OptionTypeParameter,
			Help: "Redirect URL",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, DEFAULT_CFG_DIR, "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("service name is required")
		}

		display := i.GetParameter("display")
		audience := i.GetParameter("audience")
		redirect := i.GetParameter("redirect")
		if display == nil && audience == nil && redirect == nil {
			return fmt.Errorf("at least one of --display, --audience, or --redirect is required")
		}

		payload := updateServiceRequest{
			Display:  display,
			Audience: audience,
			Redirect: redirect,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		if err := client.Put("/services/"+name, body, nil); err != nil {
			return err
		}

		fmt.Println("ok")
		return nil
	},
}

var servicesDeleteCmd = &args.Command{
	Name: "delete",
	Help: "Delete a service",
	Operands: []args.Operand{
		{
			Name: "name",
			Help: "Service name",
		},
	},
	Handler: func(i *args.Input) error {
		client, err := envs.ResolveClient(i, DEFAULT_CFG_DIR, "/api/v1")
		if err != nil {
			return err
		}

		name := i.GetOperand("name")
		if name == "" {
			return fmt.Errorf("service name is required")
		}

		if err := client.Delete("/services/"+name, nil); err != nil {
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
