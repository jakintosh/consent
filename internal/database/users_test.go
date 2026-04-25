package database_test

import (
	"database/sql"
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func insertUser(t *testing.T, store interface {
	InsertUser(string, string, []byte, []string) error
}, handle string, roles []string) {
	t.Helper()
	if err := store.InsertUser("subject-"+handle, handle, []byte("hashed-password"), roles); err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}
}

func insertUserWithSecret(t *testing.T, store interface {
	InsertUser(string, string, []byte, []string) error
}, subject, handle string, secret []byte, roles []string) {
	t.Helper()
	if err := store.InsertUser(subject, handle, secret, roles); err != nil {
		t.Fatalf("InsertUser failed: %v", err)
	}
}

func TestInsertUser_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// inserting a new user succeeds
	insertUser(t, store, "alice", nil)
}

func TestInsertUser_DuplicateHandle(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// first insert succeeds
	insertUser(t, store, "alice", nil)

	// second insert with same handle fails
	err := store.InsertUser("subject-alice-2", "alice", []byte("password2"), nil)
	if err == nil {
		t.Fatal("expected error for duplicate handle")
	}
}

func TestInsertUser_MultipleUsers(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// multiple unique users can be inserted
	insertUser(t, store, "alice", nil)
	insertUser(t, store, "bob", nil)
	insertUser(t, store, "charlie", nil)
}

func TestInsertUser_EmptyHandle(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// empty handle is allowed by schema
	err := store.InsertUser("subject-empty", "", []byte("password"), nil)
	if err != nil {
		t.Fatalf("InsertUser with empty handle failed: %v", err)
	}
}

func TestInsertUser_BinarySecret(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// binary data is stored and retrieved correctly
	binarySecret := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd}
	insertUserWithSecret(t, store, "subject-binary-user", "binary-user", binarySecret, nil)

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

func TestGetUserByHandle_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	user, err := store.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if user.Handle != "alice" {
		t.Errorf("GetUserByHandle handle = %s, want alice", user.Handle)
	}
	if user.Subject != "subject-alice" {
		t.Errorf("GetUserByHandle subject = %s, want subject-alice", user.Subject)
	}
}

func TestGetUserBySubject_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	user, err := store.GetUserBySubject("subject-alice")
	if err != nil {
		t.Fatalf("GetUserBySubject failed: %v", err)
	}
	if user.Subject != "subject-alice" {
		t.Errorf("GetUserBySubject subject = %s, want subject-alice", user.Subject)
	}
	if user.Handle != "alice" {
		t.Errorf("GetUserBySubject handle = %s, want alice", user.Handle)
	}
}

func TestListUsers_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)
	insertUser(t, store, "bob", nil)

	users, err := store.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	if users[0].Handle != "alice" {
		t.Errorf("users[0].handle = %s, want alice", users[0].Handle)
	}
	if users[1].Handle != "bob" {
		t.Errorf("users[1].handle = %s, want bob", users[1].Handle)
	}
}

func TestUpdateUser_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	err := store.UpdateUser("subject-alice", "alice-updated", nil)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	user, err := store.GetUserByHandle("alice-updated")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if user.Subject != "subject-alice" {
		t.Errorf("subject = %s, want subject-alice", user.Subject)
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.UpdateUser("nonexistent", "new-handle", nil)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	insertUser(t, store, "alice", nil)

	deleted, err := store.DeleteUser("subject-alice")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted = true")
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	deleted, err := store.DeleteUser("nonexistent")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	if deleted {
		t.Fatal("expected deleted = false")
	}
}

func TestGetSecret_ExistingUser(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	// insert user with custom secret
	insertUserWithSecret(t, store, "test-subject", "bob", []byte("my-secret-hash"), nil)

	// retrieving secret for existing user returns correct value
	secret, err := store.GetSecret("bob")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(secret) != "my-secret-hash" {
		t.Errorf("GetSecret = %s, want my-secret-hash", string(secret))
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
	insertUser(t, store, "alice", nil)
	insertUser(t, store, "bob", nil)

	// each user's secret is retrieved correctly
	secret, err := store.GetSecret("alice")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(secret) != "hashed-password" {
		t.Errorf("GetSecret = %s, want hashed-password", string(secret))
	}

	secret, err = store.GetSecret("bob")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(secret) != "hashed-password" {
		t.Errorf("GetSecret = %s, want hashed-password", string(secret))
	}
}
