package api

import (
	"net/http"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

type LoginRequest struct {
	Handle      string `json:"handle"`
	Secret      string `json:"secret"`
	Integration string `json:"integration"`
	ReturnTo    string `json:"returnTo"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

type MeResponse struct {
	Profile *MeProfile `json:"profile,omitempty"`
}

type MeProfile struct {
	Handle string `json:"handle"`
}

func meResponseFromDomain(viewer *service.Viewer) MeResponse {
	response := MeResponse{}
	if viewer != nil && viewer.Profile != nil {
		response.Profile = &MeProfile{Handle: viewer.Profile.Handle}
	}
	return response
}

func (a *API) handleLogin(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req LoginRequest
	switch r.Header.Get("Content-Type") {
	case "application/x-www-form-urlencoded":
		req = LoginRequest{
			Handle:      r.FormValue("handle"),
			Secret:      r.FormValue("secret"),
			Integration: r.FormValue("integration"),
			ReturnTo:    r.FormValue("return_to"),
		}
		if req.Handle == "" || req.Secret == "" || req.Integration == "" {
			wire.WriteError(w, http.StatusBadRequest, "Missing form fields")
			return
		}
	case "application/json":
		var err error
		if req, err = decodeRequest[LoginRequest](r); err != nil {
			wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
			return
		}
	default:
		wire.WriteError(w, http.StatusUnsupportedMediaType, "Unsupported content type")
		return
	}

	redirectURL, err := a.service.Login(req.Handle, req.Secret, req.Integration, req.ReturnTo)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
}

func (a *API) handleLogout(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[LogoutRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = a.service.RevokeRefreshToken(req.RefreshToken)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleRefresh(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[RefreshRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	accessToken, refreshToken, err := a.service.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, RefreshResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	})
}

func (a *API) handleMe(
	w http.ResponseWriter,
	r *http.Request,
) {
	encodedToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		wire.WriteError(w, httpStatusFromError(service.ErrTokenInvalid), service.ErrTokenInvalid.Error())
		return
	}

	viewer, err := a.service.GetViewer(encodedToken)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, meResponseFromDomain(viewer))
}

func bearerToken(
	header string,
) (
	string,
	bool,
) {
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	encodedToken := strings.TrimSpace(parts[1])
	if encodedToken == "" {
		return "", false
	}
	return encodedToken, true
}
