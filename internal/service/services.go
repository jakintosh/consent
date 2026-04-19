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

type ServiceDefinition struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

const (
	InternalServiceName    = "consent"
	internalServiceDisplay = "Consent"
)

func BuildInternalServiceDefinition(publicUrl string) (
	ServiceDefinition,
	error,
) {
	baseUrl, err := parseFullURL(publicUrl)
	if err != nil {
		return ServiceDefinition{}, err
	}

	redirectPath := strings.TrimRight(baseUrl.Path, "/") + "/auth/callback"
	redirectURL := &url.URL{
		Scheme: baseUrl.Scheme,
		Host:   baseUrl.Host,
		Path:   redirectPath,
	}

	return ServiceDefinition{
		Name:     InternalServiceName,
		Display:  internalServiceDisplay,
		Audience: baseUrl.Host,
		Redirect: redirectURL.String(),
	}, nil
}

func EnsureSystemServices(store Store, publicURL string) error {
	if store == nil {
		return fmt.Errorf("service: store required")
	}

	internalService, err := BuildInternalServiceDefinition(publicURL)
	if err != nil {
		return fmt.Errorf("service: failed to build internal service: %w", err)
	}

	if err := store.UpsertSystemServices([]ServiceDefinition{internalService}); err != nil {
		return fmt.Errorf("service: failed to initialize system services: %w", err)
	}

	return nil
}

type UpdateServiceRequest struct {
	Display  *string `json:"display,omitempty"`
	Audience *string `json:"audience,omitempty"`
	Redirect *string `json:"redirect,omitempty"`
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
	if name == InternalServiceName {
		return ErrServiceProtected
	}

	if strings.TrimSpace(display) == "" || strings.TrimSpace(audience) == "" || strings.TrimSpace(redirect) == "" {
		return ErrInvalidService
	}

	if _, err := parseFullURL(redirect); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRedirect, err)
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

	return &record, nil
}

func (s *Service) UpdateService(
	name string,
	req UpdateServiceRequest,
) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidService
	}
	if name == InternalServiceName {
		return ErrServiceProtected
	}

	// Fetch current record to merge with partial updates
	current, err := s.store.GetService(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %s", ErrServiceNotFound, name)
		}
		return fmt.Errorf("%w: failed to get service: %v", ErrInternal, err)
	}

	// Apply updates (use current values as defaults)
	display := current.Display
	if req.Display != nil {
		display = strings.TrimSpace(*req.Display)
	}
	audience := current.Audience
	if req.Audience != nil {
		audience = strings.TrimSpace(*req.Audience)
	}
	redirect := current.Redirect
	if req.Redirect != nil {
		redirect = strings.TrimSpace(*req.Redirect)
	}

	// Validate final values
	if display == "" || audience == "" || redirect == "" {
		return ErrInvalidService
	}

	if _, err := parseFullURL(redirect); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRedirect, err)
	}

	err = s.store.UpdateService(name, display, audience, redirect)
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
	if name == InternalServiceName {
		return ErrServiceProtected
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

func (s *Service) ListServices() (
	[]ServiceDefinition,
	error,
) {
	records, err := s.store.ListServices()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list services: %v", ErrInternal, err)
	}
	return records, nil
}

func (s *Service) handleCreateService(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[ServiceDefinition](r)
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

	wire.WriteData(w, http.StatusOK, serviceDef)
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

	err = s.UpdateService(name, req)
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

	wire.WriteData(w, http.StatusOK, services)
}

func parseFullURL(redirect string) (
	*url.URL,
	error,
) {
	parsed, err := url.Parse(redirect)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRedirect, err)
	}
	if parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidUrl
	}
	return parsed, nil
}
