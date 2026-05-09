package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/service"
	consentclient "git.sr.ht/~jakintosh/consent/pkg/client"
)

type adminBaseData struct {
	Handle    string
	CSRF      string
	LogoutURL string
	Current   string
	Error     string
}

type adminOverviewData struct {
	adminBaseData
}

type adminUsersData struct {
	adminBaseData
	Users []service.User
}

type adminUserFormData struct {
	adminBaseData
	User  service.User
	Roles []adminRoleChoice
}

type adminRolesData struct {
	adminBaseData
	Roles []adminRoleRow
}

type adminRoleFormData struct {
	adminBaseData
	Role      service.Role
	New       bool
	SaveURL   string
	DeleteURL string
}

type adminIntegrationsData struct {
	adminBaseData
	Integrations []adminIntegrationRow
}

type adminIntegrationFormData struct {
	adminBaseData
	Integration service.Integration
	Roles       []adminRoleChoice
	New         bool
	BaseURL     string
	SaveURL     string
	DeleteURL   string
}

type adminRoleRow struct {
	Name    string
	Display string
	EditURL string
}

type adminIntegrationRow struct {
	Name          string
	Display       string
	Audience      string
	RequiredRoles []string
	EditURL       string
}

type adminRoleChoice struct {
	Name    string
	Display string
	Checked bool
}

func (a *App) handleGetAdmin(
	w http.ResponseWriter,
	r *http.Request,
) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}

	a.returnTemplate(w, r, http.StatusOK, "admin.html", adminOverviewData{adminBaseData: base})
	return nil
}

func (a *App) handleListAdminUsers(w http.ResponseWriter, r *http.Request) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	users, listErr := a.service.ListUsers()
	if listErr != nil {
		return appErr(errAdminSessionUI, listErr)
	}
	a.returnTemplate(w, r, http.StatusOK, "admin_users.html", adminUsersData{adminBaseData: base, Users: users})
	return nil
}

func (a *App) handleEditAdminUser(w http.ResponseWriter, r *http.Request) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	data, loadErr := a.adminUserForm(base, r.PathValue("subject"))
	if loadErr != nil {
		return appErr(errAdminSessionUI, loadErr)
	}
	a.returnTemplate(w, r, http.StatusOK, "admin_user_form.html", data)
	return nil
}

func (a *App) handleUpdateAdminUser(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}

	subject := r.PathValue("subject")
	handle := strings.TrimSpace(r.FormValue("handle"))
	roles := r.Form["roles"]
	_, err := a.service.UpdateUser(subject, &service.UserUpdate{Handle: &handle, Roles: &roles})
	if err != nil {
		base, ok, baseErr := a.adminBase(w, r)
		if baseErr != nil || !ok {
			return baseErr
		}
		data, loadErr := a.adminUserForm(base, subject)
		if loadErr != nil {
			return appErr(errAdminActionFailed, loadErr)
		}
		data.User.Handle = handle
		data.Roles = checkedRoleChoices(data.Roles, roles)
		data.Error = err.Error()
		a.returnTemplate(w, r, http.StatusBadRequest, "admin_user_form.html", data)
		return nil
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
	return nil
}

func (a *App) handleDeleteAdminUser(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}
	if err := a.service.DeleteUser(r.PathValue("subject")); err != nil {
		return appErr(errAdminActionFailed, err)
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
	return nil
}

func (a *App) handleListAdminRoles(w http.ResponseWriter, r *http.Request) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	roles, listErr := a.service.ListRoles()
	if listErr != nil {
		return appErr(errAdminSessionUI, listErr)
	}
	a.returnTemplate(w, r, http.StatusOK, "admin_roles.html", adminRolesData{adminBaseData: base, Roles: adminRoleRows(roles)})
	return nil
}

func (a *App) handleNewAdminRole(w http.ResponseWriter, r *http.Request) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	a.returnTemplate(w, r, http.StatusOK, "admin_role_form.html", adminRoleFormData{adminBaseData: base, New: true, SaveURL: "/admin/roles"})
	return nil
}

func (a *App) handleCreateAdminRole(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}
	role, err := a.service.CreateRole(strings.TrimSpace(r.FormValue("name")), strings.TrimSpace(r.FormValue("display")))
	if err != nil {
		return a.renderRoleFormError(w, r, service.Role{Name: r.FormValue("name"), Display: r.FormValue("display")}, true, err)
	}
	http.Redirect(w, r, "/admin/roles/"+url.PathEscape(role.Name)+"/edit", http.StatusSeeOther)
	return nil
}

func (a *App) handleEditAdminRole(w http.ResponseWriter, r *http.Request) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	role, getErr := a.service.GetRole(r.PathValue("name"))
	if getErr != nil {
		return appErr(errAdminSessionUI, getErr)
	}
	a.returnTemplate(w, r, http.StatusOK, "admin_role_form.html", adminRoleFormData{adminBaseData: base, Role: *role, SaveURL: adminRoleURL(role.Name), DeleteURL: adminRoleURL(role.Name) + "/delete"})
	return nil
}

func (a *App) handleUpdateAdminRole(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}
	display := strings.TrimSpace(r.FormValue("display"))
	role, err := a.service.UpdateRole(r.PathValue("name"), &display)
	if err != nil {
		return a.renderRoleFormError(w, r, service.Role{Name: r.PathValue("name"), Display: display}, false, err)
	}
	http.Redirect(w, r, "/admin/roles/"+url.PathEscape(role.Name)+"/edit", http.StatusSeeOther)
	return nil
}

func (a *App) handleDeleteAdminRole(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}
	if err := a.service.DeleteRole(r.PathValue("name")); err != nil {
		return appErr(errAdminActionFailed, err)
	}
	http.Redirect(w, r, "/admin/roles", http.StatusSeeOther)
	return nil
}

func (a *App) handleListAdminIntegrations(w http.ResponseWriter, r *http.Request) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	integrations, listErr := a.service.ListIntegrations()
	if listErr != nil {
		return appErr(errAdminSessionUI, listErr)
	}
	a.returnTemplate(w, r, http.StatusOK, "admin_integrations.html", adminIntegrationsData{adminBaseData: base, Integrations: adminIntegrationRows(nonInternalIntegrations(integrations))})
	return nil
}

func (a *App) handleNewAdminIntegration(w http.ResponseWriter, r *http.Request) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	data, loadErr := a.adminIntegrationForm(base, service.Integration{}, true)
	if loadErr != nil {
		return appErr(errAdminSessionUI, loadErr)
	}
	a.returnTemplate(w, r, http.StatusOK, "admin_integration_form.html", data)
	return nil
}

func (a *App) handleImportAdminIntegration(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}

	baseURL := strings.TrimSpace(r.FormValue("base_url"))
	manifest, importErr := fetchIntegrationManifest(baseURL)
	data, loadErr := a.adminIntegrationForm(base, manifest, true)
	if loadErr != nil {
		return appErr(errAdminSessionUI, loadErr)
	}
	data.BaseURL = baseURL
	if importErr != nil {
		data.Error = importErr.Error()
	}
	status := http.StatusOK
	if importErr != nil {
		status = http.StatusBadRequest
	}
	a.returnTemplate(w, r, status, "admin_integration_form.html", data)
	return nil
}

func (a *App) handleCreateAdminIntegration(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}
	integration := integrationFromForm(r)
	if err := a.service.CreateIntegration(integration.Name, integration.Display, integration.Audience, integration.Redirect, integration.Homepage, integration.Logo, integration.RequiredRoles); err != nil {
		return a.renderIntegrationFormError(w, r, integration, true, err)
	}
	http.Redirect(w, r, "/admin/integrations/"+url.PathEscape(integration.Name)+"/edit", http.StatusSeeOther)
	return nil
}

func (a *App) handleEditAdminIntegration(w http.ResponseWriter, r *http.Request) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	integration, getErr := a.service.GetIntegration(r.PathValue("name"))
	if getErr != nil {
		return appErr(errAdminSessionUI, getErr)
	}
	data, loadErr := a.adminIntegrationForm(base, *integration, false)
	if loadErr != nil {
		return appErr(errAdminSessionUI, loadErr)
	}
	a.returnTemplate(w, r, http.StatusOK, "admin_integration_form.html", data)
	return nil
}

func (a *App) handleUpdateAdminIntegration(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}
	integration := integrationFromForm(r)
	updates := &service.IntegrationUpdate{
		Display:       &integration.Display,
		Audience:      &integration.Audience,
		Redirect:      &integration.Redirect,
		Homepage:      &integration.Homepage,
		Logo:          &integration.Logo,
		RequiredRoles: &integration.RequiredRoles,
	}
	if err := a.service.UpdateIntegration(r.PathValue("name"), updates); err != nil {
		integration.Name = r.PathValue("name")
		return a.renderIntegrationFormError(w, r, integration, false, err)
	}
	http.Redirect(w, r, "/admin/integrations/"+url.PathEscape(r.PathValue("name"))+"/edit", http.StatusSeeOther)
	return nil
}

func (a *App) handleDeleteAdminIntegration(w http.ResponseWriter, r *http.Request) *appError {
	if err := r.ParseForm(); err != nil {
		return appErr(errAdminFormInvalid, err)
	}
	if _, ok, err := a.requireAdminPost(w, r); err != nil || !ok {
		return err
	}
	if err := a.service.DeleteIntegration(r.PathValue("name")); err != nil {
		return appErr(errAdminActionFailed, err)
	}
	http.Redirect(w, r, "/admin/integrations", http.StatusSeeOther)
	return nil
}

func (a *App) adminBase(w http.ResponseWriter, r *http.Request) (adminBaseData, bool, *appError) {
	accessToken, csrf, err := a.auth.Verifier.VerifyAuthorizationGetCSRF(w, r)
	if err != nil {
		http.Redirect(w, r, a.auth.LoginURL, http.StatusSeeOther)
		return adminBaseData{}, false, nil
	}
	if !a.service.UserHasRole(accessToken.Subject(), service.ProtectedAdminRoleName) {
		return adminBaseData{}, false, appErr(errAdminForbidden, nil)
	}

	user, err := a.service.GetUser(accessToken.Subject())
	if err != nil {
		return adminBaseData{}, false, appErr(errAdminSessionUI, err)
	}
	logoutURL, err := buildLogoutURL(a.auth.LogoutURL, csrf)
	if err != nil {
		return adminBaseData{}, false, appErr(errAdminSessionUI, err)
	}

	return adminBaseData{
		Handle:    user.Handle,
		CSRF:      csrf,
		LogoutURL: logoutURL,
		Current:   adminCurrentSection(r.URL.Path),
	}, true, nil
}

func (a *App) requireAdminPost(w http.ResponseWriter, r *http.Request) (string, bool, *appError) {
	accessToken, csrf, err := a.auth.Verifier.VerifyAuthorizationCheckCSRF(w, r, r.FormValue("csrf"))
	if err != nil {
		return "", false, appErr(errAdminCSRFExpired, err)
	}
	if !a.service.UserHasRole(accessToken.Subject(), service.ProtectedAdminRoleName) {
		return "", false, appErr(errAdminForbidden, nil)
	}
	return csrf, true, nil
}

func (a *App) adminUserForm(base adminBaseData, subject string) (adminUserFormData, error) {
	user, err := a.service.GetUser(subject)
	if err != nil {
		return adminUserFormData{}, err
	}
	roles, err := a.service.ListRoles()
	if err != nil {
		return adminUserFormData{}, err
	}
	return adminUserFormData{
		adminBaseData: base,
		User:          *user,
		Roles:         roleChoices(roles, user.Roles),
	}, nil
}

func (a *App) adminIntegrationForm(base adminBaseData, integration service.Integration, new bool) (adminIntegrationFormData, error) {
	roles, err := a.service.ListRoles()
	if err != nil {
		return adminIntegrationFormData{}, err
	}
	return adminIntegrationFormData{
		adminBaseData: base,
		Integration:   integration,
		Roles:         roleChoices(roles, integration.RequiredRoles),
		New:           new,
		SaveURL:       adminIntegrationFormSaveURL(integration.Name, new),
		DeleteURL:     adminIntegrationFormDeleteURL(integration.Name, new),
	}, nil
}

func (a *App) renderRoleFormError(w http.ResponseWriter, r *http.Request, role service.Role, new bool, cause error) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	data := adminRoleFormData{adminBaseData: base, Role: role, New: new, SaveURL: "/admin/roles"}
	if !new {
		data.SaveURL = adminRoleURL(role.Name)
		data.DeleteURL = adminRoleURL(role.Name) + "/delete"
	}
	data.Error = cause.Error()
	a.returnTemplate(w, r, http.StatusBadRequest, "admin_role_form.html", data)
	return nil
}

func (a *App) renderIntegrationFormError(w http.ResponseWriter, r *http.Request, integration service.Integration, new bool, cause error) *appError {
	base, ok, err := a.adminBase(w, r)
	if err != nil || !ok {
		return err
	}
	data, loadErr := a.adminIntegrationForm(base, integration, new)
	if loadErr != nil {
		return appErr(errAdminActionFailed, loadErr)
	}
	data.Roles = checkedRoleChoices(data.Roles, integration.RequiredRoles)
	data.Error = cause.Error()
	a.returnTemplate(w, r, http.StatusBadRequest, "admin_integration_form.html", data)
	return nil
}

func roleChoices(roles []service.Role, selected []string) []adminRoleChoice {
	selectedSet := map[string]bool{}
	for _, name := range selected {
		selectedSet[name] = true
	}
	choices := make([]adminRoleChoice, 0, len(roles))
	for _, role := range roles {
		choices = append(choices, adminRoleChoice{
			Name:    role.Name,
			Display: role.Display,
			Checked: selectedSet[role.Name],
		})
	}
	return choices
}

func adminRoleRows(roles []service.Role) []adminRoleRow {
	rows := make([]adminRoleRow, 0, len(roles))
	for _, role := range roles {
		rows = append(rows, adminRoleRow{
			Name:    role.Name,
			Display: role.Display,
			EditURL: adminRoleURL(role.Name) + "/edit",
		})
	}
	return rows
}

func adminIntegrationRows(integrations []service.Integration) []adminIntegrationRow {
	rows := make([]adminIntegrationRow, 0, len(integrations))
	for _, integration := range integrations {
		rows = append(rows, adminIntegrationRow{
			Name:          integration.Name,
			Display:       integration.Display,
			Audience:      integration.Audience,
			RequiredRoles: integration.RequiredRoles,
			EditURL:       adminIntegrationURL(integration.Name) + "/edit",
		})
	}
	return rows
}

func adminRoleURL(name string) string {
	return "/admin/roles/" + url.PathEscape(name)
}

func adminIntegrationURL(name string) string {
	return "/admin/integrations/" + url.PathEscape(name)
}

func adminIntegrationFormSaveURL(name string, new bool) string {
	if new {
		return "/admin/integrations"
	}
	return adminIntegrationURL(name)
}

func adminIntegrationFormDeleteURL(name string, new bool) string {
	if new {
		return ""
	}
	return adminIntegrationURL(name) + "/delete"
}

func checkedRoleChoices(choices []adminRoleChoice, selected []string) []adminRoleChoice {
	selectedSet := map[string]bool{}
	for _, name := range selected {
		selectedSet[name] = true
	}
	for i := range choices {
		choices[i].Checked = selectedSet[choices[i].Name]
	}
	return choices
}

func integrationFromForm(r *http.Request) service.Integration {
	return service.Integration{
		Name:          strings.TrimSpace(r.FormValue("name")),
		Display:       strings.TrimSpace(r.FormValue("display")),
		Audience:      strings.TrimSpace(r.FormValue("audience")),
		Redirect:      strings.TrimSpace(r.FormValue("redirect")),
		Homepage:      strings.TrimSpace(r.FormValue("homepage")),
		Logo:          strings.TrimSpace(r.FormValue("logo")),
		RequiredRoles: r.Form["required_roles"],
	}
}

func fetchIntegrationManifest(baseURL string) (service.Integration, error) {
	manifestURL, err := integrationManifestURL(baseURL)
	if err != nil {
		return service.Integration{}, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(manifestURL)
	if err != nil {
		return service.Integration{}, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return service.Integration{}, fmt.Errorf("manifest returned %s", resp.Status)
	}

	var manifest consentclient.IntegrationManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return service.Integration{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := validateIntegrationManifest(manifest); err != nil {
		return service.Integration{}, err
	}

	return service.Integration{
		Name:     manifest.Name,
		Display:  manifest.Display,
		Audience: manifest.Audience,
		Redirect: manifest.Redirect,
		Homepage: manifest.Homepage,
		Logo:     manifest.Logo,
	}, nil
}

func integrationManifestURL(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	if parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("base URL must include scheme and host")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("base URL must be the integration root")
	}
	parsed.Path = consentclient.IntegrationManifestPath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func validateIntegrationManifest(manifest consentclient.IntegrationManifest) error {
	if manifest.ManifestVersion != consentclient.IntegrationManifestVersion {
		return fmt.Errorf("unsupported manifest version %d", manifest.ManifestVersion)
	}
	if manifest.Name == "" || manifest.Display == "" || manifest.Audience == "" || manifest.Redirect == "" || manifest.Homepage == "" || manifest.Logo == "" {
		return errors.New("manifest is missing required fields")
	}
	if manifest.Name == service.InternalIntegrationName {
		return service.ErrIntegrationProtected
	}

	redirect, err := url.Parse(manifest.Redirect)
	if err != nil || redirect == nil || redirect.Scheme == "" || redirect.Host == "" {
		return service.ErrInvalidRedirect
	}
	if redirect.Host != manifest.Audience {
		return errors.New("redirect host must match audience")
	}
	homepage, err := url.Parse(manifest.Homepage)
	if err != nil || homepage == nil || homepage.Scheme == "" || homepage.Host == "" {
		return errors.New("invalid homepage URL")
	}
	if homepage.Host != manifest.Audience {
		return errors.New("homepage host must match audience")
	}
	logo, err := url.Parse(manifest.Logo)
	if err != nil || logo == nil || logo.Scheme == "" || logo.Host == "" {
		return errors.New("invalid logo URL")
	}
	return nil
}

func nonInternalIntegrations(integrations []service.Integration) []service.Integration {
	filtered := make([]service.Integration, 0, len(integrations))
	for _, integration := range integrations {
		if integration.Name != service.InternalIntegrationName {
			filtered = append(filtered, integration)
		}
	}
	return filtered
}

func adminCurrentSection(path string) string {
	switch {
	case strings.HasPrefix(path, "/admin/users"):
		return "users"
	case strings.HasPrefix(path, "/admin/roles"):
		return "roles"
	case strings.HasPrefix(path, "/admin/integrations"):
		return "integrations"
	default:
		return "dashboard"
	}
}
