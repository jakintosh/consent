package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"golang.org/x/crypto/bcrypt"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

type UserInfoProfile struct {
	Handle string
}

type UserInfo struct {
	Sub     string
	Profile *UserInfoProfile
}

type UserGrant struct {
	Name          string
	Display       string
	Homepage      string
	Logo          string
	GrantedScopes []string
}

// AuthorizationRequest is a validated authorization request for an integration.
type AuthorizationRequest struct {
	Integration Integration
	Scopes      []string
	State       string
}

// AuthorizationReview summarizes a request against the subject's existing grants.
type AuthorizationReview struct {
	Request         AuthorizationRequest
	RequestedScopes []ScopeDefinition
	GrantedScopes   []ScopeDefinition
	MissingScopes   []ScopeDefinition
}

// IsAuthorized reports whether the request includes any scopes not already granted.
func (r AuthorizationReview) IsAuthorized() bool {
	return len(r.MissingScopes) == 0
}

func (s *Service) GetUserInfo(
	encodedAccessToken string,
) (
	*UserInfo,
	error,
) {
	accessToken := new(tokens.AccessToken)
	if err := accessToken.Decode(encodedAccessToken, s.resourceTokenValidator); err != nil {
		return nil, fmt.Errorf("%w: couldn't decode access token: %v", ErrTokenInvalid, err)
	}

	if !slices.Contains(accessToken.Scopes(), ScopeIdentity) {
		return nil, ErrInsufficientScope
	}

	user, err := s.store.GetUserBySubject(accessToken.Subject())
	if err != nil {
		return nil, ErrAccountNotFound
	}

	userInfo := &UserInfo{Sub: accessToken.Subject()}
	if slices.Contains(accessToken.Scopes(), ScopeProfile) {
		userInfo.Profile = &UserInfoProfile{
			Handle: user.Handle,
		}
	}

	return userInfo, nil
}

// IntegrationsAccessibleTo returns integrations the user is allowed to see (has at least one required role).
func (s *Service) IntegrationsAccessibleTo(
	subject string,
) (
	[]Integration,
	error,
) {
	integrations, err := s.ListIntegrations()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list integrations: %v", ErrInternal, err)
	}

	accessible := make([]Integration, 0, len(integrations))
	for _, integration := range integrations {
		if s.UserHasAnyRole(subject, integration.RequiredRoles) {
			accessible = append(accessible, integration)
		}
	}

	return accessible, nil
}

// ListUserGrants returns all integrations registered with the server and the
// scopes the given subject has granted to each.
func (s *Service) ListUserGrants(
	subject string,
) (
	[]UserGrant,
	error,
) {
	integrations, err := s.IntegrationsAccessibleTo(subject)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list integrations: %v", ErrInternal, err)
	}

	grants := make([]UserGrant, 0, len(integrations))
	for _, integration := range integrations {
		if integration.Name == InternalIntegrationName {
			continue
		}

		granted, err := s.store.ListGrantedScopeNames(subject, integration.Name)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to list grants for %q: %v", ErrInternal, integration.Name, err)
		}

		grants = append(grants, UserGrant{
			Name:          integration.Name,
			Display:       integration.Display,
			Homepage:      integration.Homepage,
			Logo:          integration.Logo,
			GrantedScopes: granted,
		})
	}

	return grants, nil
}

func (s *Service) Login(
	handle string,
	secret string,
	integrationName string,
	returnTo ...string,
) (
	*url.URL,
	error,
) {
	redirectReturnTo := ""
	if len(returnTo) > 0 {
		redirectReturnTo = returnTo[0]
	}

	secretHash, err := s.store.GetSecret(handle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, handle)
		}
		return nil, fmt.Errorf("%w: failed to retrieve secret: %v", ErrInternal, err)
	}

	err = bcrypt.CompareHashAndPassword(secretHash, []byte(secret))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.store.GetUserByHandle(handle)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, handle)
	}

	integration, err := s.GetIntegration(integrationName)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrIntegrationNotFound, integrationName)
	}

	if integrationName != InternalIntegrationName {
		return nil, ErrInvalidIntegration
	}

	refreshToken, err := s.tokenIssuer.IssueRefreshToken(
		user.Subject,
		[]string{integration.Audience},
		nil,
		time.Second*10,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to issue refresh token: %v", ErrInternal, err)
	}

	err = s.store.InsertRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return buildAuthCodeRedirectURL(
		integration.Redirect,
		refreshToken.Encoded(),
		"",
		redirectReturnTo,
	)
}

// ReviewAuthorizationRequest validates a request and returns a review of requested,
// granted, and missing scopes for the subject.
func (s *Service) ReviewAuthorizationRequest(
	subject string,
	integrationName string,
	requestedScopes []string,
	state string,
) (
	*AuthorizationReview,
	error,
) {
	integration, err := s.GetIntegration(integrationName)
	if err != nil {
		return nil, err
	}

	if integration.Name == InternalIntegrationName {
		return nil, ErrInvalidIntegration
	}

	if !s.UserHasAnyRole(subject, integration.RequiredRoles) {
		return nil, ErrAccessDenied
	}

	scopes, err := validateRequestedScopes(requestedScopes)
	if err != nil {
		return nil, err
	}

	grantedScopeNames, err := s.store.ListGrantedScopeNames(subject, integration.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list granted scopes: %v", ErrInternal, err)
	}

	request := &AuthorizationRequest{
		Integration: *integration,
		Scopes:      scopes,
		State:       state,
	}

	return &AuthorizationReview{
		Request:         *request,
		RequestedScopes: ScopeDefinitions(scopes),
		GrantedScopes:   ScopeDefinitions(grantedScopeNames),
		MissingScopes:   missingScopes(scopes, grantedScopeNames),
	}, nil
}

// FinalizeAuthorization stores any missing grants and returns an authorization code redirect.
func (s *Service) FinalizeAuthorization(
	subject string,
	review *AuthorizationReview,
) (
	*url.URL,
	error,
) {
	missingScopeNames := scopeNames(review.MissingScopes)
	if err := s.store.InsertGrants(
		subject,
		review.Request.Integration.Name,
		missingScopeNames,
	); err != nil {
		return nil, fmt.Errorf("%w: failed to store grants: %v", ErrInternal, err)
	}

	refreshToken, err := s.tokenIssuer.IssueRefreshToken(
		subject,
		[]string{
			s.consentAPIAudience,
			review.Request.Integration.Audience,
		},
		review.Request.Scopes,
		10*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to issue refresh token: %v", ErrInternal, err)
	}

	if err := s.store.InsertRefreshToken(refreshToken); err != nil {
		return nil, fmt.Errorf("%w: failed to store auth code: %v", ErrInternal, err)
	}

	return buildAuthCodeRedirectURL(
		review.Request.Integration.Redirect,
		refreshToken.Encoded(),
		review.Request.State,
		"",
	)
}

// DenyAuthorization returns an access_denied redirect for the reviewed request.
func (s *Service) DenyAuthorization(
	review *AuthorizationReview,
) (
	*url.URL,
	error,
) {
	return buildAuthorizationErrorRedirectURL(
		review.Request.Integration.Redirect,
		"access_denied",
		review.Request.State,
	)
}

func (s *Service) RevokeRefreshToken(
	encodedRefreshToken string,
) error {
	deleted, err := s.store.DeleteRefreshToken(encodedRefreshToken)
	if err != nil {
		return fmt.Errorf("%w: failed to delete refresh token: %v", ErrInternal, err)
	}
	if !deleted {
		return ErrTokenNotFound
	}
	return nil
}

func (s *Service) RefreshAccessToken(
	encodedRefreshToken string,
) (
	string,
	string,
	error,
) {
	token := tokens.RefreshToken{}
	if err := token.Decode(encodedRefreshToken, s.tokenValidator); err != nil {
		return "", "", fmt.Errorf("%w: couldn't decode refresh token: %v", ErrTokenInvalid, err)
	}

	deleted, err := s.store.DeleteRefreshToken(encodedRefreshToken)
	if err != nil {
		return "", "", fmt.Errorf("%w: refresh token couldn't be deleted: %v", ErrInternal, err)
	}
	if !deleted {
		return "", "", ErrTokenNotFound
	}

	if err := s.validateRefreshTokenIntegrationAccess(token.Subject(), token.Audience()); err != nil {
		return "", "", err
	}

	accessToken, err := s.tokenIssuer.IssueAccessToken(
		token.Subject(),
		token.Audience(),
		token.Scopes(),
		time.Minute*30,
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: couldn't issue access token: %v", ErrInternal, err)
	}

	newRefreshToken, err := s.tokenIssuer.IssueRefreshToken(
		token.Subject(),
		token.Audience(),
		token.Scopes(),
		time.Hour*72,
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: couldn't issue refresh token: %v", ErrInternal, err)
	}

	err = s.store.InsertRefreshToken(newRefreshToken)
	if err != nil {
		return "", "", fmt.Errorf("%w: failed to store refresh token: %v", ErrInternal, err)
	}

	return accessToken.Encoded(), newRefreshToken.Encoded(), nil
}

func (s *Service) validateRefreshTokenIntegrationAccess(
	subject string,
	audiences []string,
) error {
	for _, audience := range audiences {
		if audience == s.consentAPIAudience {
			continue
		}

		integration, err := s.store.GetIntegrationByAudience(audience)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return fmt.Errorf("%w: failed to get integration by audience: %v", ErrInternal, err)
		}
		if integration.Name == InternalIntegrationName {
			continue
		}
		if !s.UserHasAnyRole(subject, integration.RequiredRoles) {
			return ErrAccessDenied
		}
	}

	return nil
}
