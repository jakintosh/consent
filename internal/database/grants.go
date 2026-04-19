package database

import (
	"fmt"
	"time"
)

func (db *DB) ListGrantedScopeNames(subject, service string) ([]string, error) {
	rows, err := db.Conn.Query(`
		SELECT g.scope_name
		FROM "grant" g
		JOIN identity i ON g.owner = i.id
		WHERE i.subject=?1 AND g.service=?2
		ORDER BY g.scope_name;`,
		subject,
		service,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't query grants: %v", err)
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			return nil, fmt.Errorf("couldn't scan grant scope: %v", err)
		}
		scopes = append(scopes, scope)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("couldn't iterate grants: %v", err)
	}

	return scopes, nil
}

func (db *DB) InsertGrants(subject, service string, scopes []string) error {
	if len(scopes) == 0 {
		return nil
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("couldn't begin grant insert transaction: %v", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO "grant" (owner, service, scope_name, created_at)
		SELECT i.id, ?1, ?2, ?3
		FROM identity i
		WHERE i.subject=?4
		ON CONFLICT(owner, service, scope_name) DO NOTHING;`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("couldn't prepare grant insert statement: %v", err)
	}
	defer stmt.Close()

	createdAt := time.Now().Unix()
	for _, scope := range scopes {
		if _, err := stmt.Exec(service, scope, createdAt, subject); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("couldn't insert grant %q: %v", scope, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("couldn't commit grant inserts: %v", err)
	}

	return nil
}
