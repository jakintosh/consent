package service

import (
	"fmt"
	"slices"
	"strings"
)

const (
	ScopeIdentity = "identity"
	ScopeProfile  = "profile"
)

// ScopeDefinition describes a registered scope that an integration can request.
type ScopeDefinition struct {
	Name        string
	Label       string
	Description string
	Requires    []string
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

func ScopeDefinitions(
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

	return ScopeDefinitions(missing)
}
