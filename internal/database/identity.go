package database

import (
	"fmt"
	"log"
)

func InsertAccount(handle string, secret []byte) error {
	_, err := db.Exec(`
		INSERT INTO identity (handle, password)
		VALUES (?, ?)
		`,
		handle,
		secret,
	)
	if err != nil {
		return fmt.Errorf("couldn't insert into identity: %v", err)
	}
	log.Printf("insert into identity: %s", handle)
	return nil
}

func GetSecret(handle string) ([]byte, error) {
	row := db.QueryRow(`
		SELECT password
		FROM identity i
		WHERE i.handle=?
		`,
		handle,
	)

	var secret []byte
	err := row.Scan(&secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}
