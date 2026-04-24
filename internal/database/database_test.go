package database_test

import (
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestOpen_InMemory(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// in-memory store is created successfully
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestOpen_CreatesSchema(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// schema is created - insert and retrieve works
	err := store.InsertUser("subject-test-user", "test-user", []byte("secret-hash"), nil)
	if err != nil {
		t.Fatalf("schema not created - InsertIdentity failed: %v", err)
	}

	secret, err := store.GetSecret("test-user")
	if err != nil {
		t.Fatalf("schema not created - GetSecret failed: %v", err)
	}
	if string(secret) != "secret-hash" {
		t.Errorf("unexpected secret: %s", string(secret))
	}
}

func TestOpen_Close(t *testing.T) {
	t.Parallel()
	// Close is called explicitly to validate success without test cleanup.
	dbOpts := database.Options{
		Path: ":memory:",
	}
	store, err := database.Open(dbOpts)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	// closing store succeeds without error
	if err := store.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestOpen_ExistingDatabaseRunsMigrationsOnce(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/consent.sqlite"

	first, err := database.Open(database.Options{Path: path})
	if err != nil {
		t.Fatalf("first database.Open failed: %v", err)
	}
	if err := first.InsertUser("subject-alice", "alice", []byte("secret"), nil); err != nil {
		t.Fatalf("InsertIdentity failed: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}

	second, err := database.Open(database.Options{Path: path})
	if err != nil {
		t.Fatalf("second database.Open failed: %v", err)
	}
	defer second.Close()

	secret, err := second.GetSecret("alice")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(secret) != "secret" {
		t.Fatalf("secret = %q, want %q", string(secret), "secret")
	}
}
