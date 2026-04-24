package database_test

import (
	"database/sql"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestInsertRole_RoundTrip(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("editor", "Content Editor")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	role, err := store.GetRole("editor")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if role.Name != "editor" {
		t.Fatalf("role.Name = %s, want editor", role.Name)
	}
	if role.Display != "Content Editor" {
		t.Fatalf("role.Display = %s, want Content Editor", role.Display)
	}
}

func TestListRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("viewer", "Viewer")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}
	err = store.InsertRole("admin", "Administrator")
	if err != nil {
		t.Fatalf("InsertRole failed: %s", err)
	}
	err = store.InsertRole("editor", "Editor")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	roles, err := store.ListRoles()
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) != 3 {
		t.Fatalf("len(roles) = %d, want 3", len(roles))
	}
	if roles[0].Name != "admin" || roles[1].Name != "editor" || roles[2].Name != "viewer" {
		t.Fatalf("roles names = %#v, want [admin editor viewer]", roles)
	}
}

func TestUpdateRoleDisplay(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("editor", "Editor")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	display := "Senior Editor"
	err = store.UpdateRole("editor", &service.RoleUpdate{Display: &display})
	if err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}

	role, err := store.GetRole("editor")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if role.Display != "Senior Editor" {
		t.Fatalf("role.Display = %s, want Senior Editor", role.Display)
	}
}

func TestUpdateRoleDisplay_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	display := "Something"
	err := store.UpdateRole("nonexistent", &service.RoleUpdate{Display: &display})
	if err == nil {
		t.Fatal("expected error for nonexistent role")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteRole(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("deleteme", "Delete Me")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	deleted, err := store.DeleteRole("deleteme")
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted = true")
	}

	_, err = store.GetRole("deleteme")
	if err == nil {
		t.Fatal("expected GetRole to fail after deletion")
	}
}

func TestDeleteRole_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	deleted, err := store.DeleteRole("nonexistent")
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
	if deleted {
		t.Fatal("expected deleted = false")
	}
}

func TestInsertUser_RolesRoundTrip(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("admin", "Administrator")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}
	err = store.InsertRole("ops", "Operations")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertUser("subject-alice", "alice", []byte("secret"), []string{"admin", "ops"})
	if err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}

	user, err := store.GetUserBySubject("subject-alice")
	if err != nil {
		t.Fatalf("GetUserBySubject failed: %v", err)
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

func TestInsertUser_AutoCreatesRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertUser("subject-alice", "alice", []byte("secret"), []string{"auto-role"})
	if err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}

	role, err := store.GetRole("auto-role")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if role.Name != "auto-role" {
		t.Fatalf("role.Name = %s, want auto-role", role.Name)
	}
	if role.Display != "auto-role" {
		t.Fatalf("role.Display = %s, want auto-role (defaulted to name)", role.Display)
	}
}

func TestUpdateUser_RolesRoundTrip(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("admin", "Administrator")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}
	err = store.InsertRole("billing", "Billing")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertUser("subject-alice", "alice", []byte("secret"), nil)
	if err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}

	err = store.UpdateUser("subject-alice", "alice-2", []string{"admin", "billing"})
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	user, err := store.GetUserBySubject("subject-alice")
	if err != nil {
		t.Fatalf("GetUserBySubject failed: %v", err)
	}
	if user.Handle != "alice-2" {
		t.Fatalf("handle = %s, want alice-2", user.Handle)
	}
	if len(user.Roles) != 2 {
		t.Fatalf("len(user.Roles) = %d, want 2", len(user.Roles))
	}
	roleSet := make(map[string]bool)
	for _, r := range user.Roles {
		roleSet[r] = true
	}
	if !roleSet["admin"] || !roleSet["billing"] {
		t.Fatalf("roles = %#v, want admin and billing", user.Roles)
	}
}

func TestListUsers_Roles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("admin", "Administrator")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}
	err = store.InsertRole("viewer", "Viewer")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertUser("subject-alice", "alice", []byte("secret"), []string{"admin"})
	if err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}
	err = store.InsertUser("subject-bob", "bob", []byte("secret"), []string{"viewer", "admin"})
	if err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}

	users, err := store.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}

	alice := users[0]
	if alice.Subject != "subject-alice" {
		t.Fatalf("users[0].Subject = %s, want subject-alice", alice.Subject)
	}
	if len(alice.Roles) != 1 || alice.Roles[0] != "admin" {
		t.Fatalf("alice.Roles = %#v, want [admin]", alice.Roles)
	}

	bob := users[1]
	if bob.Subject != "subject-bob" {
		t.Fatalf("users[1].Subject = %s, want subject-bob", bob.Subject)
	}
	if len(bob.Roles) != 2 {
		t.Fatalf("len(bob.Roles) = %d, want 2", len(bob.Roles))
	}
}

func TestDeleteUser_CascadesUserRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("admin", "Administrator")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertUser("subject-alice", "alice", []byte("secret"), []string{"admin"})
	if err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}

	deleted, err := store.DeleteUser("subject-alice")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted = true")
	}

	_, err = store.GetUserBySubject("subject-alice")
	if err == nil {
		t.Fatal("expected GetUserBySubject to fail after deletion")
	}

	roles, err := store.ListRoles()
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) != 1 || roles[0].Name != "admin" {
		t.Fatalf("expected admin role to still exist after user deletion via cascade, got %#v", roles)
	}
}

func TestDeleteRole_CascadesUserRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("admin", "Administrator")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertUser("subject-alice", "alice", []byte("secret"), []string{"admin"})
	if err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}

	deleted, err := store.DeleteRole("admin")
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted = true")
	}

	user, err := store.GetUserBySubject("subject-alice")
	if err != nil {
		t.Fatalf("GetUserBySubject failed: %v", err)
	}
	if len(user.Roles) != 0 {
		t.Fatalf("user.Roles = %#v, want []", user.Roles)
	}
}
