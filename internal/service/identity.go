package service

func (s *Service) insertAccount(
	handle string,
	secret []byte,
) error {
	_, err := s.db.Exec(`
		INSERT INTO identity (handle, secret)
		VALUES (?, ?);`,
		handle,
		secret,
	)
	return err
}

func (s *Service) getSecret(
	handle string,
) (
	[]byte,
	error,
) {
	row := s.db.QueryRow(`
		SELECT secret
		FROM identity i
		WHERE i.handle=?;`,
		handle,
	)

	var secret []byte
	err := row.Scan(&secret)
	return secret, err
}
