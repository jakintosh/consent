package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

type Role struct {
	Name    string `json:"name"`
	Display string `json:"display"`
}

type UpdateRoleRequest struct {
	Display *string `json:"display,omitempty"`
}

func roleFromDomain(
	def service.Role,
) Role {
	return Role{
		Name:    def.Name,
		Display: def.Display,
	}
}

func rolesFromDomain(
	defs []service.Role,
) []Role {
	apiRoles := make([]Role, 0, len(defs))
	for _, def := range defs {
		apiRoles = append(apiRoles, roleFromDomain(def))
	}
	return apiRoles
}

func (a *API) buildRolesRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET    /", a.handleListRoles)
	mux.HandleFunc("POST   /", a.handleCreateRole)

	mux.HandleFunc("GET    /{name}", a.handleGetRole)
	mux.HandleFunc("PUT    /{name}", a.handleUpdateRole)
	mux.HandleFunc("DELETE /{name}", a.handleDeleteRole)

	return mux
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
