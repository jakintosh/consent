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

	_, err := env.Service.CreateUser("alice", "securepassword", []string{"bad role"})
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCreateUser_UnknownRole(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.CreateUser("alice", "securepassword", []string{"nonexistent"})
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
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

	created, err := env.Service.CreateUser("alice", "securepassword", []string{"ops"})
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
	_, err = env.Service.UpdateUser(created.Subject, &service.UserUpdate{Roles: &roles})
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestUpdateUser_CannotRemoveLastAdmin(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	created, err := env.Service.CreateUser("alice", "securepassword", []string{"admin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	roles := []string{}
	_, err = env.Service.UpdateUser(created.Subject, &service.UserUpdate{Roles: &roles})
	if !errors.Is(err, service.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}

func TestDeleteUser_CannotDeleteLastAdmin(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	created, err := env.Service.CreateUser("alice", "securepassword", []string{"admin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	err = env.Service.DeleteUser(created.Subject)
	if !errors.Is(err, service.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
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

func TestCreateUser_NoRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.CreateUser("alice", "securepassword", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
}

func TestCreateUser_ThenLogin(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.CreateUser("alice", "securepassword", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	redirectURL, err := env.Service.Login("alice", "securepassword", service.InternalIntegrationName)
	if err != nil {
		t.Errorf("user cannot login: %v", err)
	}
	if redirectURL == nil {
		t.Error("expected redirect URL")
	}
}

func TestCreateUser_DuplicateHandle(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, _ = env.Service.CreateUser("alice", "password1", nil)

	_, err := env.Service.CreateUser("alice", "password2", nil)
	if err == nil {
		t.Error("expected error for duplicate handle")
	}
	if !errors.Is(err, service.ErrHandleExists) {
		t.Errorf("expected ErrHandleExists, got %v", err)
	}
}

func TestCreateUser_PasswordHashed(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	password := "mypassword"

	_, _ = env.Service.CreateUser("alice", password, nil)

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

func TestCreateUser_MultipleUsers(t *testing.T) {
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

	for _, u := range users {
		if _, err := env.Service.CreateUser(u.handle, u.password, nil); err != nil {
			t.Fatalf("CreateUser %s failed: %v", u.handle, err)
		}
	}

	for _, u := range users {
		_, err := env.Service.Login(u.handle, u.password, service.InternalIntegrationName)
		if err != nil {
			t.Errorf("Login %s failed: %v", u.handle, err)
		}
	}
}

func TestUserHasRole_Exists(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "ops", "Operations")

	user, err := env.Service.CreateUser("alice", "securepassword", []string{"admin", "ops"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if !env.Service.UserHasRole(user.Subject, "admin") {
		t.Errorf("expected UserHasRole to be true for admin")
	}
	if !env.Service.UserHasRole(user.Subject, "ops") {
		t.Errorf("expected UserHasRole to be true for ops")
	}
	if env.Service.UserHasRole(user.Subject, "billing") {
		t.Errorf("expected UserHasRole to be false for billing")
	}
}

func TestUserHasRole_NilRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	user, err := env.Service.CreateUser("alice", "securepassword", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if env.Service.UserHasRole(user.Subject, "admin") {
		t.Errorf("expected UserHasRole to be false for user with nil roles")
	}
}

func TestUserHasRole_UserNotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if env.Service.UserHasRole("nonexistent-subject", "admin") {
		t.Errorf("expected UserHasRole to be false for non-existent user")
	}
}

func TestUserHasRole_EmptyInputs(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if env.Service.UserHasRole("", "admin") {
		t.Errorf("expected UserHasRole to be false for empty subject")
	}
	if env.Service.UserHasRole("some-subject", "") {
		t.Errorf("expected UserHasRole to be false for empty role")
	}
	if env.Service.UserHasRole("", "") {
		t.Errorf("expected UserHasRole to be false for both empty")
	}
}
