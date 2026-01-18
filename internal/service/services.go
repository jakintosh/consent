package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
)

type CreateServiceRequest struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

type UpdateServiceRequest struct {
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

func (s *Service) CreateService(
	name string,
	display string,
	audience string,
	redirect string,
) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidService
	}

	if strings.TrimSpace(display) == "" || strings.TrimSpace(audience) == "" || strings.TrimSpace(redirect) == "" {
		return ErrInvalidService
	}

	if _, err := parseRedirectURL(redirect); err != nil {
		return err
	}

	err := s.store.InsertService(name, display, audience, redirect)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrServiceExists
		}
		return fmt.Errorf("%w: failed to insert service: %v", ErrInternal, err)
	}

	return nil
}

func (s *Service) GetServiceByName(
	name string,
) (
	*ServiceDefinition,
	error,
) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidService
	}

	record, err := s.store.GetService(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, name)
		}
		return nil, fmt.Errorf("%w: failed to get service: %v", ErrInternal, err)
	}

	redirectURL, err := parseRedirectURL(record.Redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInternal, err)
	}

	return &ServiceDefinition{
		Name:     record.Name,
		Display:  record.Display,
		Audience: record.Audience,
		Redirect: redirectURL,
	}, nil
}

func (s *Service) UpdateService(
	name string,
	display string,
	audience string,
	redirect string,
) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidService
	}

	if strings.TrimSpace(display) == "" || strings.TrimSpace(audience) == "" || strings.TrimSpace(redirect) == "" {
		return ErrInvalidService
	}

	if _, err := parseRedirectURL(redirect); err != nil {
		return err
	}

	err := s.store.UpdateService(name, display, audience, redirect)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %s", ErrServiceNotFound, name)
		}
		return fmt.Errorf("%w: failed to update service: %v", ErrInternal, err)
	}

	return nil
}

func (s *Service) DeleteService(
	name string,
) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidService
	}

	deleted, err := s.store.DeleteService(name)
	if err != nil {
		return fmt.Errorf("%w: failed to delete service: %v", ErrInternal, err)
	}
	if !deleted {
		return fmt.Errorf("%w: %s", ErrServiceNotFound, name)
	}
	return nil
}

func (s *Service) ListServices() ([]*ServiceDefinition, error) {
	records, err := s.store.ListServices()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list services: %v", ErrInternal, err)
	}

	services := make([]*ServiceDefinition, 0, len(records))
	for _, record := range records {
		redirectURL, err := parseRedirectURL(record.Redirect)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInternal, err)
		}
		services = append(services, &ServiceDefinition{
			Name:     record.Name,
			Display:  record.Display,
			Audience: record.Audience,
			Redirect: redirectURL,
		})
	}

	return services, nil
}

func parseRedirectURL(redirect string) (*url.URL, error) {
	parsed, err := url.Parse(redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRedirect, err)
	}
	if parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidRedirect
	}
	return parsed, nil
}

func (s *Service) handleCreateService(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[CreateServiceRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	err = s.CreateService(req.Name, req.Display, req.Audience, req.Redirect)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (s *Service) handleGetService(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing service name")
		return
	}

	serviceDef, err := s.GetServiceByName(name)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	response := CreateServiceRequest{
		Name:     serviceDef.Name,
		Display:  serviceDef.Display,
		Audience: serviceDef.Audience,
		Redirect: serviceDef.Redirect.String(),
	}
	wire.WriteData(w, http.StatusOK, response)
}

func (s *Service) handleUpdateService(
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

	err = s.UpdateService(name, req.Display, req.Audience, req.Redirect)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (s *Service) handleDeleteService(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.PathValue("name")
	if name == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing service name")
		return
	}

	err := s.DeleteService(name)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

func (s *Service) handleListServices(
	w http.ResponseWriter,
	r *http.Request,
) {
	services, err := s.ListServices()
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	response := make([]CreateServiceRequest, 0, len(services))
	for _, serviceDef := range services {
		response = append(response, CreateServiceRequest{
			Name:     serviceDef.Name,
			Display:  serviceDef.Display,
			Audience: serviceDef.Audience,
			Redirect: serviceDef.Redirect.String(),
		})
	}

	wire.WriteData(w, http.StatusOK, response)
}
