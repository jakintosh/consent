package database

import (
	"database/sql"
	"fmt"
	"strings"

	"git.sr.ht/~jakintosh/consent/internal/service"
)

func (db *DB) InsertRole(
	name string,
	display string,
) error {
	_, err := db.Conn.Exec(`
		INSERT INTO role (name, display)
		VALUES (?1, ?2)`,
		name,
		display,
	)
	if err != nil {
		return fmt.Errorf("couldn't insert role: %w", err)
	}
	return nil
}

func (db *DB) GetRole(
	name string,
) (
	service.Role,
	error,
) {
	row := db.Conn.QueryRow(`
		SELECT name, display
		FROM role
		WHERE name=?1`,
		name,
	)

	var record service.Role
	err := row.Scan(
		&record.Name,
		&record.Display,
	)
	if err != nil {
		return service.Role{}, fmt.Errorf("couldn't scan role: %w", err)
	}
	return record, nil
}

func (db *DB) ListRoles() (
	[]service.Role,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT name, display
		FROM role
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	var records []service.Role
	for rows.Next() {
		var record service.Role
		if err := rows.Scan(
			&record.Name,
			&record.Display,
		); err != nil {
			return nil, fmt.Errorf("couldn't scan role: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}
	return records, nil
}

func (db *DB) UpdateRole(
	name string,
	updates *service.RoleUpdate,
) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if updates.Display != nil {
		setClauses = append(setClauses, fmt.Sprintf("display=?%d", argIdx))
		args = append(args, *updates.Display)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE role
		SET %s
		WHERE name=?%d`,
		strings.Join(setClauses, ", "),
		argIdx,
	)
	args = append(args, name)

	result, err := db.Conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("couldn't update role: %w", err)
	}
	if resultsEmpty(result) {
		return sql.ErrNoRows
	}
	return nil
}

func (db *DB) DeleteRole(
	name string,
) (
	bool,
	error,
) {
	result, err := db.Conn.Exec(`
		DELETE FROM role
		WHERE name=?1`,
		name,
	)
	if err != nil {
		return false, fmt.Errorf("couldn't delete role: %w", err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}
