package testing

import (
	"net/http"
)

// HandleDevLogin returns a handler that issues cookies for DefaultTestSubject.
func (tv *TestVerifier) HandleDevLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessToken, err := tv.env.IssueAccessToken(DefaultTestSubject, defaultAccessTokenLifetime)
		if err != nil {
			http.Error(w, "failed to issue access token", http.StatusInternalServerError)
			return
		}
		refreshToken, err := tv.env.IssueRefreshToken(DefaultTestSubject, defaultRefreshTokenLifetime)
		if err != nil {
			http.Error(w, "failed to issue refresh token", http.StatusInternalServerError)
			return
		}

		setTokenCookies(w, accessToken, refreshToken)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// HandleDevLogout returns a handler that clears auth cookies.
func (tv *TestVerifier) HandleDevLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tv.env.ClearTokenCookies(w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
