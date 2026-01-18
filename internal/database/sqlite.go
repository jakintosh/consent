// Package database provides SQLite persistence for identity and refresh token storage.
package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type SQLStoreOptions struct {
	Path string
}

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(opts SQLStoreOptions) *SQLStore {
	db, err := sql.Open("sqlite", opts.Path)
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatalf("failed to init database schema: couldn't enable foreign keys: %v\n", err)
	}

	if err := initSchema(db); err != nil {
		log.Fatalf("failed to init database: %v\n", err)
	}

	return &SQLStore{db: db}
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
