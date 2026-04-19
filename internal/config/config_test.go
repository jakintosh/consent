package config_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/config"
)

func TestLoad_DefaultWhenConfigMissing(t *testing.T) {
	t.Parallel()

	roots, err := config.ResolveRoots(filepath.Join(t.TempDir(), "cfg"), filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("ResolveRoots failed: %v", err)
	}

	cfg, err := config.Load(config.BuildPaths(roots))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg != config.Default() {
		t.Fatalf("Load() = %#v, want %#v", cfg, config.Default())
	}
}

func TestLoad_StrictUnknownField(t *testing.T) {
	t.Parallel()

	roots, err := config.ResolveRoots(filepath.Join(t.TempDir(), "cfg"), filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("ResolveRoots failed: %v", err)
	}

	paths := config.BuildPaths(roots)
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	payload := []byte("server:\n  publicURL: http://localhost:9001\n  issuerDomain: localhost\n  port: 9001\n  devMode: true\n  extra: nope\n")
	if err := os.WriteFile(paths.ConfigFile, payload, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err = config.Load(paths)
	if err == nil || !strings.Contains(err.Error(), "field extra not found") {
		t.Fatalf("Load error = %v, want unknown field error", err)
	}
}

func TestResolve_UsesOverridesAndSecretEnv(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "cfg")
	dataDir := filepath.Join(t.TempDir(), "data")
	roots, err := config.ResolveRoots(configDir, dataDir)
	if err != nil {
		t.Fatalf("ResolveRoots failed: %v", err)
	}

	paths := config.BuildPaths(roots)
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if err := config.Save(paths, config.Config{
		Server: config.ServerConfig{
			PublicURL:    "http://example.test:9001",
			IssuerDomain: "issuer-from-file",
			Port:         9001,
			DevMode:      false,
		},
	}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	encodedKey, err := generateSigningKeyBase64()
	if err != nil {
		t.Fatalf("generateSigningKeyBase64 failed: %v", err)
	}

	t.Setenv(config.EnvSigningKeyDERBase64, encodedKey)
	t.Setenv(config.EnvBootstrapAPIKey, "bootstrap.from.env")

	overrideURL := "http://override.test:7777"
	overridePort := 7777
	overrideDevMode := true

	opts := config.ResolveOptions{
		Overrides: config.Overrides{
			PublicURL: &overrideURL,
			Port:      &overridePort,
			DevMode:   &overrideDevMode,
		},
		ConfigDir:              configDir,
		DataDir:                dataDir,
		RequireSigningKey:      true,
		RequireBootstrapAPIKey: true,
	}
	runtime, err := config.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if runtime.Server.PublicURL != overrideURL {
		t.Fatalf("PublicURL = %q, want %q", runtime.Server.PublicURL, overrideURL)
	}
	if runtime.Server.Port != overridePort {
		t.Fatalf("Port = %d, want %d", runtime.Server.Port, overridePort)
	}
	if !runtime.Server.DevMode {
		t.Fatal("DevMode = false, want true")
	}
	if runtime.Secrets.SigningKey == nil {
		t.Fatal("SigningKey = nil, want parsed key")
	}
	if runtime.Secrets.BootstrapAPIKey != "bootstrap.from.env" {
		t.Fatalf("BootstrapAPIKey = %q, want env value", runtime.Secrets.BootstrapAPIKey)
	}
	if !runtime.Source.SigningKeyFromEnv || !runtime.Source.BootstrapAPIKeyFromEnv {
		t.Fatalf("secret source flags = %#v, want env-backed", runtime.Source)
	}

	view := runtime.View()
	if !view.Secrets.SigningKeySet || !view.Secrets.BootstrapAPIKeySet {
		t.Fatalf("View secrets = %#v, want redacted presence flags", view.Secrets)
	}
	if view.Server.ListenAddress != ":7777" {
		t.Fatalf("ListenAddress = %q, want :7777", view.Server.ListenAddress)
	}
}

func TestInit_IsNonDestructiveUnlessForced(t *testing.T) {
	t.Parallel()

	configDir := filepath.Join(t.TempDir(), "cfg")
	dataDir := filepath.Join(t.TempDir(), "data")

	result, err := config.Init(config.InitOptions{
		ConfigDir: configDir,
		DataDir:   dataDir,
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	for _, path := range []string{
		result.Paths.ConfigFile,
		result.Paths.SigningKeyFile,
		result.Paths.VerificationKeyFile,
		result.Paths.BootstrapAPIKeyFile,
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat(%s) failed: %v", path, err)
		}
	}

	_, err = config.Init(config.InitOptions{
		ConfigDir: configDir,
		DataDir:   dataDir,
	})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("second Init error = %v, want overwrite refusal", err)
	}

	overrideURL := "http://forced.test:9100"
	_, err = config.Init(config.InitOptions{
		ConfigDir: configDir,
		DataDir:   dataDir,
		Force:     true,
		Overrides: config.Overrides{PublicURL: &overrideURL},
	})
	if err != nil {
		t.Fatalf("forced Init failed: %v", err)
	}

	roots, err := config.ResolveRoots(configDir, dataDir)
	if err != nil {
		t.Fatalf("ResolveRoots failed: %v", err)
	}

	cfg, err := config.Load(config.BuildPaths(roots))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.PublicURL != overrideURL {
		t.Fatalf("PublicURL = %q, want %q", cfg.Server.PublicURL, overrideURL)
	}
}

func generateSigningKeyBase64() (string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", err
	}

	privateDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(privateDER), nil
}
