package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
)

type InitOptions struct {
	ConfigDir string
	DataDir   string
	Force     bool
	Overrides Overrides
}

type InitResult struct {
	Config Config
	Paths  Paths
	Roots  Roots
}

func Init(
	opts InitOptions,
) (
	InitResult,
	error,
) {
	roots, err := ResolveRoots(opts.ConfigDir, opts.DataDir)
	if err != nil {
		return InitResult{}, err
	}

	paths := BuildPaths(roots)
	cfg := Default().WithOverrides(opts.Overrides)
	if err := cfg.Validate(); err != nil {
		return InitResult{}, err
	}

	signingKeyDER, verificationKeyDER, err := resolveKeyMaterial()
	if err != nil {
		return InitResult{}, err
	}

	bootstrapAPIKey, err := resolveBootstrapAPIKey()
	if err != nil {
		return InitResult{}, err
	}

	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		return InitResult{}, fmt.Errorf("config: create %s: %w", paths.ConfigDir, err)
	}
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		return InitResult{}, fmt.Errorf("config: create %s: %w", paths.DataDir, err)
	}
	if err := os.MkdirAll(paths.SecretsDir, 0o700); err != nil {
		return InitResult{}, fmt.Errorf("config: create %s: %w", paths.SecretsDir, err)
	}

	if err := writeFileAtomic(paths.ConfigFile, mustMarshalConfig(cfg), 0o644, opts.Force); err != nil {
		return InitResult{}, err
	}
	if err := writeFileAtomic(paths.SigningKeyFile, signingKeyDER, 0o600, opts.Force); err != nil {
		return InitResult{}, err
	}
	if err := writeFileAtomic(paths.VerificationKeyFile, verificationKeyDER, 0o644, opts.Force); err != nil {
		return InitResult{}, err
	}
	if err := writeFileAtomic(paths.BootstrapAPIKeyFile, []byte(bootstrapAPIKey+"\n"), 0o600, opts.Force); err != nil {
		return InitResult{}, err
	}

	return InitResult{
		Config: cfg,
		Paths:  paths,
		Roots:  roots,
	}, nil
}

func resolveKeyMaterial() (
	[]byte,
	[]byte,
	error,
) {
	var privateDER []byte
	if value, ok := os.LookupEnv(EnvSigningKeyDERBase64); ok && strings.TrimSpace(value) != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			return nil, nil, fmt.Errorf("config: decode %s: %w", EnvSigningKeyDERBase64, err)
		}
		privateDER = decoded
	}

	if len(privateDER) == 0 {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, fmt.Errorf("config: generate signing key: %w", err)
		}

		privateDER, err = x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, nil, fmt.Errorf("config: encode signing key: %w", err)
		}

		publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return nil, nil, fmt.Errorf("config: encode verification key: %w", err)
		}

		return privateDER, publicDER, nil
	}

	privateKey, err := x509.ParseECPrivateKey(privateDER)
	if err != nil {
		return nil, nil, fmt.Errorf("config: parse %s: %w", EnvSigningKeyDERBase64, err)
	}

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("config: encode verification key: %w", err)
	}

	return privateDER, publicDER, nil
}

func resolveBootstrapAPIKey() (
	string,
	error,
) {
	if value := strings.TrimSpace(os.Getenv(EnvBootstrapAPIKey)); value != "" {
		return value, nil
	}

	key, err := keys.GenerateBootstrapKey()
	if err != nil {
		return "", fmt.Errorf("config: generate bootstrap api key: %w", err)
	}

	return key, nil
}

func mustMarshalConfig(
	cfg Config,
) []byte {
	data, err := marshalConfig(cfg)
	if err != nil {
		panic(err)
	}
	return data
}

func writeFileAtomic(
	path string,
	data []byte,
	mode os.FileMode,
	overwrite bool,
) error {
	if !overwrite {
		exists, err := fileExists(path)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("config: refusing to overwrite existing file: %s", path)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: create parent for %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("config: create temp file for %s: %w", path, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: write temp file for %s: %w", path, err)
	}

	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("config: chmod temp file for %s: %w", path, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp file for %s: %w", path, err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("config: rename temp file for %s: %w", path, err)
	}

	return nil
}
