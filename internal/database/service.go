package database

import (
	"database/sql"
	"fmt"

	"git.sr.ht/~jakintosh/consent/internal/service"
)

func (db *DB) InsertService(
	name string,
	display string,
	audience string,
	redirect string,
) error {
	_, err := db.Conn.Exec(`
		INSERT INTO service (name, display, audience, redirect)
		VALUES (?1, ?2, ?3, ?4);`,
		name,
		display,
		audience,
		redirect,
	)
	if err != nil {
		return fmt.Errorf("couldn't insert into service: %v", err)
	}
	return nil
}

func (db *DB) UpsertSystemServices(
	services []service.ServiceDefinition,
) error {
	if len(services) == 0 {
		return nil
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("couldn't begin system service upsert transaction: %v", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO service (name, display, audience, redirect)
		VALUES (?1, ?2, ?3, ?4)
		ON CONFLICT(name) DO UPDATE SET
			display=?2,
			audience=?3,
			redirect=?4;`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("couldn't prepare system service upsert statement: %v", err)
	}
	defer stmt.Close()

	for _, service := range services {
		if _, err := stmt.Exec(service.Name, service.Display, service.Audience, service.Redirect); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("couldn't upsert system service %q: %v", service.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("couldn't commit system service upserts: %v", err)
	}

	return nil
}

func (db *DB) GetService(
	name string,
) (
	service.ServiceDefinition,
	error,
) {
	row := db.Conn.QueryRow(`
		SELECT name, display, audience, redirect
		FROM service
		WHERE name=?1;`,
		name,
	)

	var record service.ServiceDefinition
	err := row.Scan(&record.Name, &record.Display, &record.Audience, &record.Redirect)
	if err != nil {
		return service.ServiceDefinition{}, fmt.Errorf("couldn't scan service: %w", err)
	}
	return record, nil
}

func (db *DB) UpdateService(
	name string,
	display string,
	audience string,
	redirect string,
) error {
	result, err := db.Conn.Exec(`
		UPDATE service
		SET display=?1, audience=?2, redirect=?3
		WHERE name=?4;`,
		display,
		audience,
		redirect,
		name,
	)
	if err != nil {
		return fmt.Errorf("couldn't update service: %v", err)
	}
	if resultsEmpty(result) {
		return sql.ErrNoRows
	}
	return nil
}

func (db *DB) DeleteService(
	name string,
) (
	bool,
	error,
) {
	result, err := db.Conn.Exec(`
		DELETE FROM service
		WHERE name=?1;`,
		name,
	)
	if err != nil {
		return false, fmt.Errorf("couldn't delete service: %v", err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}

func (db *DB) ListServices() ([]service.ServiceDefinition, error) {
	rows, err := db.Conn.Query(`
		SELECT name, display, audience, redirect
		FROM service
		ORDER BY name;`)
	if err != nil {
		return nil, fmt.Errorf("couldn't query services: %v", err)
	}
	defer rows.Close()

	var records []service.ServiceDefinition
	for rows.Next() {
		var record service.ServiceDefinition
		if err := rows.Scan(&record.Name, &record.Display, &record.Audience, &record.Redirect); err != nil {
			return nil, fmt.Errorf("couldn't scan service: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("couldn't iterate services: %v", err)
	}
	return records, nil
}
