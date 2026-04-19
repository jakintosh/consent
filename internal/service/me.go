package service

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

type MeResponse struct {
	Profile *MeProfile `json:"profile,omitempty"`
}

type MeProfile struct {
	Handle string `json:"handle"`
}

func (s *Service) handleMe(
	w http.ResponseWriter,
	r *http.Request,
) {
	encodedToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		wire.WriteError(w, httpStatusFromError(ErrTokenInvalid), ErrTokenInvalid.Error())
		return
	}

	accessToken := new(tokens.AccessToken)
	if err := accessToken.Decode(encodedToken, s.resourceTokenValidator); err != nil {
		wire.WriteError(w, httpStatusFromError(ErrTokenInvalid), fmt.Sprintf("%v: couldn't decode access token: %v", ErrTokenInvalid, err))
	}

	if !slices.Contains(accessToken.Scopes(), ScopeIdentity) {
		wire.WriteError(w, httpStatusFromError(ErrInsufficientScope), ErrInsufficientScope.Error())
		return
	}

	identity, err := s.store.GetIdentityBySubject(accessToken.Subject())
	if err != nil {
		wire.WriteError(w, httpStatusFromError(ErrAccountNotFound), ErrAccountNotFound.Error())
		return
	}

	response := MeResponse{}
	if slices.Contains(accessToken.Scopes(), ScopeProfile) {
		response.Profile = &MeProfile{Handle: identity.Handle}
	}

	wire.WriteData(w, http.StatusOK, response)
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
