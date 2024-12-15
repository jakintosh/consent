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

	if _, err = db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatalf("failed to init database schema: couldn't enable foreign keys: %v\n", err)
	}
	if err = initIdentity(); err != nil {
		log.Fatalf("failed to init database: %v\n", err)
	}
	if err = initRefresh(); err != nil {
		log.Fatalf("failed to init database: %v\n", err)
	}
}

func initTable(name string, sql string) error {
	_, err := db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to init '%s' table schema: %v\n", name, err)
	}
	return nil
}

func resultsEmpty(result sql.Result) bool {
	count, err := result.RowsAffected()
	if err != nil {
		return false
	}
	return count == 0
}
