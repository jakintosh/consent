package database_test

import (
	"database/sql"
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestInsertService_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertService("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if err != nil {
		t.Fatalf("InsertService failed: %v", err)
	}
}

func TestInsertService_DuplicateName(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	if err := store.InsertService("svc-a", "Service A", "aud-a", "https://svc-a.test/callback"); err != nil {
		t.Fatalf("InsertService failed: %v", err)
	}

	err := store.InsertService("svc-a", "Service A2", "aud-a", "https://svc-a.test/redirect")
	if err == nil {
		t.Fatal("expected error for duplicate service name")
	}
}

func TestGetService_Exists(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertService("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if err != nil {
		t.Fatalf("InsertService failed: %v", err)
	}

	record, err := store.GetService("svc-a")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
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

func TestGetService_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	_, err := store.GetService("missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateService_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertService("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if err != nil {
		t.Fatalf("InsertService failed: %v", err)
	}

	err = store.UpdateService("svc-a", "Service A2", "aud-b", "https://svc-a.test/new")
	if err != nil {
		t.Fatalf("UpdateService failed: %v", err)
	}

	record, err := store.GetService("svc-a")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
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

func TestUpdateService_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.UpdateService("missing", "Service", "aud", "https://example.com")
	if err == nil {
		t.Fatal("expected error for missing service")
	}
}

func TestDeleteService_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertService("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if err != nil {
		t.Fatalf("InsertService failed: %v", err)
	}

	deleted, err := store.DeleteService("svc-a")
	if err != nil {
		t.Fatalf("DeleteService failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected service to be deleted")
	}

	_, err = store.GetService("svc-a")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteService_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	deleted, err := store.DeleteService("missing")
	if err != nil {
		t.Fatalf("DeleteService failed: %v", err)
	}
	if deleted {
		t.Fatal("expected delete to report false")
	}
}

func TestListServices_Empty(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	records, err := store.ListServices()
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 services, got %d", len(records))
	}
}

func TestListServices_Multiple(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	services := []service.ServiceDefinition{
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
	for _, svc := range services {
		if err := store.InsertService(svc.Name, svc.Display, svc.Audience, svc.Redirect); err != nil {
			t.Fatalf("InsertService failed: %v", err)
		}
	}

	records, err := store.ListServices()
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 services, got %d", len(records))
	}
	if records[0].Name != "svc-a" {
		t.Errorf("expected svc-a first, got %s", records[0].Name)
	}
	if records[1].Name != "svc-b" {
		t.Errorf("expected svc-b second, got %s", records[1].Name)
	}
}
