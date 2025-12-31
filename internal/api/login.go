package api

import (
	"net/http"
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

func (a *API) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// parse request
		var req LoginRequest
		switch r.Header.Get("Content-Type") {
		case "application/x-www-form-urlencoded":
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
		case "application/json":
			if ok := decodeRequest(&req, w, r); !ok {
				return
			}
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		// run login
		redirectUrl, err := a.service.Login(req.Handle, req.Secret, req.Service)
		if err != nil {
			writeError(w, r, err)
			return
		}

		// login implemented as redirect 303 with refresh token in URL
		http.Redirect(w, r, redirectUrl.String(), http.StatusSeeOther)
	}
}
