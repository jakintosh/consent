package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/envs"
	"git.sr.ht/~jakintosh/command-go/pkg/version"
)

const (
	BIN_AUTHOR      = "jakintosh"
	DEFAULT_CFG_DIR = "~/.config/consent"
)

var root = &args.Command{
	Name: "consent",
	Help: "OAuth authorization server",
	Config: &args.Config{
		Author: BIN_AUTHOR,
		HelpOption: &args.HelpOption{
			Short: 'h',
			Long:  "help",
		},
	},
	Options: envs.ConfigOptions,
	Subcommands: []*args.Command{
		envs.Command(DEFAULT_CFG_DIR),
		serveCmd,
		apiCmd,
		version.Command(VersionInfo),
	},
}

func main() {
	root.Parse()
}

func resolveOption(
	i *args.Input,
	optionName string,
	envVarName string,
	defaultValue string,
) string {
	if param := i.GetParameter(optionName); param != nil {
		return *param
	}

	if envVal := os.Getenv(envVarName); envVal != "" {
		return envVal
	}

	return defaultValue
}

func resolveFlag(
	i *args.Input,
	flagName string,
	envVarName string,
) bool {
	if i.GetFlag(flagName) {
		return true
	}

	envValue := strings.TrimSpace(os.Getenv(envVarName))
	if envValue == "" {
		return false
	}

	enabled, err := strconv.ParseBool(envValue)
	if err == nil {
		return enabled
	}

	switch strings.ToLower(envValue) {
	case "yes", "on", "y":
		return true
	case "no", "off", "n":
		return false
	default:
		return false
	}
}

func loadCredential(
	name string,
	credsDir string,
) []byte {
	credPath := filepath.Join(credsDir, name)
	cred, err := os.ReadFile(credPath)
	if err != nil {
		log.Fatalf("failed to load required credential '%s': %v\n", name, err)
	}
	return cred
}
