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

type UserInfo struct {
	Sub     string           `json:"sub"`
	Profile *UserInfoProfile `json:"profile,omitempty"`
}

type UserInfoProfile struct {
	Handle string `json:"handle"`
}

func userInfoFromDomain(
	userInfo *service.UserInfo,
) UserInfo {
	response := UserInfo{}
	if userInfo != nil {
		response.Sub = userInfo.Sub
	}
	if userInfo != nil && userInfo.Profile != nil {
		response.Profile = &UserInfoProfile{Handle: userInfo.Profile.Handle}
	}
	return response
}

func (a *API) buildAuthRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", a.handleLogin)
	mux.HandleFunc("POST /logout", a.handleLogout)
	mux.HandleFunc("POST /refresh", a.handleRefresh)
	mux.HandleFunc("GET  /userinfo", a.handleUserInfo)

	return mux
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

	redirectURL, err := a.service.GrantAuthCode(req.Handle, req.Secret, req.Integration, req.ReturnTo)
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

func (a *API) handleUserInfo(
	w http.ResponseWriter,
	r *http.Request,
) {
	authHeader := r.Header.Get("Authorization")
	encodedToken, ok := parseBearerToken(authHeader)
	if !ok {
		wire.WriteError(w, httpStatusFromError(service.ErrTokenInvalid), service.ErrTokenInvalid.Error())
		return
	}

	userInfo, err := a.service.GetUserInfo(encodedToken)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, userInfoFromDomain(userInfo))
}

func parseBearerToken(
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
