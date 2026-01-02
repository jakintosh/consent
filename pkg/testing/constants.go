package testing

import "time"

// DefaultTestSubject is the default user identity for dev/test flows.
const DefaultTestSubject = "alice"

const (
	accessTokenCookieName       = "accessToken"
	refreshTokenCookieName      = "refreshToken"
	defaultCookiePath           = "/"
	defaultAccessTokenLifetime  = 30 * time.Minute
	defaultRefreshTokenLifetime = 24 * time.Hour
)
