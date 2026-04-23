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
		VALUES (?1, ?2);`,
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
	service.RoleDefinition,
	error,
) {
	row := db.Conn.QueryRow(`
		SELECT name, display
		FROM role
		WHERE name=?1;`,
		name,
	)

	var record service.RoleDefinition
	err := row.Scan(&record.Name, &record.Display)
	if err != nil {
		return service.RoleDefinition{}, fmt.Errorf("couldn't scan role: %w", err)
	}
	return record, nil
}

func (db *DB) UpdateRoleDisplay(
	name string,
	display string,
) error {
	result, err := db.Conn.Exec(`
		UPDATE role
		SET display=?1
		WHERE name=?2;`,
		display,
		name,
	)
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
		WHERE name=?1;`,
		name,
	)
	if err != nil {
		return false, fmt.Errorf("couldn't delete role: %w", err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}

func (db *DB) ListRoles() (
	[]service.RoleDefinition,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT name, display
		FROM role
		ORDER BY name;`)
	if err != nil {
		return nil, fmt.Errorf("couldn't query roles: %v", err)
	}
	defer rows.Close()

	var records []service.RoleDefinition
	for rows.Next() {
		var record service.RoleDefinition
		if err := rows.Scan(&record.Name, &record.Display); err != nil {
			return nil, fmt.Errorf("couldn't scan role: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("couldn't iterate roles: %v", err)
	}
	return records, nil
}

func (db *DB) CountUsersWithRole(
	name string,
) (
	int,
	error,
) {
	var count int
	err := db.Conn.QueryRow(`
		SELECT COUNT(*)
		FROM user_roles
		WHERE role_name=?1;`,
		name,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("couldn't count users with role: %w", err)
	}
	return count, nil
}

func (db *DB) ValidateRoleNames(
	names []string,
) error {
	if len(names) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString("SELECT name FROM role WHERE name IN (")
	args := make([]any, 0, len(names))
	for i, name := range names {
		if i > 0 {
			query.WriteString(", ")
		}
		fmt.Fprintf(&query, "?%d", i+1)
		args = append(args, name)
	}
	query.WriteString(")")

	rows, err := db.Conn.Query(query.String(), args...)
	if err != nil {
		return fmt.Errorf("couldn't validate roles: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("couldn't scan role: %w", err)
		}
		existing[name] = true
	}

	for _, name := range names {
		if !existing[name] {
			return fmt.Errorf("role %q not found", name)
		}
	}

	return nil
}
