package service

import (
	"fmt"
	"net/http"
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

func (s *Service) handleMe(w http.ResponseWriter, r *http.Request) {
	accessToken, err := s.accessTokenFromRequest(r)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	if !hasScope(accessToken.Scopes(), ScopeIdentity) {
		wire.WriteError(w, httpStatusFromError(ErrInsufficientScope), ErrInsufficientScope.Error())
		return
	}

	identity, err := s.store.GetIdentityBySubject(accessToken.Subject())
	if err != nil {
		wire.WriteError(w, httpStatusFromError(ErrAccountNotFound), ErrAccountNotFound.Error())
		return
	}

	response := MeResponse{}
	if hasScope(accessToken.Scopes(), ScopeProfile) {
		response.Profile = &MeProfile{Handle: identity.Handle}
	}

	wire.WriteData(w, http.StatusOK, response)
}

func (s *Service) accessTokenFromRequest(r *http.Request) (*tokens.AccessToken, error) {
	encodedToken := bearerToken(r.Header.Get("Authorization"))
	if encodedToken == "" {
		cookie, err := r.Cookie("accessToken")
		if err != nil {
			return nil, ErrTokenInvalid
		}
		encodedToken = cookie.Value
	}

	token := new(tokens.AccessToken)
	if err := token.Decode(encodedToken, s.tokenValidator); err != nil {
		return nil, fmt.Errorf("%w: couldn't decode access token: %v", ErrTokenInvalid, err)
	}
	if !validAudience(token.Audience()) {
		return nil, ErrTokenInvalid
	}

	return token, nil
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func validAudience(audiences []string) bool {
	if len(audiences) == 0 {
		return false
	}
	for _, audience := range audiences {
		if strings.TrimSpace(audience) == "" {
			return false
		}
	}
	return true
}

func hasScope(scopes []string, want string) bool {
	for _, scope := range scopes {
		if scope == want {
			return true
		}
	}
	return false
}
