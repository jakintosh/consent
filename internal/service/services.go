package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type ServiceDefinition struct {
	Name     string
	Display  string
	Audience string
	Redirect string
}

const (
	InternalServiceName    = "consent"
	internalServiceDisplay = "Consent"
)

func BuildInternalServiceDefinition(
	publicUrl string,
) (
	ServiceDefinition,
	error,
) {
	baseUrl, err := parseAndValidateRedirectURL(publicUrl)
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

func SeedSystemServices(
	store Store,
	publicURL string,
) error {
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

	if _, err := parseAndValidateRedirectURL(redirect); err != nil {
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
	display *string,
	audience *string,
	redirect *string,
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
	if display != nil {
		current.Display = strings.TrimSpace(*display)
	}
	if audience != nil {
		current.Audience = strings.TrimSpace(*audience)
	}
	if redirect != nil {
		current.Redirect = strings.TrimSpace(*redirect)
	}

	// Validate final values
	if current.Display == "" || current.Audience == "" || current.Redirect == "" {
		return ErrInvalidService
	}

	if _, err := parseAndValidateRedirectURL(current.Redirect); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRedirect, err)
	}

	err = s.store.UpdateService(name, current.Display, current.Audience, current.Redirect)
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
