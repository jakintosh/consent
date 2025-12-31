package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/command-go/pkg/version"
	"git.sr.ht/~jakintosh/consent/pkg/client"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

var root = &args.Command{
	Name: "client",
	Help: "Demo OAuth client application",
	Config: &args.Config{
		Author: "jakintosh",
		HelpOption: &args.HelpOption{
			Short: 'h',
			Long:  "help",
		},
	},
	Options: []args.Option{
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

		// Read configuration
		verbose := i.GetFlag("verbose")

		if verbose {
			log.Println("Starting demo OAuth client...")
		}

		// read "env vars" (hardcoded for demo)
		authUrl := "http://localhost:9001"
		issuerDomain := "auth.studiopollinator.com"
		audience := "localhost:10000"

		// load credentials
		verificationKeyBytes := loadCredential("verification_key.der", "./etc/secrets/")
		verificationKey := decodePublicKey(verificationKeyBytes)

		// create token client
		validator := tokens.InitClient(verificationKey, issuerDomain, audience)

		// init consent.client
		c := client.Init(validator, authUrl)

		// config router
		http.HandleFunc("/", homeHandler(c))
		http.HandleFunc("/api/example", exampleHandler(c))
		http.HandleFunc("/api/authorize", c.HandleAuthorizationCode())

		if verbose {
			log.Println("Listening on :10000")
		}

		err := http.ListenAndServe(":10000", nil)
		if err != nil {
			return fmt.Errorf("server error: %v", err)
		}

		return nil
	},
}

func main() {
	root.Parse()
}

func homeHandler(c *client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessToken, csrf, err := c.VerifyAuthorizationGetCSRF(w, r)
		if err != nil {
			log.Printf("%s: failed to verify authorization: %v", r.RequestURI, err)
		}

		if accessToken != nil {
			w.Write(fmt.Appendf(nil, homeAuth, csrf))
		} else {
			w.Write([]byte(homeUnauth))
		}
	}
}

func exampleHandler(c *client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		csrf := r.URL.Query().Get("csrf")
		accessToken, csrf, err := c.VerifyAuthorizationCheckCSRF(w, r, csrf)
		if err != nil {
			log.Printf("%s: failed to verify authorization: %v", r.RequestURI, err)
		}

		if accessToken != nil {
			w.Write(fmt.Appendf(nil, exampleAuth, accessToken.Subject(), csrf))
		} else {
			w.Write([]byte(exampleUnauth))
		}
	}
}

func decodePublicKey(bytes []byte) *ecdsa.PublicKey {
	parsedKey, err := x509.ParsePKIXPublicKey(bytes)
	if err != nil {
		log.Fatalf("decodePublicKey: failed to parse ecdsa verification key from DER")
	}

	ecdsaKey, ok := parsedKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatalf("decodePublicKey: failed to cast parsed key as *ecdsa.PublicKey")
	}

	return ecdsaKey
}

func loadCredential(name string, credsDir string) []byte {
	credPath := filepath.Join(credsDir, name)
	cred, err := os.ReadFile(credPath)
	if err != nil {
		log.Fatalf("failed to load required credential '%s': %v", name, err)
	}
	return cred
}

const homeAuth string = `<!DOCTYPE html>
<html>
<body>
<a href="http://localhost:10000/api/example?csrf=%s">Example API Call</a>
</body>
</html>`

const homeUnauth string = `<!DOCTYPE html>
<html>
<body>
<a href="http://localhost:9001/login?service=example@localhost">Log In with Pollinator</a>
</body>
</html>`

const exampleAuth string = `<!DOCTYPE html>
<html>
<body>
<p>Secret logged in page for %s!</p>
<form>
	<input hidden value="%s"/>
</form>
</body>
</html>`

const exampleUnauth string = `<!DOCTYPE html>
<html>
<body>
<p>You are not logged in.</p>
</body>
</html>`
