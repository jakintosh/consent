package database

import (
	"fmt"
)

func initRefresh() error {
	return initTable(
		"referesh",
		`CREATE TABLE IF NOT EXISTS refresh (
			id          INTEGER PRIMARY KEY,
			owner       INTEGER,
			jwt         TEXT,
			expiration  INTEGER,
			FOREIGN KEY (owner) REFERENCES identity (id)
		);`,
	)
}

func InsertRefresh(handle string, jwt string, expiration int64) error {
	_, err := db.Exec(`
        INSERT INTO refresh (owner, jwt, expiration)
        SELECT i.id, ?, ?
        FROM identity i
        WHERE i.handle=?
		`,
		jwt,
		expiration,
		handle,
	)
	if err != nil {
		return fmt.Errorf("couldn't insert into identity: %v", err)
	}
	return nil
}

func GetRefreshHandle(jwt string) (string, error) {
	row := db.QueryRow(`
			SELECT handle
			FROM refresh
			WHERE jwt=?
		`,
		jwt,
	)

	var handle string
	err := row.Scan(&handle)
	if err != nil {
		return "", fmt.Errorf("couldn't scan refresh handle: %v", err)
	}
	return handle, nil
}

func DeleteRefresh(jwt string) (bool, error) {
	result, err := db.Exec(`
        DELETE FROM refresh
        WHERE id IN (
            SELECT r.id
            FROM refresh r
            JOIN identity i ON r.owner=i.id
            WHERE jwt=?
		)`,
		jwt,
	)
	if err != nil {
		return false, fmt.Errorf("couldn't delete from identity: %v", err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}
