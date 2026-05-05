package database

import (
	"database/sql"
	"fmt"
	"strings"

	"git.sr.ht/~jakintosh/consent/internal/service"
)

func (db *DB) GetIntegration(
	name string,
) (
	service.Integration,
	error,
) {
	row := db.Conn.QueryRow(`
		SELECT name, display, audience, redirect, homepage, logo
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
		&record.Homepage,
		&record.Logo,
	)
	if err != nil {
		return service.Integration{}, fmt.Errorf("couldn't scan integration: %w", err)
	}

	roles, err := db.GetIntegrationRoles(name)
	if err != nil {
		return service.Integration{}, fmt.Errorf("couldn't scan integration roles: %w", err)
	}
	record.RequiredRoles = roles

	return record, nil
}

func (db *DB) GetIntegrationRoles(
	integrationName string,
) (
	[]string,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT role_name
		FROM integration_roles
		WHERE integration_name=?1
		ORDER BY role_name`,
		integrationName,
	)
	if err != nil {
		return nil, fmt.Errorf("query integration roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate integration roles: %w", err)
	}
	if roles == nil {
		roles = []string{}
	}

	return roles, nil
}

func (db *DB) GetIntegrationByAudience(
	audience string,
) (
	service.Integration,
	error,
) {
	row := db.Conn.QueryRow(`
		SELECT name, display, audience, redirect, homepage, logo
		FROM integration
		WHERE audience=?1`,
		audience,
	)

	var record service.Integration
	err := row.Scan(
		&record.Name,
		&record.Display,
		&record.Audience,
		&record.Redirect,
		&record.Homepage,
		&record.Logo,
	)
	if err != nil {
		return service.Integration{}, fmt.Errorf("couldn't scan integration: %w", err)
	}

	roles, err := db.GetIntegrationRoles(record.Name)
	if err != nil {
		return service.Integration{}, fmt.Errorf("couldn't scan integration roles: %w", err)
	}
	record.RequiredRoles = roles

	return record, nil
}

func (db *DB) ListIntegrations() (
	[]service.Integration,
	error,
) {
	rows, err := db.Conn.Query(`
		SELECT i.name, i.display, i.audience, i.redirect, i.homepage, i.logo, r.role_name
		FROM integration i
		LEFT JOIN integration_roles r ON i.name = r.integration_name
		ORDER BY i.name, r.role_name`)
	if err != nil {
		return nil, fmt.Errorf("query integrations: %w", err)
	}
	defer rows.Close()

	var records []service.Integration
	var current *service.Integration
	var currentIdx = -1

	for rows.Next() {
		var name, display, audience, redirect, homepage, logo string
		var role sql.NullString

		if err := rows.Scan(&name, &display, &audience, &redirect, &homepage, &logo, &role); err != nil {
			return nil, fmt.Errorf("scan integration: %w", err)
		}

		if currentIdx == -1 {
			records = append(records, service.Integration{
				Name:          name,
				Display:       display,
				Audience:      audience,
				Redirect:      redirect,
				Homepage:      homepage,
				Logo:          logo,
				RequiredRoles: []string{},
			})
			currentIdx = 0
			current = &records[0]
		} else if current.Name != name {
			records = append(records, service.Integration{
				Name:          name,
				Display:       display,
				Audience:      audience,
				Redirect:      redirect,
				Homepage:      homepage,
				Logo:          logo,
				RequiredRoles: []string{},
			})
			currentIdx++
			current = &records[currentIdx]
		}

		if role.Valid {
			current.RequiredRoles = append(current.RequiredRoles, role.String)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate integrations: %w", err)
	}
	return records, nil
}

func (db *DB) InsertIntegration(
	name string,
	display string,
	audience string,
	redirect string,
	homepage string,
	logo string,
	requiredRoles []string,
) error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("begin insert integration tx: %w", err)
	}
	defer tx.Rollback()

	if err := db.sqlInsertIntegrationTx(tx, name, display, audience, redirect, homepage, logo); err != nil {
		return err
	}
	if len(requiredRoles) > 0 {
		if err := db.sqlInsertIntegrationRolesTx(tx, name, requiredRoles); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit insert integration tx: %w", err)
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
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO integration (name, display, audience, redirect, homepage, logo)
		VALUES (?1, ?2, ?3, ?4, ?5, ?6)
		ON CONFLICT(name) DO UPDATE SET
			display=?2,
			audience=?3,
			redirect=?4,
			homepage=?5,
			logo=?6`)
	if err != nil {
		return fmt.Errorf("prepare system integration upsert statement: %w", err)
	}
	defer stmt.Close()

	for _, integration := range integrations {
		if _, err := stmt.Exec(integration.Name, integration.Display, integration.Audience, integration.Redirect, integration.Homepage, integration.Logo); err != nil {
			return fmt.Errorf("upsert system integration %q: %w", integration.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit system integration upserts: %w", err)
	}

	return nil
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
	if updates.Homepage != nil {
		setClauses = append(setClauses, fmt.Sprintf("homepage=?%d", argIdx))
		args = append(args, *updates.Homepage)
		argIdx++
	}
	if updates.Logo != nil {
		setClauses = append(setClauses, fmt.Sprintf("logo=?%d", argIdx))
		args = append(args, *updates.Logo)
		argIdx++
	}

	if len(setClauses) == 0 && updates.RequiredRoles == nil {
		return nil
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("begin update integration tx: %w", err)
	}
	defer tx.Rollback()

	if updates.RequiredRoles != nil {
		if err := db.sqlUpdateIntegrationRolesTx(tx, name, *updates.RequiredRoles); err != nil {
			return err
		}
	}

	if len(setClauses) == 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit update integration tx: %w", err)
		}
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

	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update integration %q: %w", name, err)
	}
	if resultsEmpty(result) {
		return sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update integration tx: %w", err)
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

func (db *DB) sqlInsertIntegrationTx(
	tx *sql.Tx,
	name string,
	display string,
	audience string,
	redirect string,
	homepage string,
	logo string,
) error {
	_, err := tx.Exec(`
		INSERT INTO integration (name, display, audience, redirect, homepage, logo)
		VALUES (?1, ?2, ?3, ?4, ?5, ?6)`,
		name,
		display,
		audience,
		redirect,
		homepage,
		logo,
	)
	if err != nil {
		return fmt.Errorf("insert integration: %w", err)
	}

	return nil
}

func (db *DB) sqlInsertIntegrationRolesTx(
	tx *sql.Tx,
	integrationName string,
	roles []string,
) error {
	stmt, err := tx.Prepare(`
		INSERT INTO integration_roles (integration_name, role_name)
		VALUES (?1, ?2)`)
	if err != nil {
		return fmt.Errorf("prepare role insert: %w", err)
	}
	defer stmt.Close()

	for _, role := range roles {
		if _, err := stmt.Exec(integrationName, role); err != nil {
			return fmt.Errorf("insert role %q for %q: %w", role, integrationName, err)
		}
	}

	return nil
}

func (db *DB) sqlUpdateIntegrationRolesTx(
	tx *sql.Tx,
	integrationName string,
	roles []string,
) error {
	var exists int
	if err := tx.QueryRow(`
		SELECT 1
		FROM integration
		WHERE name=?1`,
		integrationName,
	).Scan(&exists); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		DELETE FROM integration_roles
		WHERE integration_name=?1`, integrationName); err != nil {
		return fmt.Errorf("delete old roles: %w", err)
	}

	if len(roles) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO integration_roles (integration_name, role_name)
			VALUES (?1, ?2)`)
		if err != nil {
			return fmt.Errorf("prepare role insert: %w", err)
		}
		defer stmt.Close()

		for _, role := range roles {
			if _, err := stmt.Exec(integrationName, role); err != nil {
				return fmt.Errorf("insert role %q for %q: %w", role, integrationName, err)
			}
		}
	}

	return nil
}
