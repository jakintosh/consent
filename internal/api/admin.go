package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

type Integration struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

type UpdateIntegrationRequest struct {
	Display  *string `json:"display,omitempty"`
	Audience *string `json:"audience,omitempty"`
	Redirect *string `json:"redirect,omitempty"`
}

func integrationFromDomain(integration service.Integration) Integration {
	return Integration{
		Name:     integration.Name,
		Display:  integration.Display,
		Audience: integration.Audience,
		Redirect: integration.Redirect,
	}
}

func integrationsFromDomain(integrations []service.Integration) []Integration {
	apiIntegrations := make([]Integration, 0, len(integrations))
	for _, integration := range integrations {
		apiIntegrations = append(apiIntegrations, integrationFromDomain(integration))
	}
	return apiIntegrations
}

func (a *API) handleCreateIntegration(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[Integration](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = a.service.CreateIntegration(req.Name, req.Display, req.Audience, req.Redirect)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleGetIntegration(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing integration name")
		return
	}

	integration, err := a.service.GetIntegration(name)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, integrationFromDomain(*integration))
}

func (a *API) handleUpdateIntegration(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing integration name")
		return
	}

	req, err := decodeRequest[UpdateIntegrationRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = a.service.UpdateIntegration(name, &service.IntegrationUpdate{
		Display:  req.Display,
		Audience: req.Audience,
		Redirect: req.Redirect,
	})
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleDeleteIntegration(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing integration name")
		return
	}

	err := a.service.DeleteIntegration(name)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleListIntegrations(
	w http.ResponseWriter,
	r *http.Request,
) {
	integrations, err := a.service.ListIntegrations()
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, integrationsFromDomain(integrations))
}

type User struct {
	Subject string   `json:"subject"`
	Handle  string   `json:"username"`
	Roles   []string `json:"roles"`
}

type CreateUserRequest struct {
	Handle   string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

type UpdateUserRequest struct {
	Handle *string   `json:"username,omitempty"`
	Roles  *[]string `json:"roles,omitempty"`
}

func userFromDomain(user service.User) User {
	return User{
		Subject: user.Subject,
		Handle:  user.Handle,
		Roles:   append([]string(nil), user.Roles...),
	}
}

func usersFromDomain(users []service.User) []User {
	apiUsers := make([]User, 0, len(users))
	for _, user := range users {
		apiUsers = append(apiUsers, userFromDomain(user))
	}
	return apiUsers
}

func (a *API) handleCreateUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[CreateUserRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	user, err := a.service.CreateUser(req.Handle, req.Password, req.Roles)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, userFromDomain(*user))
}

func (a *API) handleGetUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	subject := r.PathValue("subject")
	if subject == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing user subject")
		return
	}

	user, err := a.service.GetUser(subject)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, userFromDomain(*user))
}

func (a *API) handleListUsers(
	w http.ResponseWriter,
	r *http.Request,
) {
	users, err := a.service.ListUsers()
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, usersFromDomain(users))
}

func (a *API) handleUpdateUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	subject := r.PathValue("subject")
	if subject == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing user subject")
		return
	}

	req, err := decodeRequest[UpdateUserRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	user, err := a.service.UpdateUser(subject, &service.UserUpdate{Handle: req.Handle, Roles: req.Roles})
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, userFromDomain(*user))
}

func (a *API) handleDeleteUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	subject := r.PathValue("subject")
	if subject == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing user subject")
		return
	}

	err := a.service.DeleteUser(subject)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

type Role struct {
	Name    string `json:"name"`
	Display string `json:"display"`
}

type UpdateRoleRequest struct {
	Display *string `json:"display,omitempty"`
}

func roleFromDomain(def service.Role) Role {
	return Role{
		Name:    def.Name,
		Display: def.Display,
	}
}

func rolesFromDomain(defs []service.Role) []Role {
	apiRoles := make([]Role, 0, len(defs))
	for _, def := range defs {
		apiRoles = append(apiRoles, roleFromDomain(def))
	}
	return apiRoles
}

func (a *API) handleCreateRole(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[Role](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	role, err := a.service.CreateRole(req.Name, req.Display)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, roleFromDomain(*role))
}

func (a *API) handleGetRole(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing role name")
		return
	}

	role, err := a.service.GetRole(name)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, roleFromDomain(*role))
}

func (a *API) handleUpdateRole(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing role name")
		return
	}

	req, err := decodeRequest[UpdateRoleRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	role, err := a.service.UpdateRole(name, req.Display)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, roleFromDomain(*role))
}

func (a *API) handleDeleteRole(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing role name")
		return
	}

	err := a.service.DeleteRole(name)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleListRoles(
	w http.ResponseWriter,
	r *http.Request,
) {
	roles, err := a.service.ListRoles()
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, rolesFromDomain(roles))
}
