package database

import (
	"database/sql"
	"fmt"
	"strings"

	"git.sr.ht/~jakintosh/consent/internal/service"
)

func (db *DB) InsertIntegration(
	name string,
	display string,
	audience string,
	redirect string,
) error {
	_, err := db.Conn.Exec(`
		INSERT INTO integration (name, display, audience, redirect)
		VALUES (?1, ?2, ?3, ?4)`,
		name,
		display,
		audience,
		redirect,
	)
	if err != nil {
		return fmt.Errorf("insert integration: %w", err)
	}
	return nil
}

func (db *DB) UpsertSystemIntegrations(
	integrations []service.Integration,
) error {
	if len(integrations) == 0 {
		return nil
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("begin system integration upsert transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO integration (name, display, audience, redirect)
		VALUES (?1, ?2, ?3, ?4)
		ON CONFLICT(name) DO UPDATE SET
			display=?2,
			audience=?3,
			redirect=?4`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare system integration upsert statement: %w", err)
	}
	defer stmt.Close()

	for _, integration := range integrations {
		if _, err := stmt.Exec(integration.Name, integration.Display, integration.Audience, integration.Redirect); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upsert system integration %q: %w", integration.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit system integration upserts: %w", err)
	}

	return nil
}

func (db *DB) GetIntegration(
	name string,
) (
	service.Integration,
	error,
) {
	row := db.Conn.QueryRow(`
		SELECT name, display, audience, redirect
		FROM integration
		WHERE name=?1`,
		name,
	)

	var record service.Integration
	err := row.Scan(
		&record.Name,
		&record.Display,
		&record.Audience,
		&record.Redirect,
	)
	if err != nil {
		return service.Integration{}, fmt.Errorf("couldn't scan integration: %w", err)
	}
	return record, nil
}

func (db *DB) UpdateIntegration(
	name string,
	updates *service.IntegrationUpdate,
) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if updates.Display != nil {
		setClauses = append(setClauses, fmt.Sprintf("display=?%d", argIdx))
		args = append(args, *updates.Display)
		argIdx++
	}
	if updates.Audience != nil {
		setClauses = append(setClauses, fmt.Sprintf("audience=?%d", argIdx))
		args = append(args, *updates.Audience)
		argIdx++
	}
	if updates.Redirect != nil {
		setClauses = append(setClauses, fmt.Sprintf("redirect=?%d", argIdx))
		args = append(args, *updates.Redirect)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE integration
		SET %s
		WHERE name=?%d`,
		strings.Join(setClauses, ", "),
		argIdx,
	)
	args = append(args, name)

	result, err := db.Conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update integration %q: %w", name, err)
	}
	if resultsEmpty(result) {
		return sql.ErrNoRows
	}
	return nil
}

func (db *DB) DeleteIntegration(
	name string,
) (
	bool,
	error,
) {
	result, err := db.Conn.Exec(`
		DELETE FROM integration
		WHERE name=?1`,
		name,
	)
	if err != nil {
		return false, fmt.Errorf("delete integration %q: %w", name, err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}

func (db *DB) ListIntegrations() (
	[]service.Integration,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT name, display, audience, redirect
		FROM integration
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query integrations: %w", err)
	}
	defer rows.Close()

	var records []service.Integration
	for rows.Next() {
		var record service.Integration
		if err := rows.Scan(
			&record.Name,
			&record.Display,
			&record.Audience,
			&record.Redirect,
		); err != nil {
			return nil, fmt.Errorf("couldn't scan integration: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate integrations: %w", err)
	}
	return records, nil
}
