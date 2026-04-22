package service

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	ScopeIdentity = "identity"
	ScopeProfile  = "profile"
)

// ScopeDefinition describes a registered scope that a service can request.
type ScopeDefinition struct {
	Name        string
	Label       string
	Description string
	Requires    []string
}

// AuthorizationRequest is a validated authorization request for a service.
type AuthorizationRequest struct {
	Service ServiceDefinition
	Scopes  []string
	State   string
}

// AuthorizationReview summarizes a request against the subject's existing grants.
type AuthorizationReview struct {
	Request         AuthorizationRequest
	RequestedScopes []ScopeDefinition
	GrantedScopes   []ScopeDefinition
	MissingScopes   []ScopeDefinition
}

// NeedsApproval reports whether the request includes any scopes not already granted.
func (r AuthorizationReview) NeedsApproval() bool {
	return len(r.MissingScopes) > 0
}

// ReviewAuthorizationRequest validates a request and returns a review of requested,
// granted, and missing scopes for the subject.
func (s *Service) ReviewAuthorizationRequest(
	subject string,
	serviceName string,
	requestedScopes []string,
	state string,
) (
	*AuthorizationReview,
	error,
) {
	serviceDef, err := s.GetServiceByName(serviceName)
	if err != nil {
		return nil, err
	}

	if serviceDef.Name == InternalServiceName {
		return nil, ErrInvalidService
	}

	scopes, err := validateRequestedScopes(requestedScopes)
	if err != nil {
		return nil, err
	}

	grantedScopeNames, err := s.store.ListGrantedScopeNames(subject, serviceDef.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list granted scopes: %v", ErrInternal, err)
	}

	request := &AuthorizationRequest{
		Service: *serviceDef,
		Scopes:  scopes,
		State:   state,
	}

	return &AuthorizationReview{
		Request:         *request,
		RequestedScopes: scopeDefinitions(scopes),
		GrantedScopes:   scopeDefinitions(grantedScopeNames),
		MissingScopes:   missingScopes(scopes, grantedScopeNames),
	}, nil
}

// ApproveAuthorization stores any missing grants and returns an authorization code redirect.
func (s *Service) ApproveAuthorization(
	subject string,
	review *AuthorizationReview,
) (
	*url.URL,
	error,
) {
	missingScopeNames := scopeNames(review.MissingScopes)
	if err := s.store.InsertGrants(
		subject,
		review.Request.Service.Name,
		missingScopeNames,
	); err != nil {
		return nil, fmt.Errorf("%w: failed to store grants: %v", ErrInternal, err)
	}

	return s.issueAuthorizationCodeRedirect(subject, review.Request)
}

// DenyAuthorization returns an access_denied redirect for the reviewed request.
func (s *Service) DenyAuthorization(
	review *AuthorizationReview,
) (
	*url.URL,
	error,
) {
	redirectURL, err := parseAndValidateRedirectURL(review.Request.Service.Redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid redirect URL: %v", ErrInternal, ErrInvalidRedirect)
	}

	return buildAuthorizationErrorRedirectURL(redirectURL, "access_denied", review.Request.State), nil
}

var scopeRegistry = map[string]ScopeDefinition{
	ScopeIdentity: {
		Name:        ScopeIdentity,
		Label:       "Identity",
		Description: "Use your stable Consent account identifier.",
	},
	ScopeProfile: {
		Name:        ScopeProfile,
		Label:       "Profile",
		Description: "Read your profile handle from Consent's user data API.",
		Requires:    []string{ScopeIdentity},
	},
}

// issueAuthorizationCodeRedirect issues a short-lived auth code and builds the callback redirect.
func (s *Service) issueAuthorizationCodeRedirect(
	subject string,
	req AuthorizationRequest,
) (
	*url.URL,
	error,
) {
	refreshToken, err := s.tokenIssuer.IssueRefreshToken(
		subject,
		[]string{req.Service.Audience, s.consentAPIAudience},
		req.Scopes,
		10*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to issue refresh token: %v", ErrInternal, err)
	}

	if err := s.store.InsertRefreshToken(refreshToken); err != nil {
		return nil, fmt.Errorf("%w: failed to store auth code: %v", ErrInternal, err)
	}

	redirectURL, err := parseAndValidateRedirectURL(req.Service.Redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid redirect URL: %v", ErrInternal, ErrInvalidRedirect)
	}

	return buildAuthCodeRedirectURL(redirectURL, refreshToken.Encoded(), req.State, ""), nil
}

func scopeDefinitions(
	scopeNames []string,
) []ScopeDefinition {
	definitions := make([]ScopeDefinition, 0, len(scopeNames))
	for _, name := range scopeNames {
		if definition, ok := scopeRegistry[name]; ok {
			definitions = append(definitions, definition)
		}
	}
	return definitions
}

func scopeNames(
	definitions []ScopeDefinition,
) []string {
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
	}
	return names
}

func validateRequestedScopes(
	requestedScopes []string,
) (
	[]string,
	error,
) {
	if len(requestedScopes) == 0 {
		return nil, ErrMissingScope
	}

	seen := make(map[string]struct{}, len(requestedScopes))
	scopes := make([]string, 0, len(requestedScopes))
	for _, scope := range requestedScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return nil, ErrInvalidScope
		}
		definition, ok := scopeRegistry[scope]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrInvalidScope, scope)
		}
		if _, ok := seen[definition.Name]; ok {
			continue
		}
		seen[definition.Name] = struct{}{}
		scopes = append(scopes, definition.Name)
	}

	if len(scopes) == 0 {
		return nil, ErrMissingScope
	}
	if _, ok := seen[ScopeIdentity]; !ok {
		return nil, ErrIdentityScopeRequired
	}

	for _, scope := range scopes {
		for _, required := range scopeRegistry[scope].Requires {
			if _, ok := seen[required]; !ok {
				return nil, fmt.Errorf("%w: %s requires %s", ErrInvalidScopeDependency, scope, required)
			}
		}
	}

	slices.Sort(scopes)
	return scopes, nil
}

func missingScopes(
	requestedScopes []string,
	grantedScopes []string,
) []ScopeDefinition {
	granted := make(map[string]struct{}, len(grantedScopes))
	for _, scope := range grantedScopes {
		granted[scope] = struct{}{}
	}

	missing := make([]string, 0, len(requestedScopes))
	for _, scope := range requestedScopes {
		if _, ok := granted[scope]; !ok {
			missing = append(missing, scope)
		}
	}

	return scopeDefinitions(missing)
}
