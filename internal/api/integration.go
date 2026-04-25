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

func integrationFromDomain(
	integration service.Integration,
) Integration {
	return Integration{
		Name:     integration.Name,
		Display:  integration.Display,
		Audience: integration.Audience,
		Redirect: integration.Redirect,
	}
}

func integrationsFromDomain(
	integrations []service.Integration,
) []Integration {
	apiIntegrations := make([]Integration, 0, len(integrations))
	for _, integration := range integrations {
		apiIntegrations = append(apiIntegrations, integrationFromDomain(integration))
	}
	return apiIntegrations
}

func (a *API) buildIntegrationsRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET    /", a.handleListIntegrations)
	mux.HandleFunc("POST   /", a.handleCreateIntegration)

	mux.HandleFunc("GET    /{name}", a.handleGetIntegration)
	mux.HandleFunc("PATCH  /{name}", a.handleUpdateIntegration)
	mux.HandleFunc("DELETE /{name}", a.handleDeleteIntegration)

	return mux
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
