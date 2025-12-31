package api

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func initDatabase(
	dbPath string,
) *sql.DB {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatalf("failed to init database schema: couldn't enable foreign keys: %v\n", err)
	}

	if err := initTable(db, "identity", `
		CREATE TABLE IF NOT EXISTS identity (
			id          INTEGER PRIMARY KEY,
			handle      TEXT UNIQUE,
			secret      BLOB
		);`,
	); err != nil {
		log.Fatalf("failed to init database: %v\n", err)
	}

	if err := initTable(db, "referesh", `
		CREATE TABLE IF NOT EXISTS refresh (
			id          INTEGER PRIMARY KEY,
			owner       INTEGER,
			jwt         TEXT,
			expiration  INTEGER,
			FOREIGN KEY (owner) REFERENCES identity (id)
		);`,
	); err != nil {
		log.Fatalf("failed to init database: %v\n", err)
	}

	return db
}

func initTable(
	db *sql.DB,
	name string,
	sql string,
) error {
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("failed to init '%s' table schema: %v\n", name, err)
	}
	return nil
}

func resultsEmpty(
	result sql.Result,
) bool {
	count, err := result.RowsAffected()
	if err != nil {
		return false
	}
	return count == 0
}
