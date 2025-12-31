package database

import "git.sr.ht/~jakintosh/consent/internal/service"

func (s *SQLiteStore) IdentityStore() service.IdentityStore {
	return s
}

func (s *SQLiteStore) InsertIdentity(
	handle string,
	secret []byte,
) error {
	_, err := s.db.Exec(`
		INSERT INTO identity (handle, secret)
		VALUES (?1, ?2);`,
		handle,
		secret,
	)
	return err
}

func (s *SQLiteStore) GetSecret(
	handle string,
) (
	[]byte,
	error,
) {
	row := s.db.QueryRow(`
		SELECT secret
		FROM identity i
		WHERE i.handle=?1;`,
		handle,
	)

	var secret []byte
	err := row.Scan(&secret)
	return secret, err
}
