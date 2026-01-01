package service_test

import (
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// registering a new user succeeds
	err := env.Service.Register("alice", "securepassword")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
}

func TestRegister_ThenLogin(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// register new user
	err := env.Service.Register("alice", "securepassword")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// registered user can login
	redirectURL, err := env.Service.Login("alice", "securepassword", "test-service")
	if err != nil {
		t.Errorf("registered user cannot login: %v", err)
	}
	if redirectURL == nil {
		t.Error("expected redirect URL")
	}
}

func TestRegister_DuplicateHandle(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// first registration succeeds
	_ = env.Service.Register("alice", "password1")

	// duplicate registration returns ErrHandleExists
	err := env.Service.Register("alice", "password2")
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
	if !errors.Is(err, service.ErrHandleExists) {
		t.Errorf("expected ErrHandleExists, got %v", err)
	}
}

func TestRegister_HashesPassword(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	password := "mypassword"

	// register user
	_ = env.Service.Register("alice", password)

	// verify password is hashed in database
	secret, err := env.DB.GetSecret("alice")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(secret) == password {
		t.Error("password stored in plain text")
	}
	if len(secret) < 50 {
		t.Errorf("hash seems too short: %d bytes", len(secret))
	}
}

func TestRegister_MultipleUsers(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	users := []struct {
		handle   string
		password string
	}{
		{"alice", "password-a"},
		{"bob", "password-b"},
		{"charlie", "password-c"},
	}

	// register multiple users
	for _, u := range users {
		if err := env.Service.Register(u.handle, u.password); err != nil {
			t.Fatalf("Register %s failed: %v", u.handle, err)
		}
	}

	// all registered users can login
	for _, u := range users {
		_, err := env.Service.Login(u.handle, u.password, "test-service")
		if err != nil {
			t.Errorf("Login %s failed: %v", u.handle, err)
		}
	}
}
