package api

import "database/sql"

func insertAccount(
	db *sql.DB,
	handle string,
	secret []byte,
) error {
	_, err := db.Exec(`
		INSERT INTO identity (handle, secret)
		VALUES (?, ?);`,
		handle,
		secret,
	)
	return err
}

func getSecret(
	db *sql.DB,
	handle string,
) (
	[]byte,
	error,
) {
	row := db.QueryRow(`
		SELECT secret
		FROM identity i
		WHERE i.handle=?;`,
		handle,
	)

	var secret []byte
	err := row.Scan(&secret)
	return secret, err
}
