package api_test

import (
	"net/http"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// valid registration succeeds
	body := `{
		"username": "newuser",
		"password": "securepass"
	}`
	result := wire.TestPost[any](env.Router, "/register", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestRegister_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed JSON returns 400
	result := wire.TestPost[any](env.Router, "/register", "not-json", jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestRegister_DuplicateUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// first registration succeeds
	body := `{
		"username": "alice",
		"password": "pass1"
	}`
	wire.TestPost[any](env.Router, "/register", body, jsonHeader)

	// second registration with same username returns 409
	body2 := `{
		"username": "alice",
		"password": "pass2"
	}`
	result := wire.TestPost[any](env.Router, "/register", body2, jsonHeader)
	result.ExpectStatus(t, http.StatusConflict)
	result.ExpectError(t)
}

func TestRegister_ThenLogin(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// register new user
	regBody := `{
		"username": "newuser",
		"password": "mypassword"
	}`
	result := wire.TestPost[any](env.Router, "/register", regBody, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	// login with registered credentials succeeds
	loginBody := `{
		"handle": "newuser",
		"secret": "mypassword",
		"service": "test-service"
	}`
	loginResult := wire.TestPost[any](env.Router, "/login", loginBody, jsonHeader)
	loginResult.ExpectStatus(t, http.StatusSeeOther)
	location := loginResult.Headers.Get("Location")
	if location == "" {
		t.Fatal("expected Location header in redirect")
	}
	if !strings.Contains(location, "auth_code=") {
		t.Errorf("login after register should work, got redirect: %s", location)
	}
}

func TestRegister_EmptyBody(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// empty JSON body returns 400
	result := wire.TestPost[any](env.Router, "/register", "{}", jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestRegister_MultipleUsers(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// multiple unique users can register
	users := []string{"alice", "bob", "charlie"}
	for _, user := range users {
		body := `{
			"username": "` + user + `",
			"password": "password"
		}`
		result := wire.TestPost[any](env.Router, "/register", body, jsonHeader)
		result.ExpectStatus(t, http.StatusOK)
	}
}
