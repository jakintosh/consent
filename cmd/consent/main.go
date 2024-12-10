package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/routing"
)

var privateKey ecdsa.PrivateKey

func main() {
	dbPath := readEnvVar("DB_PATH")
	port := fmt.Sprintf(":%s", readEnvVar("PORT"))

	signingKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("couldn't generate required ECDSA private key")
	}

	database.Init(dbPath)
	api.Init(signingKey)
	r := routing.BuildRouter()

	log.Fatal(http.ListenAndServe(port, r))
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
