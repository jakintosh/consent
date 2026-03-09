package database

import "git.sr.ht/~jakintosh/consent/internal/service"

func (s *SQLStore) InsertIdentity(
	subject string,
	handle string,
	secret []byte,
) error {
	_, err := s.db.Exec(`
		INSERT INTO identity (subject, handle, secret)
		VALUES (?1, ?2, ?3);`,
		subject,
		handle,
		secret,
	)
	return err
}

func (s *SQLStore) GetIdentityByHandle(
	handle string,
) (
	service.IdentityRecord,
	error,
) {
	row := s.db.QueryRow(`
		SELECT subject, handle, secret
		FROM identity i
		WHERE i.handle=?1;`,
		handle,
	)

	var identity service.IdentityRecord
	err := row.Scan(&identity.Subject, &identity.Handle, &identity.Secret)
	return identity, err
}

func (s *SQLStore) GetIdentityBySubject(
	subject string,
) (
	service.IdentityRecord,
	error,
) {
	row := s.db.QueryRow(`
		SELECT subject, handle, secret
		FROM identity i
		WHERE i.subject=?1;`,
		subject,
	)

	var identity service.IdentityRecord
	err := row.Scan(&identity.Subject, &identity.Handle, &identity.Secret)
	return identity, err
}

func (s *SQLStore) GetSecret(
	handle string,
) (
	[]byte,
	error,
) {
	identity, err := s.GetIdentityByHandle(handle)
	if err != nil {
		return nil, err
	}
	return identity.Secret, nil
}
