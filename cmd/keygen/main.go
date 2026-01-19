package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	signingKeyName      = "signing_key"
	verificationKeyName = "verification_key.der"
)

func main() {
	outDir := flag.String("out", "./secrets", "Output directory")
	flag.Parse()

	if err := run(*outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	if outDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	signingKeyPath := filepath.Join(outDir, signingKeyName)
	verificationKeyPath := filepath.Join(outDir, verificationKeyName)

	if err := ensureNotExists(signingKeyPath); err != nil {
		return err
	}
	if err := ensureNotExists(verificationKeyPath); err != nil {
		return err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	privateBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}

	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal public key: %w", err)
	}

	if err := os.WriteFile(signingKeyPath, privateBytes, 0o600); err != nil {
		return fmt.Errorf("write signing key: %w", err)
	}
	if err := os.WriteFile(verificationKeyPath, publicBytes, 0o644); err != nil {
		return fmt.Errorf("write verification key: %w", err)
	}

	return nil
}

func ensureNotExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("refusing to overwrite existing file: %s", path)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("stat %s: %w", path, err)
}
