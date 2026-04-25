package database

import (
	"fmt"
	"time"
)

func (db *DB) ListGrantedScopeNames(
	subject string,
	integration string,
) (
	[]string,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT g.scope_name
		FROM grant g
		JOIN user u ON g.owner = u.id
		WHERE u.subject=?1 AND g.integration=?2
		ORDER BY g.scope_name`,
		subject,
		integration,
	)
	if err != nil {
		return nil, fmt.Errorf("query granted scope names: %w", err)
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			return nil, fmt.Errorf("scan grant scope: %w", err)
		}
		scopes = append(scopes, scope)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate grants: %w", err)
	}

	return scopes, nil
}

func (db *DB) InsertGrants(
	subject string,
	integration string,
	scopes []string,
) error {
	if len(scopes) == 0 {
		return nil
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("begin grant insert transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO grant (owner, integration, scope_name, created_at)
		SELECT u.id, ?1, ?2, ?3
		FROM user u
		WHERE u.subject=?4
		ON CONFLICT(owner, integration, scope_name) DO NOTHING`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare grant insert statement: %w", err)
	}
	defer stmt.Close()

	createdAt := time.Now().Unix()
	for _, scope := range scopes {
		if _, err := stmt.Exec(integration, scope, createdAt, subject); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert grant %q: %w", scope, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit grant inserts: %w", err)
	}

	return nil
}
