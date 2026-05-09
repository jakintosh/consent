package app

import (
	"errors"
	"net/http"
	"net/url"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/pkg/client"
)

type homePageData struct {
	Authenticated bool
	Handle        string
	Roles         []homeRole
	Integrations  []homeIntegration
	IsAdmin       bool
	LoginURL      string
	LogoutURL     string
}

type homeRole struct {
	Name    string
	Display string
}

type homeIntegration struct {
	Name            string
	Display         string
	Homepage        string
	Logo            string
	Status          string
	GrantedScopes   []service.ScopeDefinition
	UngrantedScopes []service.ScopeDefinition
}

func (a *App) handleGetHome(
	w http.ResponseWriter,
	r *http.Request,
) *appError {
	// get authorization
	accessToken, csrfSecret, err := a.auth.Verifier.VerifyAuthorizationGetCSRF(w, r)
	if err != nil {
		if !errors.Is(err, client.ErrTokenAbsent) {
			logAppErr(r, "failed to verify authorization: "+err.Error())
		}
	}

	// build page data
	var data homePageData
	if accessToken != nil {
		logoutUrl, err := buildLogoutURL(a.auth.LogoutURL, csrfSecret)
		if err != nil {
			return appErr(errHomeSessionUI, err)
		}

		user, err := a.service.GetUser(accessToken.Subject())
		if err != nil {
			logAppErr(r, "failed to get user: "+err.Error())
		} else if user != nil {
			data.Handle = user.Handle
			data.Roles = a.homeRoles(r, user.Roles)
			data.IsAdmin = a.service.UserHasRole(user.Subject, service.ProtectedAdminRoleName)
		}

		grants, err := a.service.ListUserGrants(accessToken.Subject())
		if err != nil {
			logAppErr(r, "failed to list user grants: "+err.Error())
		}
		scopeDefinitions := allHomeScopeDefinitions()

		data = homePageData{
			Authenticated: true,
			Handle:        data.Handle,
			Roles:         data.Roles,
			Integrations:  buildHomeIntegrations(grants, scopeDefinitions),
			IsAdmin:       data.IsAdmin,
			LoginURL:      a.auth.LoginURL,
			LogoutURL:     logoutUrl,
		}
	} else {
		data = homePageData{
			Authenticated: false,
			LoginURL:      a.auth.LoginURL,
		}
	}

	// render page
	a.returnTemplate(w, r, http.StatusOK, "home.html", data)
	return nil
}

func (a *App) homeRoles(
	r *http.Request,
	roleNames []string,
) []homeRole {
	roles := make([]homeRole, 0, len(roleNames))
	for _, roleName := range roleNames {
		role := homeRole{Name: roleName, Display: roleName}
		if definition, err := a.service.GetRole(roleName); err != nil {
			logAppErr(r, "failed to get role "+roleName+": "+err.Error())
		} else if definition.Display != "" {
			role.Display = definition.Display
		}
		roles = append(roles, role)
	}
	return roles
}

func buildHomeIntegrations(
	grants []service.UserGrant,
	scopeDefinitions []service.ScopeDefinition,
) []homeIntegration {
	integrations := make([]homeIntegration, 0, len(grants))
	for _, grant := range grants {
		logo := grant.Logo
		if logo == "" {
			logo = service.DefaultIntegrationLogoPath
		}

		grantedScopes := service.ScopeDefinitions(grant.GrantedScopes)
		ungrantedScopes := ungrantedHomeScopes(scopeDefinitions, grant.GrantedScopes)

		status := "No scopes granted"
		if len(grantedScopes) == len(scopeDefinitions) {
			status = "All scopes granted"
		} else if len(grantedScopes) > 0 {
			status = "Some scopes granted"
		}

		integrations = append(integrations, homeIntegration{
			Name:            grant.Name,
			Display:         grant.Display,
			Homepage:        grant.Homepage,
			Logo:            logo,
			Status:          status,
			GrantedScopes:   grantedScopes,
			UngrantedScopes: ungrantedScopes,
		})
	}
	return integrations
}

func ungrantedHomeScopes(
	scopeDefinitions []service.ScopeDefinition,
	grantedScopeNames []string,
) []service.ScopeDefinition {
	granted := make(map[string]struct{}, len(grantedScopeNames))
	for _, name := range grantedScopeNames {
		granted[name] = struct{}{}
	}

	ungranted := make([]service.ScopeDefinition, 0, len(scopeDefinitions))
	for _, definition := range scopeDefinitions {
		if _, ok := granted[definition.Name]; !ok {
			ungranted = append(ungranted, definition)
		}
	}
	return ungranted
}

func allHomeScopeDefinitions() []service.ScopeDefinition {
	return service.ScopeDefinitions([]string{
		service.ScopeIdentity,
		service.ScopeProfile,
	})
}

func buildLogoutURL(
	logoutURL string,
	csrfSecret string,
) (
	string,
	error,
) {
	parsed, err := url.Parse(logoutURL)
	if err != nil {
		return "", err
	}

	queries := parsed.Query()
	queries.Set("csrf", csrfSecret)
	parsed.RawQuery = queries.Encode()

	return parsed.String(), nil
}
