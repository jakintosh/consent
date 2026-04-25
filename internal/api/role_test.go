package api_test

import (
	"errors"
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestAPICreateRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"editor",
		"display":"Content Editor"
	}`
	result := wire.TestPost[api.Role](env.Router, "/admin/roles", body, jsonHeader, authHeader)
	response := result.ExpectOK(t)
	if response.Name != "editor" {
		t.Fatalf("role.Name = %s, want editor", response.Name)
	}
	if response.Display != "Content Editor" {
		t.Fatalf("role.Display = %s, want Content Editor", response.Display)
	}
}

func TestAPICreateRole_EmptyName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"",
		"display":"Something"
	}`
	result := wire.TestPost[any](env.Router, "/admin/roles", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateRole_EmptyDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"editor",
		"display":""
	}`
	result := wire.TestPost[any](env.Router, "/admin/roles", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"admin",
		"display":"Administrator"
	}`
	result := wire.TestPost[any](env.Router, "/admin/roles", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusForbidden)
}

func TestAPICreateRole_Duplicate(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"editor",
		"display":"Editor"
	}`
	wire.TestPost[any](env.Router, "/admin/roles", body, jsonHeader, authHeader)

	body2 := `{
		"name":"editor",
		"display":"Another"
	}`
	result := wire.TestPost[any](env.Router, "/admin/roles", body2, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusConflict)
}

func TestAPIGetRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")

	result := wire.TestGet[api.Role](env.Router, "/admin/roles/editor", authHeader)
	response := result.ExpectOK(t)
	if response.Name != "editor" {
		t.Fatalf("role.Name = %s, want editor", response.Name)
	}
	if response.Display != "Editor" {
		t.Fatalf("role.Display = %s, want Editor", response.Display)
	}
}

func TestAPIGetRole_AdminSeeded(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[api.Role](env.Router, "/admin/roles/admin", authHeader)
	response := result.ExpectOK(t)
	if response.Name != "admin" {
		t.Fatalf("role.Name = %s, want admin", response.Name)
	}
}

func TestAPIGetRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[any](env.Router, "/admin/roles/nonexistent", authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIGetRole_MissingName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[any](env.Router, "/admin/roles/", authHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestAPIUpdateRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")

	body := `{
		"display":"Senior Editor"
	}`
	result := wire.TestPut[api.Role](env.Router, "/admin/roles/editor", body, jsonHeader, authHeader)
	response := result.ExpectOK(t)
	if response.Display != "Senior Editor" {
		t.Fatalf("role.Display = %s, want Senior Editor", response.Display)
	}
}

func TestAPIUpdateRole_EmptyDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")

	body := `{
		"display":""
	}`
	result := wire.TestPut[any](env.Router, "/admin/roles/editor", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIUpdateRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"display":"Super Admin"
	}`
	result := wire.TestPut[any](env.Router, "/admin/roles/admin", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusForbidden)
}

func TestAPIUpdateRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"display":"Something"
	}`
	result := wire.TestPut[any](env.Router, "/admin/roles/nonexistent", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIDeleteRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "temp", "Temporary")

	result := wire.TestDelete[any](env.Router, "/admin/roles/temp", authHeader)
	result.ExpectStatus(t, http.StatusOK)

	_, err := env.Service.GetRole("temp")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestAPIDeleteRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/roles/admin", authHeader)
	result.ExpectStatusError(t, http.StatusForbidden)
}

func TestAPIDeleteRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/roles/nonexistent", authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIDeleteRole_InUse(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	env.CreateTestRole(t, "editor", "Editor")
	env.RegisterTestUser(t, "alice", "password")

	// Create a user with this role
	_, err := env.Service.CreateUser("bob", "password2", []string{"editor"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Role delete succeeds via FK cascade, user's roles are cleaned up
	result := wire.TestDelete[any](env.Router, "/admin/roles/editor", authHeader)
	result.ExpectStatus(t, http.StatusOK)

	// Verify role is gone
	_, err = env.Service.GetRole("editor")
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected role to be deleted, got: %v", err)
	}
}

func TestAPIListRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestRole(t, "viewer", "Viewer")

	result := wire.TestGet[[]api.Role](env.Router, "/admin/roles", authHeader)
	response := result.ExpectOK(t)
	if len(response) != 3 {
		t.Fatalf("len(response) = %d, want 3 (admin seeded + editor + viewer)", len(response))
	}
	roleNames := make(map[string]bool)
	for _, r := range response {
		roleNames[r.Name] = true
	}
	if !roleNames["admin"] || !roleNames["editor"] || !roleNames["viewer"] {
		t.Fatalf("expected admin, editor, viewer in roles, got %#v", response)
	}
}
