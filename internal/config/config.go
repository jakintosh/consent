package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	AppName            = "consent"
	ConfigFileName     = "config.yaml"
	SecretsDirName     = "secrets"
	SigningKeyFileName = "signing_key"
	VerifyKeyFileName  = "verification_key.der"
	APIKeyFileName     = "api_key"
	DatabaseFileName   = "auth.db"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	PublicURL    string `yaml:"publicURL"`
	IssuerDomain string `yaml:"issuerDomain"`
	Port         int    `yaml:"port"`
	DevMode      bool   `yaml:"devMode"`
}

type Roots struct {
	ConfigDir string
	DataDir   string
}

type Paths struct {
	ConfigDir           string `yaml:"configDir" json:"configDir"`
	DataDir             string `yaml:"dataDir" json:"dataDir"`
	ConfigFile          string `yaml:"configFile" json:"configFile"`
	SecretsDir          string `yaml:"secretsDir" json:"secretsDir"`
	SigningKeyFile      string `yaml:"signingKeyFile" json:"signingKeyFile"`
	VerificationKeyFile string `yaml:"verificationKeyFile" json:"verificationKeyFile"`
	BootstrapAPIKeyFile string `yaml:"bootstrapAPIKeyFile" json:"bootstrapAPIKeyFile"`
	DatabaseFile        string `yaml:"databaseFile" json:"databaseFile"`
}

type Overrides struct {
	PublicURL    *string
	IssuerDomain *string
	Port         *int
	DevMode      *bool
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			PublicURL:    "http://localhost:9001",
			IssuerDomain: "localhost",
			Port:         9001,
			DevMode:      true,
		},
	}
}

func ResolveRoots(
	configDir string,
	dataDir string,
) (
	Roots,
	error,
) {
	resolvedConfigDir := configDir
	if strings.TrimSpace(resolvedConfigDir) == "" {
		resolvedConfigDir = defaultConfigDir()
	}

	resolvedDataDir := dataDir
	if strings.TrimSpace(resolvedDataDir) == "" {
		resolvedDataDir = defaultDataDir()
	}

	var err error
	resolvedConfigDir, err = expandPath(resolvedConfigDir)
	if err != nil {
		return Roots{}, err
	}

	resolvedDataDir, err = expandPath(resolvedDataDir)
	if err != nil {
		return Roots{}, err
	}

	return Roots{
		ConfigDir: resolvedConfigDir,
		DataDir:   resolvedDataDir,
	}, nil
}

func BuildPaths(
	roots Roots,
) Paths {
	secretsDir := filepath.Join(roots.ConfigDir, SecretsDirName)

	return Paths{
		ConfigDir:           roots.ConfigDir,
		DataDir:             roots.DataDir,
		ConfigFile:          filepath.Join(roots.ConfigDir, ConfigFileName),
		SecretsDir:          secretsDir,
		SigningKeyFile:      filepath.Join(secretsDir, SigningKeyFileName),
		VerificationKeyFile: filepath.Join(secretsDir, VerifyKeyFileName),
		BootstrapAPIKeyFile: filepath.Join(secretsDir, APIKeyFileName),
		DatabaseFile:        filepath.Join(roots.DataDir, DatabaseFileName),
	}
}

func Load(paths Paths) (
	Config,
	error,
) {
	cfg := Default()

	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("config: read %s: %w", paths.ConfigFile, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: decode %s: %w", paths.ConfigFile, err)
	}

	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Save(
	paths Paths,
	cfg Config,
) error {
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return err
	}

	data, err := marshalConfig(cfg)
	if err != nil {
		return err
	}

	return writeFileAtomic(paths.ConfigFile, data, 0o644, true)
}

func (c *Config) Normalize() {
	c.Server.PublicURL = strings.TrimSpace(c.Server.PublicURL)
	c.Server.IssuerDomain = strings.TrimSpace(c.Server.IssuerDomain)
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Server.PublicURL) == "" {
		return fmt.Errorf("config: server.publicURL is required")
	}

	if _, _, err := normalizePublicURL(c.Server.PublicURL); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if strings.TrimSpace(c.Server.IssuerDomain) == "" {
		return fmt.Errorf("config: server.issuerDomain is required")
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("config: server.port must be between 1 and 65535")
	}

	return nil
}

func (c Config) WithOverrides(
	overrides Overrides,
) Config {
	resolved := c

	if overrides.PublicURL != nil {
		resolved.Server.PublicURL = *overrides.PublicURL
	}
	if overrides.IssuerDomain != nil {
		resolved.Server.IssuerDomain = *overrides.IssuerDomain
	}
	if overrides.Port != nil {
		resolved.Server.Port = *overrides.Port
	}
	if overrides.DevMode != nil {
		resolved.Server.DevMode = *overrides.DevMode
	}

	resolved.Normalize()
	return resolved
}

func defaultConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join("~", ".config", AppName)
	}
	return filepath.Join(base, AppName)
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".local", "share", AppName)
	}
	return filepath.Join(home, ".local", "share", AppName)
}

func expandPath(
	path string,
) (
	string,
	error,
) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("config: path is empty")
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("config: resolve home directory: %w", err)
		}
		if trimmed == "~" {
			trimmed = home
		} else {
			trimmed = filepath.Join(home, trimmed[2:])
		}
	}

	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("config: resolve %s: %w", path, err)
	}

	return abs, nil
}

func marshalConfig(
	cfg Config,
) (
	[]byte,
	error,
) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("config: encode yaml: %w", err)
	}
	return data, nil
}
