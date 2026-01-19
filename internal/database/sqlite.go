// Package database provides SQLite persistence for identity and refresh token storage.
package database

import (
	"database/sql"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
	_ "modernc.org/sqlite"
)

type SQLStoreOptions struct {
	Path string
}

type SQLStore struct {
	db        *sql.DB
	KeysStore *keys.SQLStore
}

func NewSQLStore(opts SQLStoreOptions) (*SQLStore, error) {
	db, err := sql.Open("sqlite", opts.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to init database schema: couldn't enable foreign keys: %v", err)
	}

	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init database: %v", err)
	}

	keysStore, err := keys.NewSQL(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keys store: %v", err)
	}

	return &SQLStore{db: db, KeysStore: keysStore}, nil
}

func (s *SQLStore) Close() error {
	return s.db.Close()
}

func initSchema(db *sql.DB) error {
	if err := initTable(db, "identity", `
		CREATE TABLE IF NOT EXISTS identity (
			id          INTEGER PRIMARY KEY,
			handle      TEXT UNIQUE,
			secret      BLOB
		);`,
	); err != nil {
		return err
	}

	if err := initTable(db, "refresh", `
		CREATE TABLE IF NOT EXISTS refresh (
			id          INTEGER PRIMARY KEY,
			owner       INTEGER,
			jwt         TEXT,
			expiration  INTEGER,
			FOREIGN KEY (owner) REFERENCES identity (id)
		);`,
	); err != nil {
		return err
	}

	if err := initTable(db, "service", `
		CREATE TABLE IF NOT EXISTS service (
			name      TEXT PRIMARY KEY,
			display   TEXT NOT NULL,
			audience  TEXT NOT NULL,
			redirect  TEXT NOT NULL
		);`,
	); err != nil {
		return err
	}

	return nil
}

func initTable(
	db *sql.DB,
	name string,
	sql string,
) error {
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("failed to init '%s' table schema: %v", name, err)
	}
	return nil
}
