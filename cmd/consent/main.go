package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"git.sr.ht/~jakintosh/consent/internal/app"
	"git.sr.ht/~jakintosh/consent/pkg/api"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
	"github.com/gorilla/mux"
)

func main() {

	// read env vars
	dbPath := readEnvVar("DB_PATH")
	issuerDomain := readEnvVar("ISSUER_DOMAIN")
	templatesPath := readEnvVar("TEMPLATES_PATH")
	servicesPath := readEnvVar("SERVICES_PATH")
	port := fmt.Sprintf(":%s", readEnvVar("PORT"))

	// load credentials
	credsDir := readEnvVar("CREDENTIALS_DIRECTORY")
	signingKeyRaw := loadCredential("signing_key", credsDir)
	signingKey, err := x509.ParseECPrivateKey(signingKeyRaw)
	if err != nil {
		log.Fatalf("failed to parse ecdsa signing key from signing_key: %v\n", err)
	}

	// init program services
	services := api.NewDynamicServicesDirectory(servicesPath)
	templates := app.NewDynamicTemplatesDirectory(templatesPath)
	issuer, validator := tokens.InitServer(signingKey, issuerDomain)

	// init endpoints
	app.Init(services, templates)
	api.Init(issuer, validator, services, dbPath)

	// config and serve router
	r := mux.NewRouter()
	r.HandleFunc("/", app.Home)
	r.HandleFunc("/login", app.Login)

	// api subrouter
	apiRouter := r.PathPrefix("/api").Subrouter()
	api.BuildRouter(apiRouter)

	err = http.ListenAndServe(port, r)
	if err != nil {
		log.Fatal(err)
	}
}

func readEnvVar(name string) string {
	var present bool
	str, present := os.LookupEnv(name)
	if !present {
		log.Fatalf("missing required env var '%s'\n", name)
	}
	return str
}

func loadCredential(name string, credsDir string) []byte {
	credPath := filepath.Join(credsDir, name)
	cred, err := os.ReadFile(credPath)
	if err != nil {
		log.Fatalf("failed to load required credential '%s': %v\n", name, err)
	}
	return cred
}
