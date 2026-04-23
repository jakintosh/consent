package service_test

import (
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestCreateRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	role, err := env.Service.CreateRole("editor", "Content Editor")
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if role.Name != "editor" {
		t.Fatalf("role.Name = %s, want editor", role.Name)
	}
	if role.Display != "Content Editor" {
		t.Fatalf("role.Display = %s, want Content Editor", role.Display)
	}
}

func TestCreateRole_EmptyName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.CreateRole("", "Something")
	if !errors.Is(err, service.ErrInvalidHandle) {
		t.Fatalf("expected ErrInvalidHandle, got %v", err)
	}
}

func TestCreateRole_EmptyDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.CreateRole("editor", "")
	if !errors.Is(err, service.ErrInvalidHandle) {
		t.Fatalf("expected ErrInvalidHandle, got %v", err)
	}
}

func TestCreateRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.CreateRole("admin", "Administrator")
	if !errors.Is(err, service.ErrRoleProtected) {
		t.Fatalf("expected ErrRoleProtected, got %v", err)
	}
}

func TestCreateRole_Duplicate(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if _, err := env.Service.CreateRole("editor", "Editor"); err != nil {
		t.Fatalf("first CreateRole failed: %v", err)
	}

	_, err := env.Service.CreateRole("editor", "Another Editor")
	if !errors.Is(err, service.ErrRoleExists) {
		t.Fatalf("expected ErrRoleExists, got %v", err)
	}
}

func TestGetRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	created, err := env.Service.CreateRole("editor", "Editor")
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	role, err := env.Service.GetRole("editor")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if role.Name != created.Name {
		t.Fatalf("role.Name = %s, want %s", role.Name, created.Name)
	}
	if role.Display != created.Display {
		t.Fatalf("role.Display = %s, want %s", role.Display, created.Display)
	}
}

func TestGetRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.GetRole("nonexistent")
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestGetRole_EmptyName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.GetRole("")
	if !errors.Is(err, service.ErrInvalidHandle) {
		t.Fatalf("expected ErrInvalidHandle, got %v", err)
	}
}

func TestUpdateRole_Display(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if _, err := env.Service.CreateRole("editor", "Editor"); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	display := "Senior Editor"
	role, err := env.Service.UpdateRole("editor", &display)
	if err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}
	if role.Display != "Senior Editor" {
		t.Fatalf("role.Display = %s, want Senior Editor", role.Display)
	}
}

func TestUpdateRole_EmptyDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if _, err := env.Service.CreateRole("editor", "Editor"); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	empty := ""
	_, err := env.Service.UpdateRole("editor", &empty)
	if !errors.Is(err, service.ErrInvalidHandle) {
		t.Fatalf("expected ErrInvalidHandle, got %v", err)
	}
}

func TestUpdateRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	display := "Super Admin"
	_, err := env.Service.UpdateRole("admin", &display)
	if !errors.Is(err, service.ErrRoleProtected) {
		t.Fatalf("expected ErrRoleProtected, got %v", err)
	}
}

func TestUpdateRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	display := "Something"
	_, err := env.Service.UpdateRole("nonexistent", &display)
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestDeleteRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if _, err := env.Service.CreateRole("temp", "Temporary"); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	err := env.Service.DeleteRole("temp")
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	_, err = env.Service.GetRole("temp")
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestDeleteRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.DeleteRole("admin")
	if !errors.Is(err, service.ErrRoleProtected) {
		t.Fatalf("expected ErrRoleProtected, got %v", err)
	}
}

func TestDeleteRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.DeleteRole("nonexistent")
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestDeleteRole_InUse(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	env.CreateTestRole(t, "editor", "Editor")

	if _, err := env.Service.CreateUser("alice", "securepassword", []string{"editor"}); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	err := env.Service.DeleteRole("editor")
	if !errors.Is(err, service.ErrRoleInUse) {
		t.Fatalf("expected ErrRoleInUse, got %v", err)
	}
}

func TestDeleteRole_EmptyName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.DeleteRole("")
	if !errors.Is(err, service.ErrInvalidHandle) {
		t.Fatalf("expected ErrInvalidHandle, got %v", err)
	}
}

func TestListRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if _, err := env.Service.CreateRole("editor", "Editor"); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if _, err := env.Service.CreateRole("viewer", "Viewer"); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	roles, err := env.Service.ListRoles()
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}

	// admin is seeded, plus editor and viewer we created
	if len(roles) != 3 {
		t.Fatalf("len(roles) = %d, want 3", len(roles))
	}

	roleNames := make(map[string]bool)
	for _, r := range roles {
		roleNames[r.Name] = true
	}
	if !roleNames["admin"] || !roleNames["editor"] || !roleNames["viewer"] {
		t.Fatalf("expected admin, editor, viewer in roles, got %#v", roles)
	}
}
