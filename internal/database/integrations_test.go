package database_test

import (
	"database/sql"
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestInsertIntegration_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}
}

func TestInsertIntegration_DuplicateName(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	if err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil); err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	err := store.InsertIntegration("svc-a", "Service A2", "aud-a", "https://svc-a.test/redirect", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	if err == nil {
		t.Fatal("expected error for duplicate integration name")
	}
}

func TestInsertIntegration_DuplicateAudience(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	if err := store.InsertIntegration("svc-a", "Service A", "shared-audience", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil); err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	err := store.InsertIntegration("svc-b", "Service B", "shared-audience", "https://svc-b.test/callback", "https://svc-b.test", "https://svc-b.test/logo.png", nil)
	if err == nil {
		t.Fatal("expected error for duplicate integration audience")
	}
}

func TestUpsertSystemIntegrations_Empty(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.UpsertSystemIntegrations(nil)
	if err != nil {
		t.Fatalf("UpsertSystemIntegrations failed: %v", err)
	}
}

func TestUpsertSystemIntegrations_Insert(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.UpsertSystemIntegrations([]service.Integration{
		{
			Name:     "consent",
			Display:  "Consent",
			Audience: "consent.test",
			Redirect: "https://consent.test/auth/callback",
			Homepage: "https://consent.test",
			Logo:     "",
		},
	})
	if err != nil {
		t.Fatalf("UpsertSystemIntegrations failed: %v", err)
	}

	record, err := store.GetIntegration("consent")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if record.Display != "Consent" {
		t.Fatalf("Display = %s, want Consent", record.Display)
	}
	if record.Homepage != "https://consent.test" {
		t.Fatalf("Homepage = %s, want https://consent.test", record.Homepage)
	}
}

func TestUpsertSystemIntegrations_MixedBatch(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	if err := store.InsertIntegration("svc-a", "Old", "old-aud", "https://old.test/callback", "https://old.test", "https://old.test/logo.png", nil); err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	err := store.UpsertSystemIntegrations([]service.Integration{
		{
			Name:     "svc-a",
			Display:  "Service A",
			Audience: "aud-a",
			Redirect: "https://svc-a.test/callback",
			Homepage: "https://svc-a.test",
			Logo:     "https://svc-a.test/logo.png",
		},
		{
			Name:     "consent",
			Display:  "Consent",
			Audience: "consent.test",
			Redirect: "https://consent.test/auth/callback",
			Homepage: "https://consent.test",
			Logo:     "",
		},
	})
	if err != nil {
		t.Fatalf("UpsertSystemIntegrations failed: %v", err)
	}

	record, err := store.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if record.Display != "Service A" {
		t.Fatalf("Display = %s, want Service A", record.Display)
	}
	if record.Homepage != "https://svc-a.test" {
		t.Fatalf("Homepage = %s, want https://svc-a.test", record.Homepage)
	}

	_, err = store.GetIntegration("consent")
	if err != nil {
		t.Fatalf("GetIntegration consent failed: %v", err)
	}
}

func TestGetIntegration_Exists(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	record, err := store.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if record.Name != "svc-a" {
		t.Errorf("Name = %s, want svc-a", record.Name)
	}
	if record.Display != "Service A" {
		t.Errorf("Display = %s, want Service A", record.Display)
	}
	if record.Audience != "aud-a" {
		t.Errorf("Audience = %s, want aud-a", record.Audience)
	}
	if record.Redirect != "https://svc-a.test/callback" {
		t.Errorf("Redirect = %s, want https://svc-a.test/callback", record.Redirect)
	}
	if record.Homepage != "https://svc-a.test" {
		t.Errorf("Homepage = %s, want https://svc-a.test", record.Homepage)
	}
	if record.Logo != "https://svc-a.test/logo.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo.png", record.Logo)
	}
	if record.RequiredRoles == nil {
		t.Error("RequiredRoles should be an empty slice, got nil")
	}
}

func TestGetIntegration_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	_, err := store.GetIntegration("missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateIntegration_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	display := "Service A2"
	audience := "aud-b"
	redirect := "https://svc-a.test/new"
	homepage := "https://svc-a-v2.test"
	logo := "https://svc-a.test/logo-v2.png"
	err = store.UpdateIntegration("svc-a", &service.IntegrationUpdate{Display: &display, Audience: &audience, Redirect: &redirect, Homepage: &homepage, Logo: &logo})
	if err != nil {
		t.Fatalf("UpdateIntegration failed: %v", err)
	}

	record, err := store.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if record.Display != "Service A2" {
		t.Errorf("Display = %s, want Service A2", record.Display)
	}
	if record.Audience != "aud-b" {
		t.Errorf("Audience = %s, want aud-b", record.Audience)
	}
	if record.Redirect != "https://svc-a.test/new" {
		t.Errorf("Redirect = %s, want https://svc-a.test/new", record.Redirect)
	}
	if record.Homepage != "https://svc-a-v2.test" {
		t.Errorf("Homepage = %s, want https://svc-a-v2.test", record.Homepage)
	}
	if record.Logo != "https://svc-a.test/logo-v2.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo-v2.png", record.Logo)
	}
}

func TestUpdateIntegration_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	display := "Service"
	err := store.UpdateIntegration("missing", &service.IntegrationUpdate{Display: &display})
	if err == nil {
		t.Fatal("expected error for missing integration")
	}
}

func TestUpdateIntegration_RequiredRolesNotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	roles := []string{"viewer"}
	err := store.UpdateIntegration("missing", &service.IntegrationUpdate{RequiredRoles: &roles})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateIntegration_DuplicateAudience(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	if err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil); err != nil {
		t.Fatalf("InsertIntegration svc-a failed: %v", err)
	}
	if err := store.InsertIntegration("svc-b", "Service B", "aud-b", "https://svc-b.test/callback", "https://svc-b.test", "https://svc-b.test/logo.png", nil); err != nil {
		t.Fatalf("InsertIntegration svc-b failed: %v", err)
	}

	audience := "aud-a"
	err := store.UpdateIntegration("svc-b", &service.IntegrationUpdate{Audience: &audience})
	if err == nil {
		t.Fatal("expected error for duplicate integration audience")
	}
}

func TestUpdateIntegration_HomepageLogo(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	homepage := "https://svc-a-v2.test"
	logo := "https://svc-a.test/logo-v2.png"
	err = store.UpdateIntegration("svc-a", &service.IntegrationUpdate{Homepage: &homepage, Logo: &logo})
	if err != nil {
		t.Fatalf("UpdateIntegration failed: %v", err)
	}

	record, err := store.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if record.Homepage != "https://svc-a-v2.test" {
		t.Errorf("Homepage = %s, want https://svc-a-v2.test", record.Homepage)
	}
	if record.Logo != "https://svc-a.test/logo-v2.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo-v2.png", record.Logo)
	}
}

func TestDeleteIntegration_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	deleted, err := store.DeleteIntegration("svc-a")
	if err != nil {
		t.Fatalf("DeleteIntegration failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected integration to be deleted")
	}

	_, err = store.GetIntegration("svc-a")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteIntegration_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	deleted, err := store.DeleteIntegration("missing")
	if err != nil {
		t.Fatalf("DeleteIntegration failed: %v", err)
	}
	if deleted {
		t.Fatal("expected delete to report false")
	}
}

func TestListIntegrations_ReturnsHomepageAndLogo(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	records, err := store.ListIntegrations()
	if err != nil {
		t.Fatalf("ListIntegrations failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(records))
	}
	if records[0].Homepage != "https://svc-a.test" {
		t.Errorf("Homepage = %s, want https://svc-a.test", records[0].Homepage)
	}
	if records[0].Logo != "https://svc-a.test/logo.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo.png", records[0].Logo)
	}
}

func TestListIntegrations_Empty(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	records, err := store.ListIntegrations()
	if err != nil {
		t.Fatalf("ListIntegrations failed: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 integrations, got %d", len(records))
	}
}

func TestListIntegrations_Multiple(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	integrations := []service.Integration{
		{
			Name:     "svc-b",
			Display:  "Service B",
			Audience: "aud-b",
			Redirect: "https://svc-b.test/callback",
			Homepage: "https://svc-b.test",
			Logo:     "https://svc-b.test/logo.png",
		},
		{
			Name:     "svc-a",
			Display:  "Service A",
			Audience: "aud-a",
			Redirect: "https://svc-a.test/callback",
			Homepage: "https://svc-a.test",
			Logo:     "https://svc-a.test/logo.png",
		},
	}
	for _, integration := range integrations {
		if err := store.InsertIntegration(integration.Name, integration.Display, integration.Audience, integration.Redirect, integration.Homepage, integration.Logo, nil); err != nil {
			t.Fatalf("InsertIntegration failed: %v", err)
		}
	}

	records, err := store.ListIntegrations()
	if err != nil {
		t.Fatalf("ListIntegrations failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 integrations, got %d", len(records))
	}
	if records[0].Name != "svc-a" {
		t.Errorf("expected svc-a first, got %s", records[0].Name)
	}
	if records[1].Name != "svc-b" {
		t.Errorf("expected svc-b second, got %s", records[1].Name)
	}
}

func TestIntegrations_RequiredRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("superadmin", "Super Admin")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}
	err = store.InsertRole("editor", "Editor")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertIntegration(
		"role-app",
		"Role App",
		"role-audience",
		"https://role.test/callback",
		"https://role.test",
		"https://role.test/logo.png",
		[]string{"superadmin", "editor"},
	)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	got, err := store.GetIntegration("role-app")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if got.Name != "role-app" {
		t.Errorf("Name = %s, want role-app", got.Name)
	}
	if len(got.RequiredRoles) != 2 {
		t.Fatalf("RequiredRoles length = %d, want 2", len(got.RequiredRoles))
	}
	if got.RequiredRoles[0] != "editor" || got.RequiredRoles[1] != "superadmin" {
		t.Errorf("RequiredRoles = %v, want [editor superadmin]", got.RequiredRoles)
	}
}

func TestIntegrations_UpdateRequiredRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("viewer", "Viewer")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertIntegration(
		"role-app",
		"Role App",
		"role-audience",
		"https://role.test/callback",
		"https://role.test",
		"https://role.test/logo.png",
		nil,
	)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	roles := []string{"viewer"}
	err = store.UpdateIntegration("role-app", &service.IntegrationUpdate{RequiredRoles: &roles})
	if err != nil {
		t.Fatalf("UpdateIntegration failed: %v", err)
	}

	got, err := store.GetIntegration("role-app")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if len(got.RequiredRoles) != 1 || got.RequiredRoles[0] != "viewer" {
		t.Errorf("RequiredRoles = %v, want [viewer]", got.RequiredRoles)
	}
}

func TestIntegrations_ClearRequiredRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("superadmin", "Super Admin")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertIntegration(
		"role-app",
		"Role App",
		"role-audience",
		"https://role.test/callback",
		"https://role.test",
		"https://role.test/logo.png",
		[]string{"superadmin"},
	)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	got, err := store.GetIntegration("role-app")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if len(got.RequiredRoles) != 1 || got.RequiredRoles[0] != "superadmin" {
		t.Fatalf("RequiredRoles = %v, want [superadmin]", got.RequiredRoles)
	}

	var emptyRoles []string
	err = store.UpdateIntegration("role-app", &service.IntegrationUpdate{RequiredRoles: &emptyRoles})
	if err != nil {
		t.Fatalf("UpdateIntegration failed: %v", err)
	}

	got, err = store.GetIntegration("role-app")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if len(got.RequiredRoles) != 0 {
		t.Errorf("RequiredRoles = %v, want []", got.RequiredRoles)
	}
}

func TestIntegrations_ListRequiredRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertRole("superadmin", "Super Admin")
	if err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err = store.InsertIntegration(
		"role-app",
		"Role App",
		"role-audience",
		"https://role.test/callback",
		"https://role.test",
		"https://role.test/logo.png",
		[]string{"superadmin"},
	)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	records, err := store.ListIntegrations()
	if err != nil {
		t.Fatalf("ListIntegrations failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(records))
	}
	if len(records[0].RequiredRoles) != 1 || records[0].RequiredRoles[0] != "superadmin" {
		t.Errorf("RequiredRoles = %v, want [superadmin]", records[0].RequiredRoles)
	}
}

func TestGetIntegrationByAudience_RequiredRoles(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	if err := store.InsertRole("editor", "Editor"); err != nil {
		t.Fatalf("InsertRole failed: %v", err)
	}

	err := store.InsertIntegration(
		"role-app",
		"Role App",
		"role-audience",
		"https://role.test/callback",
		"https://role.test",
		"https://role.test/logo.png",
		[]string{"editor"},
	)
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	got, err := store.GetIntegrationByAudience("role-audience")
	if err != nil {
		t.Fatalf("GetIntegrationByAudience failed: %v", err)
	}
	if got.Name != "role-app" {
		t.Fatalf("Name = %s, want role-app", got.Name)
	}
	if len(got.RequiredRoles) != 1 || got.RequiredRoles[0] != "editor" {
		t.Fatalf("RequiredRoles = %v, want [editor]", got.RequiredRoles)
	}
}
