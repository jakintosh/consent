package database_test

import (
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	store := testutil.SetupTestDB(t)

	err := store.InsertUser("subject-alice", "alice", []byte("secret"), nil)
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
}
