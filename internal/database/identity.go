package database

func initIdentity() error {
	return initTable(
		"identity",
		`CREATE TABLE IF NOT EXISTS identity (
			id          INTEGER PRIMARY KEY,
			handle      TEXT UNIQUE,
			secret      BLOB
		);`,
	)
}

func InsertAccount(handle string, secret []byte) error {
	_, err := db.Exec(`
		INSERT INTO identity (handle, secret)
		VALUES (?, ?)
		`,
		handle,
		secret,
	)
	return err
}

func GetSecret(handle string) ([]byte, error) {
	row := db.QueryRow(`
		SELECT secret
		FROM identity i
		WHERE i.handle=?
		`,
		handle,
	)

	var secret []byte
	err := row.Scan(&secret)
	return secret, err
}
