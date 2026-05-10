package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func (db *DB) InsertRefreshToken(
	token *tokens.RefreshToken,
) error {
	_, err := db.Conn.Exec(`
		INSERT INTO refresh (owner, token_hash, expiration)
		SELECT u.id, ?1, ?2
		FROM user u
		WHERE u.subject=?3`,
		refreshTokenHash(token.Encoded()),
		token.Expiration().Unix(),
		token.Subject(),
	)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (db *DB) GetRefreshTokenOwner(
	jwt string,
) (
	string,
	error,
) {
	row := db.Conn.QueryRow(`
		SELECT u.subject
		FROM refresh r
		JOIN user u ON r.owner = u.id
		WHERE r.token_hash=?1`,
		refreshTokenHash(jwt),
	)

	var subject string
	err := row.Scan(&subject)
	if err != nil {
		return "", fmt.Errorf("query refresh token owner: %w", err)
	}
	return subject, nil
}

func (db *DB) DeleteRefreshToken(
	jwt string,
) (
	bool,
	error,
) {
	result, err := db.Conn.Exec(`
		DELETE FROM refresh
		WHERE id IN (
			SELECT r.id
			FROM refresh r
			JOIN user u ON r.owner=u.id
			WHERE token_hash=?1
		)`,
		refreshTokenHash(jwt),
	)
	if err != nil {
		return false, fmt.Errorf("delete refresh token: %w", err)
	}

	deleted := !resultsEmpty(result)
	return deleted, nil
}

func refreshTokenHash(jwt string) string {
	sum := sha256.Sum256([]byte(jwt))
	return hex.EncodeToString(sum[:])
}
