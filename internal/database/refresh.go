package database

import (
	"database/sql"
	"fmt"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func (s *SQLStore) InsertRefreshToken(
	token *tokens.RefreshToken,
) error {
	_, err := s.db.Exec(`
		INSERT INTO refresh (owner, jwt, expiration)
		SELECT i.id, ?1, ?2
		FROM identity i
		WHERE i.subject=?3;`,
		token.Encoded(),
		token.Expiration().Unix(),
		token.Subject(),
	)
	if err != nil {
		return fmt.Errorf("couldn't insert into refresh: %v", err)
	}
	return nil
}

func (s *SQLStore) GetRefreshTokenOwner(
	jwt string,
) (
	string,
	error,
) {
	row := s.db.QueryRow(`
		SELECT i.subject
		FROM refresh r
		JOIN identity i ON r.owner = i.id
		WHERE r.jwt=?1;`,
		jwt,
	)

	var subject string
	err := row.Scan(&subject)
	if err != nil {
		return "", fmt.Errorf("couldn't scan refresh handle: %v", err)
	}
	return subject, nil
}

func (s *SQLStore) DeleteRefreshToken(
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
			WHERE jwt=?1
		);`,
		jwt,
	)
	if err != nil {
		return false, fmt.Errorf("couldn't delete from refresh: %v", err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}

func resultsEmpty(result sql.Result) bool {
	count, err := result.RowsAffected()
	if err != nil {
		return false
	}
	return count == 0
}
