package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	InternalIntegrationName    = "consent"
	internalIntegrationDisplay = "Consent"
)

type Integration struct {
	Name     string
	Display  string
	Audience string
	Redirect string
}

type IntegrationUpdate struct {
	Display  *string
	Audience *string
	Redirect *string
}

func BuildInternalIntegration(
	publicUrl string,
) (
	Integration,
	error,
) {
	baseUrl, err := parseAndValidateRedirectURL(publicUrl)
	if err != nil {
		return Integration{}, err
	}

	redirectPath := strings.TrimRight(baseUrl.Path, "/") + "/auth/callback"
	redirectURL := &url.URL{
		Scheme: baseUrl.Scheme,
		Host:   baseUrl.Host,
		Path:   redirectPath,
	}

	return Integration{
		Name:     InternalIntegrationName,
		Display:  internalIntegrationDisplay,
		Audience: baseUrl.Host,
		Redirect: redirectURL.String(),
	}, nil
}

func SeedSystemIntegrations(
	store Store,
	publicURL string,
) error {
	if store == nil {
		return fmt.Errorf("service: store required")
	}

	internalIntegration, err := BuildInternalIntegration(publicURL)
	if err != nil {
		return fmt.Errorf("service: failed to build internal integration: %w", err)
	}

	if err := store.UpsertSystemIntegrations([]Integration{internalIntegration}); err != nil {
		return fmt.Errorf("service: failed to initialize system integrations: %w", err)
	}

	return nil
}

func (s *Service) CreateIntegration(
	name string,
	display string,
	audience string,
	redirect string,
) error {
	if name == "" {
		return ErrInvalidIntegration
	}
	if name == InternalIntegrationName {
		return ErrIntegrationProtected
	}

	if display == "" || audience == "" || redirect == "" {
		return ErrInvalidIntegration
	}

	if _, err := parseAndValidateRedirectURL(redirect); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRedirect, err)
	}

	err := s.store.InsertIntegration(name, display, audience, redirect)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrIntegrationExists
		}
		return fmt.Errorf("%w: failed to insert integration: %v", ErrInternal, err)
	}

	return nil
}

func (s *Service) GetIntegration(
	name string,
) (
	*Integration,
	error,
) {
	if name == "" {
		return nil, ErrInvalidIntegration
	}

	record, err := s.store.GetIntegration(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrIntegrationNotFound, name)
		}
		return nil, fmt.Errorf("%w: failed to get integration: %v", ErrInternal, err)
	}

	return &record, nil
}

func (s *Service) UpdateIntegration(
	name string,
	updates *IntegrationUpdate,
) error {
	if updates == nil {
		return nil
	}

	if name == "" {
		return ErrInvalidIntegration
	}
	if name == InternalIntegrationName {
		return ErrIntegrationProtected
	}

	current, err := s.store.GetIntegration(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %s", ErrIntegrationNotFound, name)
		}
		return fmt.Errorf("%w: failed to get integration: %v", ErrInternal, err)
	}

	if updates.Display != nil {
		current.Display = *updates.Display
	}
	if updates.Audience != nil {
		current.Audience = *updates.Audience
	}
	if updates.Redirect != nil {
		current.Redirect = *updates.Redirect
	}

	if current.Display == "" || current.Audience == "" || current.Redirect == "" {
		return ErrInvalidIntegration
	}

	if _, err := parseAndValidateRedirectURL(current.Redirect); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRedirect, err)
	}

	err = s.store.UpdateIntegration(name, updates)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %s", ErrIntegrationNotFound, name)
		}
		return fmt.Errorf("%w: failed to update integration: %v", ErrInternal, err)
	}

	return nil
}

func (s *Service) DeleteIntegration(
	name string,
) error {
	if name == "" {
		return ErrInvalidIntegration
	}
	if name == InternalIntegrationName {
		return ErrIntegrationProtected
	}

	deleted, err := s.store.DeleteIntegration(name)
	if err != nil {
		return fmt.Errorf("%w: failed to delete integration: %v", ErrInternal, err)
	}
	if !deleted {
		return fmt.Errorf("%w: %s", ErrIntegrationNotFound, name)
	}
	return nil
}

func (s *Service) ListIntegrations() (
	[]Integration,
	error,
) {
	records, err := s.store.ListIntegrations()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list integrations: %v", ErrInternal, err)
	}
	return records, nil
}
