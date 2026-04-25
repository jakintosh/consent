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
		INSERT INTO user (subject, handle, secret)
		VALUES (?1, ?2, ?3)`,
		subject,
		handle,
		secret,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	if err := ensureAndAssignRolesTx(tx, subject, roles); err != nil {
		return err
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
		SELECT u.subject, u.handle, r.name
		FROM user u
		LEFT JOIN user_roles ur ON u.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		WHERE u.handle=?1`,
		handle,
	)
	if err != nil {
		return nil, fmt.Errorf("query user by handle %q: %w", handle, err)
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
		SELECT u.subject, u.handle, r.name
		FROM user u
		LEFT JOIN user_roles ur ON u.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		WHERE u.subject=?1`,
		subject,
	)
	if err != nil {
		return nil, fmt.Errorf("query user by subject %q: %w", subject, err)
	}
	defer rows.Close()

	return scanUserRows(rows)
}

func (db *DB) ListUsers() (
	[]service.User,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT u.subject, u.handle, r.name
		FROM user u
		LEFT JOIN user_roles ur ON u.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		ORDER BY u.handle`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	bySubject := make(map[string]*service.User)
	var order []string

	for rows.Next() {
		var subject, handle string
		var roleName *string

		err := rows.Scan(&subject, &handle, &roleName)
		if err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
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
		UPDATE user
		SET handle=?1
		WHERE subject=?2`,
		handle,
		subject,
	)
	if err != nil {
		return fmt.Errorf("update user %q: %w", subject, err)
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
			WHERE user_subject=?1 AND role_name NOT IN (%s)`, notInClause),
			args...,
		)
		if err != nil {
			return fmt.Errorf("remove obsolete roles for user %q: %w", subject, err)
		}

		if err := ensureAndAssignRolesTx(tx, subject, roles); err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(`
			DELETE FROM user_roles
			WHERE user_subject=?1`,
			subject,
		)
		if err != nil {
			return fmt.Errorf("remove all roles for user %q: %w", subject, err)
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
		DELETE FROM user
		WHERE subject=?1`,
		subject,
	)
	if err != nil {
		return false, fmt.Errorf("delete user %q: %w", subject, err)
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
		FROM user
		WHERE handle=?1`,
		handle,
	).Scan(&secret)
	if err != nil {
		return nil, fmt.Errorf("get secret for handle %q: %w", handle, err)
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
			return nil, fmt.Errorf("scan user row: %w", err)
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

func ensureAndAssignRolesTx(
	tx *sql.Tx,
	subject string,
	roles []string,
) error {
	ensureStmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO role (name, display)
		VALUES (?1, ?1)`)
	if err != nil {
		return fmt.Errorf("prepare role ensure statement: %w", err)
	}
	defer ensureStmt.Close()

	assignStmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO user_roles (user_subject, role_name)
		VALUES (?1, ?2)`)
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

	return nil
}
