package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/app"
	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/routing"
)

func main() {
	dbPath := readEnvVar("DB_PATH")
	templatePath := readEnvVar("TEMPLATE_PATH")
	port := fmt.Sprintf(":%s", readEnvVar("PORT"))

	// load credentials
	credsDir := readEnvVar("CREDENTIALS_DIRECTORY")
	signingKeyRaw := loadCredential("signing_key", credsDir)
	signingKey, err := x509.ParseECPrivateKey(signingKeyRaw)
	if err != nil {
		log.Fatalf("failed to parse ecdsa signing key from signing_key: %v\n", err)
	}

	database.Init(dbPath)
	api.Init(signingKey)
	app.Init(templatePath)

	r := routing.BuildRouter()

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

func readEnvInt(name string) int {
	v := readEnvVar(name)
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("required env var '%s' could not be parsed as integer (\"%v\")\n", name, v)
	}
	return i
}

// func decodePublicKey(bytes []byte) *ecdsa.PublicKey {
// 	parsedKey, err := x509.ParsePKIXPublicKey(bytes)
// 	if err != nil {
// 		log.Fatalf("decodePublicKey: failed to parse ecdsa verification key from PEM block\n")
// 	}

// 	ecdsaKey, ok := parsedKey.(*ecdsa.PublicKey)
// 	if !ok {
// 		log.Fatalf("decodePublicKey: failed to cast parsed key as *ecdsa.PublicKey")
// 	}

// 	return ecdsaKey
// }

func loadCredential(name string, credsDir string) []byte {
	credPath := filepath.Join(credsDir, name)
	cred, err := os.ReadFile(credPath)
	if err != nil {
		log.Fatalf("failed to load required credential '%s': %v\n", name, err)
	}
	return cred
}
