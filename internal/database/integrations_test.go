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

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}
}

func TestInsertIntegration_DuplicateName(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	if err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback"); err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	err := store.InsertIntegration("svc-a", "Service A2", "aud-a", "https://svc-a.test/redirect")
	if err == nil {
		t.Fatal("expected error for duplicate integration name")
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
}

func TestUpsertSystemIntegrations_MixedBatch(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	if err := store.InsertIntegration("svc-a", "Old", "old-aud", "https://old.test/callback"); err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	err := store.UpsertSystemIntegrations([]service.Integration{
		{
			Name:     "svc-a",
			Display:  "Service A",
			Audience: "aud-a",
			Redirect: "https://svc-a.test/callback",
		},
		{
			Name:     "consent",
			Display:  "Consent",
			Audience: "consent.test",
			Redirect: "https://consent.test/auth/callback",
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

	_, err = store.GetIntegration("consent")
	if err != nil {
		t.Fatalf("GetIntegration consent failed: %v", err)
	}
}

func TestGetIntegration_Exists(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
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

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}

	display := "Service A2"
	audience := "aud-b"
	redirect := "https://svc-a.test/new"
	err = store.UpdateIntegration("svc-a", &service.IntegrationUpdate{Display: &display, Audience: &audience, Redirect: &redirect})
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

func TestDeleteIntegration_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
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
		},
		{
			Name:     "svc-a",
			Display:  "Service A",
			Audience: "aud-a",
			Redirect: "https://svc-a.test/callback",
		},
	}
	for _, integration := range integrations {
		if err := store.InsertIntegration(integration.Name, integration.Display, integration.Audience, integration.Redirect); err != nil {
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
