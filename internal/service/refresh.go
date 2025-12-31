package service

import "fmt"

func (s *Service) insertRefresh(
	handle string,
	jwt string,
	expiration int64,
) error {
	_, err := s.db.Exec(`
        INSERT INTO refresh (owner, jwt, expiration)
        SELECT i.id, ?, ?
        FROM identity i
        WHERE i.handle=?;`,
		jwt,
		expiration,
		handle,
	)
	if err != nil {
		return fmt.Errorf("couldn't insert into refresh: %v", err)
	}
	return nil
}

func (s *Service) getRefreshHandle(
	jwt string,
) (
	string,
	error,
) {
	row := s.db.QueryRow(`
		SELECT handle
		FROM refresh
		WHERE jwt=?;`,
		jwt,
	)

	var handle string
	err := row.Scan(&handle)
	if err != nil {
		return "", fmt.Errorf("couldn't scan refresh handle: %v", err)
	}
	return handle, nil
}

func (s *Service) deleteRefresh(
	jwt string,
) (
	bool,
	error,
) {
	result, err := s.db.Exec(`
        DELETE FROM refresh
        WHERE id IN (
            SELECT r.id
            FROM refresh r
            JOIN identity i ON r.owner=i.id
            WHERE jwt=?
		);`,
		jwt,
	)
	if err != nil {
		return false, fmt.Errorf("couldn't delete from refresh: %v", err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}
