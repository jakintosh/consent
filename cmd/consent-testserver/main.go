package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"embed"
	"encoding/base64"
	"encoding/json"
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

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/version"
	"git.sr.ht/~jakintosh/consent/internal/app"
	"git.sr.ht/~jakintosh/consent/pkg/api"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*
var templatesFS embed.FS

var root = &args.Command{
	Name: "consent-testserver",
	Help: "Test server for integration testing",
	Config: &args.Config{
		Author: "jakintosh",
		HelpOption: &args.HelpOption{
			Short: 'h',
			Long:  "help",
		},
	},
	Options: []args.Option{
		{
			Long: "listen",
			Type: args.OptionTypeParameter,
			Help: "Listen address (default uses ephemeral port)",
		},
		{
			Long: "issuer-domain",
			Type: args.OptionTypeParameter,
			Help: "Issuer domain for JWT tokens",
		},
		{
			Long: "service-name",
			Type: args.OptionTypeParameter,
			Help: "Service name (used as filename)",
		},
		{
			Long: "service-display",
			Type: args.OptionTypeParameter,
			Help: "Service display name",
		},
		{
			Long: "service-audience",
			Type: args.OptionTypeParameter,
			Help: "Service audience",
		},
		{
			Long: "service-redirect",
			Type: args.OptionTypeParameter,
			Help: "Service redirect URL (required)",
		},
		{
			Long: "user",
			Type: args.OptionTypeArray,
			Help: "User credentials in format 'handle:password' (repeatable)",
		},
		{
			Long: "data-dir",
			Type: args.OptionTypeParameter,
			Help: "Data directory (uses temp dir if not set)",
		},
		{
			Long: "keep",
			Type: args.OptionTypeFlag,
			Help: "Keep data directory on exit",
		},
		{
			Long: "quiet",
			Type: args.OptionTypeFlag,
			Help: "Suppress log output",
		},
		{
			Short: 'v',
			Long:  "verbose",
			Type:  args.OptionTypeFlag,
			Help:  "Verbose output",
		},
	},
	Subcommands: []*args.Command{
		version.Command(VersionInfo),
	},
	Handler: func(i *args.Input) error {

		// Parse configuration from input
		cfg, err := parseConfig(i)
		if err != nil {
			return err
		}

		// Suppress logs if requested
		if cfg.Quiet {
			log.SetOutput(io.Discard)
		}

		// Create workspace
		workspace, cleanup, err := createWorkspace(cfg)
		if err != nil {
			return fmt.Errorf("failed to create workspace: %v", err)
		}
		defer cleanup()

		// Generate keys
		signingKey, verificationKeyDER, err := generateKeys(workspace.CredentialsDir)
		if err != nil {
			return fmt.Errorf("failed to generate keys: %v", err)
		}

		// Write templates
		if err := writeTemplates(workspace.TemplatesDir); err != nil {
			return fmt.Errorf("failed to write templates: %v", err)
		}

		// Write service definition
		if err := writeServiceDefinition(workspace.ServicesDir, cfg); err != nil {
			return fmt.Errorf("failed to write service definition: %v", err)
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
			return fmt.Errorf("failed to seed users: %v", err)
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
			return fmt.Errorf("failed to listen: %v", err)
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
			return fmt.Errorf("failed to encode JSON contract: %v", err)
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
			return fmt.Errorf("server error: %v", err)
		case sig := <-sigChan:
			log.Printf("received signal %v, shutting down\n", sig)
		}

		return nil
	},
}

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
	BaseURL      string        `json:"base_url"`
	IssuerDomain string        `json:"issuer_domain"`
	Paths        OutputPaths   `json:"paths"`
	Service      OutputService `json:"service"`
	Users        []OutputUser  `json:"users"`
	Keys         OutputKeys    `json:"keys"`
}

type OutputPaths struct {
	DataDir             string `json:"data_dir"`
	DBPath              string `json:"db_path"`
	ServicesDir         string `json:"services_dir"`
	CredentialsDir      string `json:"credentials_dir"`
	VerificationKeyPath string `json:"verification_key_path"`
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

func main() {
	root.Parse()
}

func parseConfig(i *args.Input) (Config, error) {
	var cfg Config

	// Read options with defaults
	cfg.ListenAddr = i.GetParameterOr("listen", "127.0.0.1:0")
	cfg.IssuerDomain = i.GetParameterOr("issuer-domain", "consent.test")
	cfg.ServiceName = i.GetParameterOr("service-name", "test-service")
	cfg.ServiceDisplay = i.GetParameterOr("service-display", "Test Service")
	cfg.ServiceAudience = i.GetParameterOr("service-audience", "test-audience")
	cfg.DataDir = i.GetParameterOr("data-dir", "")
	cfg.Keep = i.GetFlag("keep")
	cfg.Quiet = i.GetFlag("quiet")

	// Service redirect is required
	if redirect := i.GetParameter("service-redirect"); redirect != nil {
		cfg.ServiceRedirect = *redirect
	} else {
		return cfg, fmt.Errorf("--service-redirect is required")
	}

	// Parse user credentials from array
	userStrings := i.GetArray("user")
	if len(userStrings) == 0 {
		// Default user
		cfg.Users = []UserCredentials{{Handle: "test", Password: "test"}}
	} else {
		cfg.Users = make([]UserCredentials, 0, len(userStrings))
		for _, userStr := range userStrings {
			parts := strings.SplitN(userStr, ":", 2)
			if len(parts) != 2 {
				return cfg, fmt.Errorf("user must be in format 'handle:password', got: %s", userStr)
			}
			cfg.Users = append(cfg.Users, UserCredentials{Handle: parts[0], Password: parts[1]})
		}
	}

	return cfg, nil
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
