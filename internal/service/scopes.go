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

type ScopeDefinition struct {
	Name        string
	Label       string
	Description string
	Requires    []string
}

type AuthorizationRequest struct {
	Service ServiceDefinition
	Scopes  []string
	State   string
}

type AuthorizationDecision struct {
	Request       AuthorizationRequest
	GrantedScopes []string
	MissingScopes []string
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

func (s *Service) PrepareAuthorizationRequest(serviceName string, requestedScopes []string, state string) (AuthorizationRequest, error) {
	serviceDef, err := s.GetServiceByName(serviceName)
	if err != nil {
		return AuthorizationRequest{}, err
	}
	if serviceDef.Name == InternalServiceName {
		return AuthorizationRequest{}, ErrInvalidService
	}

	scopes, err := validateRequestedScopes(requestedScopes)
	if err != nil {
		return AuthorizationRequest{}, err
	}

	return AuthorizationRequest{
		Service: *serviceDef,
		Scopes:  scopes,
		State:   state,
	}, nil
}

func (s *Service) GetAuthorizationDecision(subject string, req AuthorizationRequest) (AuthorizationDecision, error) {
	grantedScopes, err := s.store.ListGrantedScopeNames(subject, req.Service.Name)
	if err != nil {
		return AuthorizationDecision{}, fmt.Errorf("%w: failed to list granted scopes: %v", ErrInternal, err)
	}

	return AuthorizationDecision{
		Request:       req,
		GrantedScopes: grantedScopes,
		MissingScopes: missingScopes(req.Scopes, grantedScopes),
	}, nil
}

func (s *Service) ApproveAuthorization(subject string, decision AuthorizationDecision) (*url.URL, error) {
	if err := s.store.InsertGrants(subject, decision.Request.Service.Name, decision.MissingScopes); err != nil {
		return nil, fmt.Errorf("%w: failed to store grants: %v", ErrInternal, err)
	}

	return s.IssueAuthorizationCodeRedirect(subject, decision.Request.Service.Name, decision.Request.Scopes, decision.Request.State)
}

func (s *Service) IssueAuthorizationCodeRedirect(subject, serviceName string, requestedScopes []string, state string) (*url.URL, error) {
	serviceDef, err := s.GetServiceByName(serviceName)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenIssuer.IssueRefreshToken(
		subject,
		[]string{serviceDef.Audience},
		requestedScopes,
		10*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to issue refresh token: %v", ErrInternal, err)
	}

	if err := s.store.InsertRefreshToken(refreshToken); err != nil {
		return nil, fmt.Errorf("%w: failed to store auth code: %v", ErrInternal, err)
	}

	redirectURL, err := parseFullURL(serviceDef.Redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid redirect URL: %v", ErrInternal, ErrInvalidRedirect)
	}

	return buildAuthCodeRedirectURL(redirectURL, refreshToken.Encoded(), state, ""), nil
}

func (s *Service) IssueAuthorizationDeniedRedirect(serviceName, state string) (*url.URL, error) {
	serviceDef, err := s.GetServiceByName(serviceName)
	if err != nil {
		return nil, err
	}

	redirectURL, err := parseFullURL(serviceDef.Redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid redirect URL: %v", ErrInternal, ErrInvalidRedirect)
	}

	return buildAuthorizationErrorRedirectURL(redirectURL, "access_denied", state), nil
}

func ScopeDefinitions(scopeNames []string) []ScopeDefinition {
	definitions := make([]ScopeDefinition, 0, len(scopeNames))
	for _, name := range scopeNames {
		if definition, ok := scopeRegistry[name]; ok {
			definitions = append(definitions, definition)
		}
	}
	return definitions
}

func validateRequestedScopes(requestedScopes []string) ([]string, error) {
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

func missingScopes(requestedScopes []string, grantedScopes []string) []string {
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

	return missing
}
