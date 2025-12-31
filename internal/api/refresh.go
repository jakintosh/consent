package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func (a *API) Refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := RefreshRequest{}
		if ok := decodeRequest(&req, w, r); !ok {
			return
		}

		// read the token in the request
		token := tokens.RefreshToken{}
		if err := token.Decode(req.RefreshToken, a.tokenValidator); err != nil {
			logApiErr(r, fmt.Sprintf("couldn't decode refresh token: %v", err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// consume the token from the db
		ok, err := deleteRefresh(a.db, req.RefreshToken)
		if !ok {
			logApiErr(r, "refresh token couldn't be deleted: not found")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err != nil {
			logApiErr(r, fmt.Sprintf("refresh token couldn't be deleted: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// issue new access token
		accessToken, err := a.tokenIssuer.IssueAccessToken(
			token.Subject(),
			token.Audience(),
			time.Minute*30,
		)
		if err != nil {
			logApiErr(r, fmt.Sprintf("couldn't issue access token: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// issue new refresh token
		refreshToken, err := a.tokenIssuer.IssueRefreshToken(
			token.Subject(),
			token.Audience(),
			time.Hour*72,
		)
		if err != nil {
			logApiErr(r, fmt.Sprintf("couldn't issue refresh token: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// insert into database
		err = insertRefresh(
			a.db,
			refreshToken.Subject(),
			refreshToken.Encoded(),
			refreshToken.Expiration().Unix(),
		)
		if err != nil {
			logApiErr(r, "failed to insert refresh token")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := RefreshResponse{
			RefreshToken: refreshToken.Encoded(),
			AccessToken:  accessToken.Encoded(),
		}
		returnJson(&response, w)
	}
}

func insertRefresh(
	db *sql.DB,
	handle string,
	jwt string,
	expiration int64,
) error {
	_, err := db.Exec(`
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

func getRefreshHandle(
	db *sql.DB,
	jwt string,
) (
	string,
	error,
) {
	row := db.QueryRow(`
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

func deleteRefresh(
	db *sql.DB,
	jwt string,
) (
	bool,
	error,
) {
	result, err := db.Exec(`
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
