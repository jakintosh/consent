package database

import "fmt"

type Migration struct {
	Version int
	Name    string
	SQL     string
}

var migrations = []Migration{
	{
		Version: 1,
		Name:    "create initial schema",
		SQL: `
			CREATE TABLE IF NOT EXISTS user (
				id      INTEGER PRIMARY KEY,
				subject TEXT UNIQUE NOT NULL,
				handle  TEXT UNIQUE NOT NULL,
				secret  BLOB NOT NULL
			);

			CREATE TABLE IF NOT EXISTS role (
				name    TEXT PRIMARY KEY,
				display TEXT NOT NULL
			);

			CREATE TABLE IF NOT EXISTS user_roles (
				user_subject TEXT NOT NULL,
				role_name    TEXT NOT NULL,
				PRIMARY KEY (user_subject, role_name),
				FOREIGN KEY (user_subject) REFERENCES user(subject) ON DELETE CASCADE,
				FOREIGN KEY (role_name)    REFERENCES role(name) ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS refresh (
				id         INTEGER PRIMARY KEY,
				owner      INTEGER,
				jwt        TEXT,
				expiration INTEGER,
				FOREIGN KEY (owner) REFERENCES user(id) ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS integration (
				name     TEXT PRIMARY KEY,
				display  TEXT NOT NULL,
				audience TEXT NOT NULL,
				redirect TEXT NOT NULL
			);

			CREATE TABLE IF NOT EXISTS grant (
				id         INTEGER PRIMARY KEY,
				owner      INTEGER NOT NULL,
				integration TEXT NOT NULL,
				scope_name TEXT NOT NULL,
				created_at INTEGER NOT NULL,
				FOREIGN KEY (owner) REFERENCES user(id) ON DELETE CASCADE,
				UNIQUE (owner, integration, scope_name)
			)`,
	},
}

func (db *DB) migrate() error {
	current, err := db.userVersion()
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for _, migration := range migrations {
		if migration.Version <= current {
			continue
		}

		tx, err := db.Conn.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d %q: %w", migration.Version, migration.Name, err)
		}

		if _, err := tx.Exec(migration.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("run migration %d %q: %w", migration.Version, migration.Name, err)
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", migration.Version)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("set schema version %d: %w", migration.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d %q: %w", migration.Version, migration.Name, err)
		}

		current = migration.Version
	}

	return nil
}

func (db *DB) userVersion() (int, error) {
	var version int
	if err := db.Conn.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}
