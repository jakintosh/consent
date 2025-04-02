package api

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Handle  string `json:"handle"`
	Secret  string `json:"secret"`
	Service string `json:"service"`
}

type LoginResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func LoginForm(w http.ResponseWriter, r *http.Request) {
	req := LoginRequest{
		Handle:  r.FormValue("handle"),
		Secret:  r.FormValue("secret"),
		Service: r.FormValue("service"),
	}
	if req.Handle == "" ||
		req.Secret == "" ||
		req.Service == "" {
		logApiErr(r, "bad form request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	login(req, w, r)
}

func LoginJson(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if ok := decodeRequest(&req, w, r); !ok {
		return
	}
	login(req, w, r)
}

func login(req LoginRequest, w http.ResponseWriter, r *http.Request) {
	err := authenticate(req.Handle, req.Secret)
	if err != nil {
		logApiErr(r, fmt.Sprintf("'%s' failed to authenticate: %v", req.Handle, err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	service, err := services.GetService(req.Service)
	if err != nil {
		logApiErr(r, fmt.Sprintf("invalid service: %s", req.Service))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	refreshToken, err := tokenIssuer.IssueRefreshToken(
		req.Handle,
		[]string{service.Audience},
		time.Second*10,
	)
	if err != nil {
		logApiErr(r, fmt.Sprintf("failed to issue refresh token: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// insert into database
	err = InsertRefresh(
		refreshToken.Subject(),
		refreshToken.Encoded(),
		refreshToken.Expiration().Unix(),
	)
	if err != nil {
		logApiErr(r, "failed to insert refresh token")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	redirectUrl := buildRedirectUrlString(service.Redirect, refreshToken.Encoded())

	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
}

func authenticate(handle string, secret string) error {
	hash, err := GetSecret(handle)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %v", err)
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(secret))
	if err != nil {
		return fmt.Errorf("secret does not match")
	}

	return nil
}

func buildRedirectUrlString(redirect *url.URL, refreshToken string) string {
	redirectUrl := *redirect // 'clone' the url by dereferencing the ptr
	q := redirectUrl.Query()
	q.Set("auth_code", refreshToken)
	redirectUrl.RawQuery = q.Encode()
	return redirectUrl.String()
}
