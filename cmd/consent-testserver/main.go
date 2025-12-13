package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"git.sr.ht/~jakintosh/consent/internal/app"
	"git.sr.ht/~jakintosh/consent/pkg/api"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*
var templatesFS embed.FS

// Config holds all command-line configuration
type Config struct {
	ListenAddr      string
	IssuerDomain    string
	ServiceName     string
	ServiceDisplay  string
	ServiceAudience string
	ServiceRedirect string
	Users           []UserCredentials
	DataDir         string
	Keep            bool
	Quiet           bool
}

// UserCredentials holds username and password
type UserCredentials struct {
	Handle   string
	Password string
}

// OutputContract is the JSON structure emitted on stdout
type OutputContract struct {
	BaseURL      string               `json:"base_url"`
	IssuerDomain string               `json:"issuer_domain"`
	Paths        OutputPaths          `json:"paths"`
	Service      OutputService        `json:"service"`
	Users        []OutputUser         `json:"users"`
	Keys         OutputKeys           `json:"keys"`
}

type OutputPaths struct {
	DataDir               string `json:"data_dir"`
	DBPath                string `json:"db_path"`
	ServicesDir           string `json:"services_dir"`
	CredentialsDir        string `json:"credentials_dir"`
	VerificationKeyPath   string `json:"verification_key_path"`
}

type OutputService struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

type OutputUser struct {
	Handle   string `json:"handle"`
	Password string `json:"password"`
}

type OutputKeys struct {
	VerificationKeyDERBase64 string `json:"verification_key_der_base64"`
}

// UserFlag is a custom flag type for repeatable --user flags
type UserFlag []UserCredentials

func (u *UserFlag) String() string {
	return fmt.Sprintf("%v", *u)
}

func (u *UserFlag) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("user must be in format 'handle:password'")
	}
	*u = append(*u, UserCredentials{Handle: parts[0], Password: parts[1]})
	return nil
}

func main() {
	// Parse flags
	cfg := parseFlags()

	// Suppress logs if requested
	if cfg.Quiet {
		log.SetOutput(io.Discard)
	}

	// Create workspace
	workspace, cleanup, err := createWorkspace(cfg)
	if err != nil {
		log.Fatalf("failed to create workspace: %v\n", err)
	}
	defer cleanup()

	// Generate keys
	signingKey, verificationKeyDER, err := generateKeys(workspace.CredentialsDir)
	if err != nil {
		log.Fatalf("failed to generate keys: %v\n", err)
	}

	// Write templates
	if err := writeTemplates(workspace.TemplatesDir); err != nil {
		log.Fatalf("failed to write templates: %v\n", err)
	}

	// Write service definition
	if err := writeServiceDefinition(workspace.ServicesDir, cfg); err != nil {
		log.Fatalf("failed to write service definition: %v\n", err)
	}

	// Initialize server components
	services := api.NewDynamicServicesDirectory(workspace.ServicesDir)
	templates := app.NewDynamicTemplatesDirectory(workspace.TemplatesDir)
	issuer, validator := tokens.InitServer(signingKey, cfg.IssuerDomain)

	// Initialize endpoints
	app.Init(services, templates)
	api.Init(issuer, validator, services, workspace.DBPath)

	// Seed test users
	if err := seedUsers(cfg.Users); err != nil {
		log.Fatalf("failed to seed users: %v\n", err)
	}

	// Build router
	r := mux.NewRouter()
	r.HandleFunc("/", app.Home)
	r.HandleFunc("/login", app.Login)
	apiRouter := r.PathPrefix("/api").Subrouter()
	api.BuildRouter(apiRouter)

	// Start HTTP server with ephemeral port
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	baseURL := fmt.Sprintf("http://%s:%d", addr.IP, addr.Port)

	// Emit JSON contract to stdout
	contract := OutputContract{
		BaseURL:      baseURL,
		IssuerDomain: cfg.IssuerDomain,
		Paths: OutputPaths{
			DataDir:             workspace.DataDir,
			DBPath:              workspace.DBPath,
			ServicesDir:         workspace.ServicesDir,
			CredentialsDir:      workspace.CredentialsDir,
			VerificationKeyPath: filepath.Join(workspace.CredentialsDir, "verification_key.der"),
		},
		Service: OutputService{
			Name:     cfg.ServiceName,
			Display:  cfg.ServiceDisplay,
			Audience: cfg.ServiceAudience,
			Redirect: cfg.ServiceRedirect,
		},
		Users: make([]OutputUser, len(cfg.Users)),
		Keys: OutputKeys{
			VerificationKeyDERBase64: base64.StdEncoding.EncodeToString(verificationKeyDER),
		},
	}

	for i, user := range cfg.Users {
		contract.Users[i] = OutputUser{Handle: user.Handle, Password: user.Password}
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(contract); err != nil {
		log.Fatalf("failed to encode JSON contract: %v\n", err)
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- http.Serve(listener, r)
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v\n", err)
	case sig := <-sigChan:
		log.Printf("received signal %v, shutting down\n", sig)
	}
}

func parseFlags() Config {
	var cfg Config
	var users UserFlag

	flag.StringVar(&cfg.ListenAddr, "listen", "127.0.0.1:0", "Listen address (default uses ephemeral port)")
	flag.StringVar(&cfg.IssuerDomain, "issuer-domain", "consent.test", "Issuer domain for JWT tokens")
	flag.StringVar(&cfg.ServiceName, "service-name", "test-service", "Service name (used as filename)")
	flag.StringVar(&cfg.ServiceDisplay, "service-display", "Test Service", "Service display name")
	flag.StringVar(&cfg.ServiceAudience, "service-audience", "test-audience", "Service audience")
	flag.StringVar(&cfg.ServiceRedirect, "service-redirect", "", "Service redirect URL (required)")
	flag.Var(&users, "user", "User credentials in format 'handle:password' (repeatable)")
	flag.StringVar(&cfg.DataDir, "data-dir", "", "Data directory (uses temp dir if not set)")
	flag.BoolVar(&cfg.Keep, "keep", false, "Keep data directory on exit")
	flag.BoolVar(&cfg.Quiet, "quiet", false, "Suppress log output")

	flag.Parse()

	if cfg.ServiceRedirect == "" {
		log.Fatal("--service-redirect is required")
	}

	if len(users) == 0 {
		cfg.Users = []UserCredentials{{Handle: "test", Password: "test"}}
	} else {
		cfg.Users = users
	}

	return cfg
}

type Workspace struct {
	DataDir        string
	DBPath         string
	ServicesDir    string
	CredentialsDir string
	TemplatesDir   string
}

func createWorkspace(cfg Config) (*Workspace, func(), error) {
	var dataDir string
	var shouldCleanup bool

	if cfg.DataDir != "" {
		dataDir = cfg.DataDir
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, nil, err
		}
	} else {
		tempDir, err := os.MkdirTemp("", "consent-testserver-*")
		if err != nil {
			return nil, nil, err
		}
		dataDir = tempDir
		shouldCleanup = !cfg.Keep
	}

	workspace := &Workspace{
		DataDir:        dataDir,
		DBPath:         filepath.Join(dataDir, "db.sqlite"),
		ServicesDir:    filepath.Join(dataDir, "services"),
		CredentialsDir: filepath.Join(dataDir, "credentials"),
		TemplatesDir:   filepath.Join(dataDir, "templates"),
	}

	// Create subdirectories
	for _, dir := range []string{workspace.ServicesDir, workspace.CredentialsDir, workspace.TemplatesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, nil, err
		}
	}

	cleanup := func() {
		if shouldCleanup {
			os.RemoveAll(dataDir)
		}
	}

	return workspace, cleanup, nil
}

func generateKeys(credentialsDir string) (*ecdsa.PrivateKey, []byte, error) {
	// Generate ECDSA P-256 keypair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	// Marshal private key to DER format
	privateKeyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private key: %w", err)
	}

	// Write signing key
	signingKeyPath := filepath.Join(credentialsDir, "signing_key")
	if err := os.WriteFile(signingKeyPath, privateKeyDER, 0600); err != nil {
		return nil, nil, fmt.Errorf("write signing key: %w", err)
	}

	// Marshal public key to DER format
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal public key: %w", err)
	}

	// Write verification key
	verificationKeyPath := filepath.Join(credentialsDir, "verification_key.der")
	if err := os.WriteFile(verificationKeyPath, publicKeyDER, 0644); err != nil {
		return nil, nil, fmt.Errorf("write verification key: %w", err)
	}

	return privateKey, publicKeyDER, nil
}

func writeTemplates(templatesDir string) error {
	// Read embedded templates
	files, err := templatesFS.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("read embedded templates: %w", err)
	}

	// Write each template file
	for _, file := range files {
		content, err := templatesFS.ReadFile(filepath.Join("templates", file.Name()))
		if err != nil {
			return fmt.Errorf("read %s: %w", file.Name(), err)
		}

		destPath := filepath.Join(templatesDir, file.Name())
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("write %s: %w", file.Name(), err)
		}
	}

	return nil
}

func writeServiceDefinition(servicesDir string, cfg Config) error {
	service := map[string]string{
		"display":  cfg.ServiceDisplay,
		"audience": cfg.ServiceAudience,
		"redirect": cfg.ServiceRedirect,
	}

	data, err := json.MarshalIndent(service, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal service JSON: %w", err)
	}

	servicePath := filepath.Join(servicesDir, cfg.ServiceName)
	if err := os.WriteFile(servicePath, data, 0644); err != nil {
		return fmt.Errorf("write service file: %w", err)
	}

	return nil
}

func seedUsers(users []UserCredentials) error {
	for _, user := range users {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", user.Handle, err)
		}

		if err := api.InsertAccount(user.Handle, hashedPassword); err != nil {
			return fmt.Errorf("insert account %s: %w", user.Handle, err)
		}
	}

	return nil
}
