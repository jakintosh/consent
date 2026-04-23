package database

import (
	"database/sql"
	"fmt"

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

func (db *DB) InsertIdentity(
	subject string,
	handle string,
	secret []byte,
) error {
	return db.InsertUser(
		subject,
		handle,
		secret,
		nil,
	)
}

func (db *DB) GetUserByHandle(
	handle string,
) (
	service.IdentityRecord,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT i.subject, i.handle, i.secret, r.name
		FROM identity i
		LEFT JOIN user_roles ur ON i.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		WHERE i.handle=?1;`,
		handle,
	)
	if err != nil {
		return service.IdentityRecord{}, fmt.Errorf("couldn't query user by handle: %w", err)
	}
	defer rows.Close()

	return scanUserRows(rows)
}

func (db *DB) GetIdentityByHandle(
	handle string,
) (
	service.IdentityRecord,
	error,
) {
	return db.GetUserByHandle(handle)
}

func (db *DB) GetUserBySubject(
	subject string,
) (
	service.IdentityRecord,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT i.subject, i.handle, i.secret, r.name
		FROM identity i
		LEFT JOIN user_roles ur ON i.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		WHERE i.subject=?1;`,
		subject,
	)
	if err != nil {
		return service.IdentityRecord{}, fmt.Errorf("couldn't query user by subject: %w", err)
	}
	defer rows.Close()

	return scanUserRows(rows)
}

func (db *DB) GetIdentityBySubject(
	subject string,
) (
	service.IdentityRecord,
	error,
) {
	return db.GetUserBySubject(subject)
}

func (db *DB) ListUsers() (
	[]service.IdentityRecord,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT i.subject, i.handle, i.secret, r.name
		FROM identity i
		LEFT JOIN user_roles ur ON i.subject = ur.user_subject
		LEFT JOIN role r ON ur.role_name = r.name
		ORDER BY i.handle;`)
	if err != nil {
		return nil, fmt.Errorf("couldn't query users: %w", err)
	}
	defer rows.Close()

	bySubject := make(map[string]*service.IdentityRecord)
	var order []string

	for rows.Next() {
		var subject, handle string
		var secret []byte
		var roleName *string

		err := rows.Scan(&subject, &handle, &secret, &roleName)
		if err != nil {
			return nil, fmt.Errorf("couldn't scan user row: %w", err)
		}

		record, exists := bySubject[subject]
		if !exists {
			record = &service.IdentityRecord{
				Subject: subject,
				Handle:  handle,
				Secret:  secret,
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

	records := make([]service.IdentityRecord, 0, len(order))
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

	_, err = tx.Exec(`
		DELETE FROM user_roles
		WHERE user_subject=?1;`,
		subject,
	)
	if err != nil {
		return fmt.Errorf("couldn't remove old user roles: %w", err)
	}

	if len(roles) > 0 {
		ensureStmt, err := tx.Prepare(`
			INSERT OR IGNORE INTO role (name, display)
			VALUES (?1, ?1);`)
		if err != nil {
			return fmt.Errorf("prepare role ensure statement: %w", err)
		}

		assignStmt, err := tx.Prepare(`
			INSERT OR IGNORE INTO user_roles (user_subject, role_name)
			VALUES (?1, ?2);`)
		if err != nil {
			_ = ensureStmt.Close()
			return fmt.Errorf("prepare role assign statement: %w", err)
		}

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

		_ = ensureStmt.Close()
		_ = assignStmt.Close()
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
	identity, err := db.GetUserByHandle(handle)
	if err != nil {
		return nil, err
	}
	return identity.Secret, nil
}

func scanUserRows(
	rows *sql.Rows,
) (
	service.IdentityRecord,
	error,
) {
	var record service.IdentityRecord
	var roleNames []string

	for rows.Next() {
		var subject, handle string
		var secret []byte
		var roleName *string

		if err := rows.Scan(
			&subject,
			&handle,
			&secret,
			&roleName,
		); err != nil {
			return service.IdentityRecord{}, fmt.Errorf("couldn't scan user row: %w", err)
		}

		if record.Subject == "" {
			record = service.IdentityRecord{
				Subject: subject,
				Handle:  handle,
				Secret:  secret,
				Roles:   roleNames,
			}
		}

		if roleName != nil {
			roleNames = append(roleNames, *roleName)
		}
	}

	if record.Subject == "" {
		return service.IdentityRecord{}, sql.ErrNoRows
	}

	if roleNames == nil {
		roleNames = []string{}
	}
	record.Roles = roleNames

	return record, nil
}
