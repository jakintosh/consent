package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
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
			req = LoginRequest{
				Handle:  r.FormValue("handle"),
				Secret:  r.FormValue("secret"),
				Service: r.FormValue("service"),
			}
			if req.Handle == "" ||
				req.Secret == "" ||
				req.Service == "" {
				wire.WriteError(w, http.StatusBadRequest, "Missing form fields")
				return
			}
		case "application/json":
			if ok := decodeRequest(&req, w, r); !ok {
				return
			}
		default:
			wire.WriteError(w, http.StatusUnsupportedMediaType, "Unsupported content type")
			return
		}

		// run login
		redirectUrl, err := a.service.Login(req.Handle, req.Secret, req.Service)
		if err != nil {
			wire.WriteError(w, httpStatusFromError(err), err.Error())
			return
		}

		// login implemented as redirect 303 with refresh token in URL
		http.Redirect(w, r, redirectUrl.String(), http.StatusSeeOther)
	}
}
