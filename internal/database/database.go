package database

import (
	"database/sql"
	"fmt"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
	"git.sr.ht/~jakintosh/consent/internal/service"
	_ "modernc.org/sqlite"
)

type Options struct {
	Path string
	WAL  bool
}

type DB struct {
	Conn      *sql.DB
	KeysStore *keys.SQLStore
}

var _ service.Store = (*DB)(nil)

func Open(opts Options) (*DB, error) {
	conn, err := sql.Open("sqlite", opts.Path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := conn.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if opts.WAL && isFileBackedSQLite(opts.Path) {
		if _, err := conn.Exec("PRAGMA journal_mode = WAL;"); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("enable wal mode: %w", err)
		}
	}

	db := &DB{Conn: conn}
	if err := db.migrate(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	keysStore, err := keys.NewSQL(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("initialize keys store: %w", err)
	}
	db.KeysStore = keysStore

	return db, nil
}

func (db *DB) Close() error {
	return db.Conn.Close()
}

func isFileBackedSQLite(path string) bool {
	if path == ":memory:" {
		return false
	}
	if strings.HasPrefix(path, "file:") && strings.Contains(path, "mode=memory") {
		return false
	}
	return true
}
