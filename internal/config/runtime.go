package config

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	EnvSigningKeyDERBase64 = "CONSENT_SIGNING_KEY_DER_BASE64"
	EnvBootstrapAPIKey     = "CONSENT_BOOTSTRAP_API_KEY"
)

type RuntimeOptions struct {
	Overrides              Overrides
	RequireSigningKey      bool
	RequireBootstrapAPIKey bool
}

type SecretSource string

const (
	SecretSourceNone SecretSource = ""
	SecretSourceEnv  SecretSource = "env"
	SecretSourceFile SecretSource = "file"
)

type Runtime struct {
	Config  Config
	Paths   Paths
	Server  RuntimeServer
	Secrets RuntimeSecrets
	Source  RuntimeSource
}

type RuntimeServer struct {
	PublicURL       string
	PublicBaseURL   string
	PublicHost      string
	ParsedPublicURL *url.URL
	AuthorityDomain string
	Port            int
	ListenAddress   string
	DevMode         bool
}

type RuntimeSecrets struct {
	SigningKey      *ecdsa.PrivateKey
	BootstrapAPIKey string
}

type RuntimeSource struct {
	SigningKeySource       SecretSource
	BootstrapAPIKeySource  SecretSource
	VerificationKeyPresent bool
	ConfigFilePresent      bool
}

type View struct {
	Config  Config      `yaml:"config" json:"config"`
	Paths   Paths       `yaml:"paths" json:"paths"`
	Server  ViewServer  `yaml:"server" json:"server"`
	Secrets ViewSecrets `yaml:"secrets" json:"secrets"`
	Source  ViewSource  `yaml:"source" json:"source"`
}

type ViewServer struct {
	PublicURL       string `yaml:"publicURL" json:"publicURL"`
	PublicBaseURL   string `yaml:"publicBaseURL" json:"publicBaseURL"`
	PublicHost      string `yaml:"publicHost" json:"publicHost"`
	AuthorityDomain string `yaml:"authorityDomain" json:"authorityDomain"`
	Port            int    `yaml:"port" json:"port"`
	ListenAddress   string `yaml:"listenAddress" json:"listenAddress"`
	DevMode         bool   `yaml:"devMode" json:"devMode"`
}

type ViewSecrets struct {
	SigningKeySet      bool `yaml:"signingKeySet" json:"signingKeySet"`
	VerificationKeySet bool `yaml:"verificationKeySet" json:"verificationKeySet"`
	BootstrapAPIKeySet bool `yaml:"bootstrapAPIKeySet" json:"bootstrapAPIKeySet"`
}

type ViewSource struct {
	ConfigFilePresent      bool         `yaml:"configFilePresent" json:"configFilePresent"`
	SigningKeySource       SecretSource `yaml:"signingKeySource" json:"signingKeySource"`
	BootstrapAPIKeySource  SecretSource `yaml:"bootstrapAPIKeySource" json:"bootstrapAPIKeySource"`
	VerificationKeyPresent bool         `yaml:"verificationKeyPresent" json:"verificationKeyPresent"`
}

func Resolve(configDir string, dataDir string, opts RuntimeOptions) (Runtime, error) {
	paths, err := resolvePaths(configDir, dataDir)
	if err != nil {
		return Runtime{}, err
	}

	cfg, err := Load(configDir, dataDir)
	if err != nil {
		return Runtime{}, err
	}

	cfg = cfg.WithOverrides(opts.Overrides)
	if err := cfg.Validate(); err != nil {
		return Runtime{}, err
	}

	publicURL, parsedURL, err := normalizePublicURL(cfg.Server.PublicURL)
	if err != nil {
		return Runtime{}, fmt.Errorf("config: %w", err)
	}

	signingKeyDER, signingKeySource, err := loadSecretBytes(paths.SigningKeyFile, EnvSigningKeyDERBase64, true)
	if err != nil {
		return Runtime{}, err
	}

	var signingKey *ecdsa.PrivateKey
	if len(signingKeyDER) > 0 {
		signingKey, err = x509.ParseECPrivateKey(signingKeyDER)
		if err != nil {
			return Runtime{}, fmt.Errorf("config: parse signing key: %w", err)
		}
	} else if opts.RequireSigningKey {
		return Runtime{}, fmt.Errorf("config: signing key is required; set %s or create %s", EnvSigningKeyDERBase64, paths.SigningKeyFile)
	}

	bootstrapAPIKey, bootstrapKeySource, err := loadSecretString(paths.BootstrapAPIKeyFile, EnvBootstrapAPIKey)
	if err != nil {
		return Runtime{}, err
	}
	if bootstrapAPIKey == "" && opts.RequireBootstrapAPIKey {
		return Runtime{}, fmt.Errorf("config: bootstrap api key is required; set %s or create %s", EnvBootstrapAPIKey, paths.BootstrapAPIKeyFile)
	}

	verificationKeyPresent, err := fileExists(paths.VerificationKeyFile)
	if err != nil {
		return Runtime{}, err
	}

	configFilePresent, err := fileExists(paths.ConfigFile)
	if err != nil {
		return Runtime{}, err
	}

	return Runtime{
		Config: cfg,
		Paths:  paths,
		Server: RuntimeServer{
			PublicURL:       publicURL,
			PublicBaseURL:   strings.TrimRight(publicURL, "/"),
			PublicHost:      parsedURL.Host,
			ParsedPublicURL: parsedURL,
			AuthorityDomain: cfg.Server.AuthorityDomain,
			Port:            cfg.Server.Port,
			ListenAddress:   fmt.Sprintf(":%d", cfg.Server.Port),
			DevMode:         cfg.Server.DevMode,
		},
		Secrets: RuntimeSecrets{
			SigningKey:      signingKey,
			BootstrapAPIKey: bootstrapAPIKey,
		},
		Source: RuntimeSource{
			SigningKeySource:       signingKeySource,
			BootstrapAPIKeySource:  bootstrapKeySource,
			VerificationKeyPresent: verificationKeyPresent,
			ConfigFilePresent:      configFilePresent,
		},
	}, nil
}

func (r Runtime) View() View {
	return View{
		Config: r.Config,
		Paths:  r.Paths,
		Server: ViewServer{
			PublicURL:       r.Server.PublicURL,
			PublicBaseURL:   r.Server.PublicBaseURL,
			PublicHost:      r.Server.PublicHost,
			AuthorityDomain: r.Server.AuthorityDomain,
			Port:            r.Server.Port,
			ListenAddress:   r.Server.ListenAddress,
			DevMode:         r.Server.DevMode,
		},
		Secrets: ViewSecrets{
			SigningKeySet:      r.Secrets.SigningKey != nil,
			VerificationKeySet: r.Source.VerificationKeyPresent,
			BootstrapAPIKeySet: strings.TrimSpace(r.Secrets.BootstrapAPIKey) != "",
		},
		Source: ViewSource{
			ConfigFilePresent:      r.Source.ConfigFilePresent,
			SigningKeySource:       r.Source.SigningKeySource,
			BootstrapAPIKeySource:  r.Source.BootstrapAPIKeySource,
			VerificationKeyPresent: r.Source.VerificationKeyPresent,
		},
	}
}

func normalizePublicURL(raw string) (string, *url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return "", nil, fmt.Errorf("server.publicURL must be an absolute URL with scheme and host")
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", nil, fmt.Errorf("server.publicURL must be an absolute URL with scheme and host")
	}

	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", nil, fmt.Errorf("server.publicURL cannot include query or fragment")
	}

	return parsed.String(), parsed, nil
}

func loadSecretBytes(path string, envVar string, base64Decode bool) ([]byte, SecretSource, error) {
	if value, ok := os.LookupEnv(envVar); ok && strings.TrimSpace(value) != "" {
		if !base64Decode {
			return []byte(value), SecretSourceEnv, nil
		}

		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			return nil, SecretSourceEnv, fmt.Errorf("config: decode %s: %w", envVar, err)
		}
		return decoded, SecretSourceEnv, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, SecretSourceNone, nil
		}
		return nil, SecretSourceNone, fmt.Errorf("config: read %s: %w", path, err)
	}

	return data, SecretSourceFile, nil
}

func loadSecretString(path string, envVar string) (string, SecretSource, error) {
	if value, ok := os.LookupEnv(envVar); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value), SecretSourceEnv, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", SecretSourceNone, nil
		}
		return "", SecretSourceNone, fmt.Errorf("config: read %s: %w", path, err)
	}

	return strings.TrimSpace(string(data)), SecretSourceFile, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("config: stat %s: %w", path, err)
}
