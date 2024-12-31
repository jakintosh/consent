package api

import (
	"fmt"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"git.sr.ht/~jakintosh/consent/internal/tokens"
)

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if ok := decodeRequest(&req, w, r); !ok {
		return
	}

	// read the token in the request
	token := tokens.RefreshToken{}
	if err := token.Decode(req.RefreshToken); err != nil {
		logApiErr(r, fmt.Sprintf("couldn't decode refresh token: %v", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// consume the token from the db
	ok, err := database.DeleteRefresh(req.RefreshToken)
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
	accessToken, err := tokens.IssueAccessToken(
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
	refreshToken, err := tokens.IssueRefreshToken(
		token.Subject(),
		token.Audience(),
		time.Hour*72,
	)
	if err != nil {
		logApiErr(r, fmt.Sprintf("couldn't issue refresh token: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := RefreshResponse{
		RefreshToken: refreshToken.Encoded(),
		AccessToken:  accessToken.Encoded(),
	}
	returnJson(&response, w)
}
