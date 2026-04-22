package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

type RegistrationRequest struct {
	Handle   string `json:"username"`
	Password string `json:"password"`
}

type ServiceDefinition struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

type UpdateServiceRequest struct {
	Display  *string `json:"display,omitempty"`
	Audience *string `json:"audience,omitempty"`
	Redirect *string `json:"redirect,omitempty"`
}

func serviceDefinitionFromDomain(def service.ServiceDefinition) ServiceDefinition {
	return ServiceDefinition{
		Name:     def.Name,
		Display:  def.Display,
		Audience: def.Audience,
		Redirect: def.Redirect,
	}
}

func serviceDefinitionsFromDomain(defs []service.ServiceDefinition) []ServiceDefinition {
	apiDefs := make([]ServiceDefinition, 0, len(defs))
	for _, def := range defs {
		apiDefs = append(apiDefs, serviceDefinitionFromDomain(def))
	}
	return apiDefs
}

func (a *API) handleRegister(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[RegistrationRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = a.service.Register(req.Handle, req.Password)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleCreateService(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[ServiceDefinition](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = a.service.CreateService(req.Name, req.Display, req.Audience, req.Redirect)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleGetService(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing service name")
		return
	}

	serviceDef, err := a.service.GetServiceByName(name)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, serviceDefinitionFromDomain(*serviceDef))
}

func (a *API) handleUpdateService(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing service name")
		return
	}

	req, err := decodeRequest[UpdateServiceRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = a.service.UpdateService(name, req.Display, req.Audience, req.Redirect)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleDeleteService(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing service name")
		return
	}

	err := a.service.DeleteService(name)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (a *API) handleListServices(
	w http.ResponseWriter,
	r *http.Request,
) {
	services, err := a.service.ListServices()
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, serviceDefinitionsFromDomain(services))
}
