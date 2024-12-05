package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func Init(dbPath string) {

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}
	db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE IF NOT EXISTS identity (
			id INTEGER PRIMARY KEY,
			handle TEXT UNIQUE,
			password BLOB
		);
		CREATE TABLE IF NOT EXISTS refresh (
			id INTEGER PRIMARY KEY,
			owner INTEGER,
			jwt TEXT,
			expiration INTEGER,
			FOREIGN_KEY (identity.id)
				REFERENCES identity (id),
		);
	`)
}

func InsertAccount(handle string, secret []byte) error {
	_, err := db.Exec(`
		INSERT INTO identity (handle, password)
		VALUES (?, ?)
		`, handle, secret)
	if err != nil {
		return fmt.Errorf("couldn't insert into identity: %v", err)
	}
	log.Printf("insert into identity: %s", handle)
	return nil
}
