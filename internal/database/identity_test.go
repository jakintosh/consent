package database_test

import (
	"database/sql"
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/database"
)

func setupIdentityStore(t *testing.T) *database.SQLiteStore {
	t.Helper()
	store := database.NewSQLiteStore(":memory:")
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestInsertIdentity_Success(t *testing.T) {
	t.Parallel()
	store := setupIdentityStore(t)

	// inserting a new identity succeeds
	err := store.InsertIdentity("alice", []byte("hashed-password"))
	if err != nil {
		t.Fatalf("InsertIdentity failed: %v", err)
	}
}

func TestInsertIdentity_DuplicateHandle(t *testing.T) {
	t.Parallel()
	store := setupIdentityStore(t)

	// first insert succeeds
	if err := store.InsertIdentity("alice", []byte("password1")); err != nil {
		t.Fatalf("InsertIdentity failed: %v", err)
	}

	// second insert with same handle fails
	err := store.InsertIdentity("alice", []byte("password2"))
	if err == nil {
		t.Fatal("expected error for duplicate handle")
	}
}

func TestInsertIdentity_MultipleUsers(t *testing.T) {
	t.Parallel()
	store := setupIdentityStore(t)

	// multiple unique users can be inserted
	if err := store.InsertIdentity("alice", []byte("password-a")); err != nil {
		t.Fatalf("InsertIdentity alice failed: %v", err)
	}
	if err := store.InsertIdentity("bob", []byte("password-b")); err != nil {
		t.Fatalf("InsertIdentity bob failed: %v", err)
	}
	if err := store.InsertIdentity("charlie", []byte("password-c")); err != nil {
		t.Fatalf("InsertIdentity charlie failed: %v", err)
	}
}

func TestGetSecret_ExistingUser(t *testing.T) {
	t.Parallel()
	store := setupIdentityStore(t)

	// setup
	expected := []byte("my-secret-hash")

	// first insert identity
	if err := store.InsertIdentity("bob", expected); err != nil {
		t.Fatalf("InsertIdentity failed: %v", err)
	}

	// retrieving secret for existing user returns correct value
	secret, err := store.GetSecret("bob")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(secret) != string(expected) {
		t.Errorf("GetSecret = %s, want %s", string(secret), string(expected))
	}
}

func TestGetSecret_NonExistentUser(t *testing.T) {
	t.Parallel()
	store := setupIdentityStore(t)

	// querying non-existent user returns ErrNoRows
	_, err := store.GetSecret("unknown")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetSecret_CorrectUser(t *testing.T) {
	t.Parallel()
	store := setupIdentityStore(t)

	// setup two users
	if err := store.InsertIdentity("alice", []byte("alice-secret")); err != nil {
		t.Fatalf("InsertIdentity failed: %v", err)
	}
	if err := store.InsertIdentity("bob", []byte("bob-secret")); err != nil {
		t.Fatalf("InsertIdentity failed: %v", err)
	}

	// each user's secret is retrieved correctly
	secret, err := store.GetSecret("alice")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(secret) != "alice-secret" {
		t.Errorf("GetSecret = %s, want alice-secret", string(secret))
	}

	secret, err = store.GetSecret("bob")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(secret) != "bob-secret" {
		t.Errorf("GetSecret = %s, want bob-secret", string(secret))
	}
}

func TestInsertIdentity_EmptyHandle(t *testing.T) {
	t.Parallel()
	store := setupIdentityStore(t)

	// empty handle is allowed by schema
	err := store.InsertIdentity("", []byte("password"))
	if err != nil {
		t.Fatalf("InsertIdentity with empty handle failed: %v", err)
	}
}

func TestInsertIdentity_BinarySecret(t *testing.T) {
	t.Parallel()
	store := setupIdentityStore(t)

	// binary data is stored and retrieved correctly
	binarySecret := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd}
	if err := store.InsertIdentity("binary-user", binarySecret); err != nil {
		t.Fatalf("InsertIdentity with binary secret failed: %v", err)
	}

	secret, err := store.GetSecret("binary-user")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if len(secret) != len(binarySecret) {
		t.Fatalf("secret length mismatch: got %d, want %d", len(secret), len(binarySecret))
	}
	for i := range binarySecret {
		if secret[i] != binarySecret[i] {
			t.Errorf("secret byte %d mismatch: got %x, want %x", i, secret[i], binarySecret[i])
		}
	}
}
