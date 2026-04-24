package database

import (
	"database/sql"
	"fmt"
	"strings"

	"git.sr.ht/~jakintosh/consent/internal/service"
)

func (db *DB) InsertUser(
	subject string,
	handle string,
	secret []byte,
	roles []string,
) error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("begin user insert transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO identity (subject, handle, secret)
		VALUES (?1, ?2, ?3);`,
		subject,
		handle,
		secret,
	)
	if err != nil {
		return fmt.Errorf("couldn't insert user: %w", err)
	}

	ensureStmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO role (name, display)
		VALUES (?1, ?1);`)
	if err != nil {
		return fmt.Errorf("prepare role ensure statement: %w", err)
	}
	defer ensureStmt.Close()

	assignStmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO user_roles (user_subject, role_name)
		VALUES (?1, ?2);`)
	if err != nil {
		return fmt.Errorf("prepare role assign statement: %w", err)
	}
	defer assignStmt.Close()

	for _, role := range roles {
		if _, err := ensureStmt.Exec(role); err != nil {
			return fmt.Errorf("ensure role %q: %w", role, err)
		}
		if _, err := assignStmt.Exec(subject, role); err != nil {
			return fmt.Errorf("assign role %q to user %q: %w", role, subject, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user insert transaction: %w", err)
	}

	return nil
}

func (db *DB) GetUserByHandle(
	handle string,
) (
	*service.User,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT i.subject, i.handle, r.name
		FROM identity i
		LEFT JOIN user_roles ur ON i.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		WHERE i.handle=?1;`,
		handle,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't query user by handle: %w", err)
	}
	defer rows.Close()

	return scanUserRows(rows)
}

func (db *DB) GetUserBySubject(
	subject string,
) (
	*service.User,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT i.subject, i.handle, r.name
		FROM identity i
		LEFT JOIN user_roles ur ON i.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		WHERE i.subject=?1;`,
		subject,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't query user by subject: %w", err)
	}
	defer rows.Close()

	return scanUserRows(rows)
}

func (db *DB) ListUsers() (
	[]service.User,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT i.subject, i.handle, r.name
		FROM identity i
		LEFT JOIN user_roles ur ON i.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		ORDER BY i.handle;`)
	if err != nil {
		return nil, fmt.Errorf("couldn't query users: %w", err)
	}
	defer rows.Close()

	bySubject := make(map[string]*service.User)
	var order []string

	for rows.Next() {
		var subject, handle string
		var roleName *string

		err := rows.Scan(&subject, &handle, &roleName)
		if err != nil {
			return nil, fmt.Errorf("couldn't scan user row: %w", err)
		}

		record, exists := bySubject[subject]
		if !exists {
			record = &service.User{
				Subject: subject,
				Handle:  handle,
				Roles:   nil,
			}
			bySubject[subject] = record
			order = append(order, subject)
		}

		if roleName != nil {
			record.Roles = append(record.Roles, *roleName)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("couldn't iterate users: %w", err)
	}

	records := make([]service.User, 0, len(order))
	for _, subject := range order {
		record := bySubject[subject]
		if record.Roles == nil {
			record.Roles = []string{}
		}
		records = append(records, *record)
	}

	return records, nil
}

func (db *DB) UpdateUser(
	subject string,
	handle string,
	roles []string,
) error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("begin user update transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE identity
		SET handle=?1
		WHERE subject=?2;`,
		handle,
		subject,
	)
	if err != nil {
		return fmt.Errorf("couldn't update user: %w", err)
	}
	if resultsEmpty(result) {
		return sql.ErrNoRows
	}

	if len(roles) > 0 {
		var placeholders []string
		args := make([]any, 0, len(roles)+1)
		args = append(args, subject)
		for i, role := range roles {
			placeholders = append(placeholders, fmt.Sprintf("?%d", i+2))
			args = append(args, role)
		}
		notInClause := strings.Join(placeholders, ", ")

		_, err = tx.Exec(fmt.Sprintf(`
			DELETE FROM user_roles
			WHERE user_subject=?1 AND role_name NOT IN (%s);`, notInClause),
			args...,
		)
		if err != nil {
			return fmt.Errorf("couldn't remove obsolete user roles: %w", err)
		}

		ensureStmt, err := tx.Prepare(`
			INSERT OR IGNORE INTO role (name, display)
			VALUES (?1, ?1);`)
		if err != nil {
			return fmt.Errorf("prepare role ensure statement: %w", err)
		}
		defer ensureStmt.Close()

		assignStmt, err := tx.Prepare(`
			INSERT OR IGNORE INTO user_roles (user_subject, role_name)
			VALUES (?1, ?2);`)
		if err != nil {
			_ = ensureStmt.Close()
			return fmt.Errorf("prepare role assign statement: %w", err)
		}
		defer assignStmt.Close()

		for _, role := range roles {
			if _, err := ensureStmt.Exec(role); err != nil {
				_ = ensureStmt.Close()
				_ = assignStmt.Close()
				return fmt.Errorf("ensure role %q: %w", role, err)
			}
			if _, err := assignStmt.Exec(subject, role); err != nil {
				_ = ensureStmt.Close()
				_ = assignStmt.Close()
				return fmt.Errorf("assign role %q to user %q: %w", role, subject, err)
			}
		}
	} else {
		_, err = tx.Exec(`
			DELETE FROM user_roles
			WHERE user_subject=?1;`,
			subject,
		)
		if err != nil {
			return fmt.Errorf("couldn't remove all user roles: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user update transaction: %w", err)
	}

	return nil
}

func (db *DB) DeleteUser(
	subject string,
) (
	bool,
	error,
) {
	result, err := db.Conn.Exec(`
		DELETE FROM identity
		WHERE subject=?1;`,
		subject,
	)
	if err != nil {
		return false, fmt.Errorf("couldn't delete user: %w", err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}

func (db *DB) GetSecret(
	handle string,
) (
	[]byte,
	error,
) {
	var secret []byte
	err := db.Conn.QueryRow(`
		SELECT secret
		FROM identity
		WHERE handle=?1;`,
		handle,
	).Scan(&secret)
	if err != nil {
		return nil, fmt.Errorf("couldn't get secret: %w", err)
	}
	return secret, nil
}

func scanUserRows(
	rows *sql.Rows,
) (
	*service.User,
	error,
) {
	var record *service.User
	var subject, handle string
	var roleNames []string

	for rows.Next() {
		var roleName *string

		if err := rows.Scan(
			&subject,
			&handle,
			&roleName,
		); err != nil {
			return nil, fmt.Errorf("couldn't scan user row: %w", err)
		}

		if record == nil {
			record = &service.User{
				Subject: subject,
				Handle:  handle,
				Roles:   roleNames,
			}
		}

		if roleName != nil {
			roleNames = append(roleNames, *roleName)
		}
	}

	if record == nil {
		return nil, sql.ErrNoRows
	}

	if roleNames == nil {
		roleNames = []string{}
	}
	record.Roles = roleNames

	return record, nil
}
