package api_test

import (
	"net/http"
	"strings"
	"testing"

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
	result := testutil.PostJSON(env.Router, "/register", body, nil)
	testutil.ExpectStatus(t, http.StatusOK, result)
}

func TestRegister_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed JSON returns 400
	result := testutil.PostJSON(env.Router, "/register", "not-json", nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestRegister_DuplicateUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// first registration succeeds
	body := `{
		"username": "alice",
		"password": "pass1"
	}`
	testutil.PostJSON(env.Router, "/register", body, nil)

	// second registration with same username returns 409
	body2 := `{
		"username": "alice",
		"password": "pass2"
	}`
	result := testutil.PostJSON(env.Router, "/register", body2, nil)
	testutil.ExpectStatus(t, http.StatusConflict, result)
}

func TestRegister_ThenLogin(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// register new user
	regBody := `{
		"username": "newuser",
		"password": "mypassword"
	}`
	result := testutil.PostJSON(env.Router, "/register", regBody, nil)
	testutil.ExpectStatus(t, http.StatusOK, result)

	// login with registered credentials succeeds
	loginBody := `{
		"handle": "newuser",
		"secret": "mypassword",
		"service": "test-service"
	}`
	result = testutil.PostJSON(env.Router, "/login", loginBody, nil)
	location := testutil.ExpectRedirect(t, result)
	if !strings.Contains(location, "auth_code=") {
		t.Errorf("login after register should work, got redirect: %s", location)
	}
}

func TestRegister_EmptyBody(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// empty JSON body returns 400
	result := testutil.PostJSON(env.Router, "/register", "{}", nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
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
		result := testutil.PostJSON(env.Router, "/register", body, nil)
		testutil.ExpectStatus(t, http.StatusOK, result)
	}
}
