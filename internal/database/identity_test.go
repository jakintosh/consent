package database_test

import (
	"database/sql"
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func insertIdentity(t *testing.T, store interface {
	InsertUser(string, string, []byte, []string) error
}, handle string, secret []byte) {
	t.Helper()
	if err := store.InsertUser("subject-"+handle, handle, secret, nil); err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}
}

func TestInsertIdentity_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// inserting a new identity succeeds
	insertIdentity(t, store, "alice", []byte("hashed-password"))
}

func TestInsertIdentity_DuplicateHandle(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// first insert succeeds
	insertIdentity(t, store, "alice", []byte("password1"))

	// second insert with same handle fails
	err := store.InsertUser("subject-alice-2", "alice", []byte("password2"), nil)
	if err == nil {
		t.Fatal("expected error for duplicate handle")
	}
}

func TestInsertIdentity_MultipleUsers(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// multiple unique users can be inserted
	insertIdentity(t, store, "alice", []byte("password-a"))
	insertIdentity(t, store, "bob", []byte("password-b"))
	insertIdentity(t, store, "charlie", []byte("password-c"))
}

func TestGetSecret_ExistingUser(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// setup
	expected := []byte("my-secret-hash")

	// first insert identity
	insertIdentity(t, store, "bob", expected)

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
	store := testutil.SetupTestDB(t)

	// querying non-existent user returns ErrNoRows
	_, err := store.GetSecret("unknown")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetSecret_CorrectUser(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// setup two users
	insertIdentity(t, store, "alice", []byte("alice-secret"))
	insertIdentity(t, store, "bob", []byte("bob-secret"))

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
	store := testutil.SetupTestDB(t)

	// empty handle is allowed by schema
	err := store.InsertUser("subject-empty", "", []byte("password"), nil)
	if err != nil {
		t.Fatalf("InsertUser with empty handle failed: %v", err)
	}
}

func TestInsertIdentity_BinarySecret(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// binary data is stored and retrieved correctly
	binarySecret := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd}
	insertIdentity(t, store, "binary-user", binarySecret)

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
