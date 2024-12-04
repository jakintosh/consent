package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/routing"
)

func main() {
	dbPath := readEnvVar("DB_PATH")
	port := fmt.Sprintf(":%s", readEnvVar("PORT"))

	database.Init(dbPath)
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
