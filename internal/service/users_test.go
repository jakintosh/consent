package service_test

import (
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestCreateUser_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	env.CreateTestRole(t, "ops", "Operations")

	user, err := env.Service.CreateUser("alice", "securepassword", []string{"admin", "ops"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Subject == "" {
		t.Fatal("expected subject")
	}
	if user.Handle != "alice" {
		t.Fatalf("handle = %s, want alice", user.Handle)
	}
	if len(user.Roles) != 2 {
		t.Fatalf("len(user.Roles) = %d, want 2", len(user.Roles))
	}
	roleSet := make(map[string]bool)
	for _, r := range user.Roles {
		roleSet[r] = true
	}
	if !roleSet["admin"] || !roleSet["ops"] {
		t.Fatalf("roles = %#v, want admin and ops", user.Roles)
	}
}

func TestCreateUser_InvalidRole(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// roles with spaces are auto-created by the database
	user, err := env.Service.CreateUser("alice", "securepassword", []string{"bad role"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user == nil {
		t.Fatal("expected user")
	}
	roleSet := make(map[string]bool)
	for _, r := range user.Roles {
		roleSet[r] = true
	}
	if !roleSet["bad role"] {
		t.Fatalf("expected 'bad role' to be auto-created, got roles: %#v", user.Roles)
	}
}

func TestCreateUser_UnknownRole(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// unknown roles are auto-created by the database
	user, err := env.Service.CreateUser("alice", "securepassword", []string{"nonexistent"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user == nil {
		t.Fatal("expected user")
	}
	roleSet := make(map[string]bool)
	for _, r := range user.Roles {
		roleSet[r] = true
	}
	if !roleSet["nonexistent"] {
		t.Fatalf("expected 'nonexistent' to be auto-created, got roles: %#v", user.Roles)
	}
}

func TestGetUser_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	created, err := env.Service.CreateUser("alice", "securepassword", []string{"admin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := env.Service.GetUser(created.Subject)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Handle != "alice" {
		t.Fatalf("handle = %s, want alice", user.Handle)
	}
	if len(user.Roles) != 1 || user.Roles[0] != "admin" {
		t.Fatalf("roles = %#v, want [admin]", user.Roles)
	}
}

func TestListUsers(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if _, err := env.Service.CreateUser("bob", "securepassword", nil); err != nil {
		t.Fatalf("CreateUser bob failed: %v", err)
	}
	if _, err := env.Service.CreateUser("alice", "securepassword", []string{"admin"}); err != nil {
		t.Fatalf("CreateUser alice failed: %v", err)
	}

	users, err := env.Service.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Handle != "alice" || users[1].Handle != "bob" {
		t.Fatalf("unexpected order: %#v", users)
	}
}

func TestUpdateUser_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	env.CreateTestRole(t, "ops", "Operations")
	env.CreateTestRole(t, "billing", "Billing")

	created, err := env.Service.CreateUser("alice", "securepassword", []string{"admin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	handle := "alice-2"
	roles := []string{"ops", "billing"}
	updated, err := env.Service.UpdateUser(created.Subject, &service.UserUpdate{Handle: &handle, Roles: &roles})
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if updated.Handle != "alice-2" {
		t.Fatalf("handle = %s, want alice-2", updated.Handle)
	}
	if len(updated.Roles) != 2 {
		t.Fatalf("len(updated.Roles) = %d, want 2", len(updated.Roles))
	}
	roleSet := make(map[string]bool)
	for _, r := range updated.Roles {
		roleSet[r] = true
	}
	if !roleSet["ops"] || !roleSet["billing"] {
		t.Fatalf("roles = %#v, want ops and billing", updated.Roles)
	}
}

func TestUpdateUser_UnknownRole(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	created, err := env.Service.CreateUser("alice", "securepassword", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	roles := []string{"nonexistent"}
	updated, err := env.Service.UpdateUser(created.Subject, &service.UserUpdate{Roles: &roles})
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated user")
	}
	roleSet := make(map[string]bool)
	for _, r := range updated.Roles {
		roleSet[r] = true
	}
	if !roleSet["nonexistent"] {
		t.Fatalf("expected 'nonexistent' to be auto-created, got roles: %#v", updated.Roles)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	created, err := env.Service.CreateUser("alice", "securepassword", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if err := env.Service.DeleteUser(created.Subject); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	_, err = env.Service.GetUser(created.Subject)
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
