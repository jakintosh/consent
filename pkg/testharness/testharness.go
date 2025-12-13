package testharness

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Config holds configuration for starting the test harness.
type Config struct {
	RedirectURL     string // required
	ServiceName     string
	ServiceAudience string
	ServiceDisplay  string
	IssuerDomain    string
	Users           []User
	ListenAddr      string
	DataDir         string
	Keep            bool
	BinaryPath      string
	Quiet           bool
}

// User holds test user credentials.
type User struct {
	Handle   string
	Password string
}

// Harness represents a running consent-testserver instance.
type Harness struct {
	BaseURL             string
	IssuerDomain        string
	VerificationKeyPath string
	VerificationKeyDER  []byte
	ServiceName         string
	ServiceAudience     string
	ServiceRedirect     string
	Users               []User

	// Internal state
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

// outputContract matches the JSON structure from consent-testserver
type outputContract struct {
	BaseURL      string        `json:"base_url"`
	IssuerDomain string        `json:"issuer_domain"`
	Paths        outputPaths   `json:"paths"`
	Service      outputService `json:"service"`
	Users        []outputUser  `json:"users"`
	Keys         outputKeys    `json:"keys"`
}

type outputPaths struct {
	DataDir             string `json:"data_dir"`
	DBPath              string `json:"db_path"`
	ServicesDir         string `json:"services_dir"`
	CredentialsDir      string `json:"credentials_dir"`
	VerificationKeyPath string `json:"verification_key_path"`
}

type outputService struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

type outputUser struct {
	Handle   string `json:"handle"`
	Password string `json:"password"`
}

type outputKeys struct {
	VerificationKeyDERBase64 string `json:"verification_key_der_base64"`
}

// Start spawns a consent-testserver and returns a handle to it.
// It registers cleanup with t.Cleanup().
func Start(t *testing.T, cfg Config) *Harness {
	t.Helper()

	if cfg.RedirectURL == "" {
		t.Fatal("RedirectURL is required")
	}

	// Find binary
	binaryPath := findBinary(cfg.BinaryPath)
	if binaryPath == "" {
		t.Fatal("consent-testserver binary not found (check PATH or set Config.BinaryPath or CONSENT_TESTSERVER_BIN)")
	}

	// Build arguments
	args := buildArgs(cfg)

	// Create context for process lifecycle
	ctx, cancel := context.WithCancel(context.Background())

	// Start process
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("failed to start consent-testserver: %v", err)
	}

	// Read first line (JSON contract) from stdout
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		cancel()
		cmd.Wait()
		t.Fatal("failed to read JSON contract from consent-testserver")
	}

	var contract outputContract
	if err := json.Unmarshal(scanner.Bytes(), &contract); err != nil {
		cancel()
		cmd.Wait()
		t.Fatalf("failed to parse JSON contract: %v", err)
	}

	// Decode verification key from base64
	verificationKeyDER, err := base64.StdEncoding.DecodeString(contract.Keys.VerificationKeyDERBase64)
	if err != nil {
		cancel()
		cmd.Wait()
		t.Fatalf("failed to decode verification key: %v", err)
	}

	// Stream remaining logs to test output if not quiet
	if !cfg.Quiet {
		go func() {
			for scanner.Scan() {
				t.Logf("[consent-testserver] %s", scanner.Text())
			}
		}()

		go func() {
			stderrScanner := bufio.NewScanner(stderr)
			for stderrScanner.Scan() {
				t.Logf("[consent-testserver stderr] %s", stderrScanner.Text())
			}
		}()
	}

	// Build harness
	harness := &Harness{
		BaseURL:             contract.BaseURL,
		IssuerDomain:        contract.IssuerDomain,
		VerificationKeyPath: contract.Paths.VerificationKeyPath,
		VerificationKeyDER:  verificationKeyDER,
		ServiceName:         contract.Service.Name,
		ServiceAudience:     contract.Service.Audience,
		ServiceRedirect:     contract.Service.Redirect,
		Users:               make([]User, len(contract.Users)),
		cmd:                 cmd,
		cancel:              cancel,
	}

	for i, user := range contract.Users {
		harness.Users[i] = User{Handle: user.Handle, Password: user.Password}
	}

	// Register cleanup
	t.Cleanup(func() {
		if err := harness.Close(); err != nil {
			t.Logf("warning: harness cleanup failed: %v", err)
		}
	})

	return harness
}

// Close terminates the consent-testserver process.
func (h *Harness) Close() error {
	if h.cancel != nil {
		h.cancel()
	}

	if h.cmd == nil || h.cmd.Process == nil {
		return nil
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		done <- h.cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown takes too long
		if err := h.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("force kill: %w", err)
		}
		return fmt.Errorf("timeout waiting for graceful shutdown, process killed")
	}
}

func findBinary(configPath string) string {
	// Check config path first
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check environment variable
	if envPath := os.Getenv("CONSENT_TESTSERVER_BIN"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Check PATH
	if pathBinary, err := exec.LookPath("consent-testserver"); err == nil {
		return pathBinary
	}

	return ""
}

func buildArgs(cfg Config) []string {
	args := []string{
		"--service-redirect", cfg.RedirectURL,
	}

	if cfg.ServiceName != "" {
		args = append(args, "--service-name", cfg.ServiceName)
	}

	if cfg.ServiceAudience != "" {
		args = append(args, "--service-audience", cfg.ServiceAudience)
	}

	if cfg.ServiceDisplay != "" {
		args = append(args, "--service-display", cfg.ServiceDisplay)
	}

	if cfg.IssuerDomain != "" {
		args = append(args, "--issuer-domain", cfg.IssuerDomain)
	}

	if cfg.ListenAddr != "" {
		args = append(args, "--listen", cfg.ListenAddr)
	}

	if cfg.DataDir != "" {
		args = append(args, "--data-dir", cfg.DataDir)
	}

	if cfg.Keep {
		args = append(args, "--keep")
	}

	if cfg.Quiet {
		args = append(args, "--quiet")
	}

	for _, user := range cfg.Users {
		args = append(args, "--user", fmt.Sprintf("%s:%s", user.Handle, user.Password))
	}

	return args
}
