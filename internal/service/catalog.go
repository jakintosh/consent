package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type ServiceDefinition struct {
	Display  string   `json:"display"`
	Audience string   `json:"audience"`
	Redirect *url.URL `json:"redirect"`
}

func (s *ServiceDefinition) UnmarshalJSON(
	data []byte,
) error {
	type Alias ServiceDefinition
	tmp := &struct {
		Redirect string `json:"redirect"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	redirect, err := url.Parse(tmp.Redirect)
	if err != nil {
		return err
	}
	s.Redirect = redirect
	return nil
}

type ServiceCatalog struct {
	services map[string]*ServiceDefinition
}

func NewServiceCatalog(
	dir string,
) (
	*ServiceCatalog,
	error,
) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read services directory '%s': %w", dir, err)
	}

	svcs := make(map[string]*ServiceDefinition)
	for _, file := range files {
		if !file.Type().IsRegular() {
			continue
		}
		name := file.Name()
		service, err := loadServiceDefinition(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("failed to load service '%s': %w", name, err)
		}
		svcs[name] = service
	}

	return &ServiceCatalog{services: svcs}, nil
}

func (c *ServiceCatalog) GetService(
	name string,
) (
	*ServiceDefinition,
	error,
) {
	if service, ok := c.services[name]; ok {
		return service, nil
	}
	return nil, fmt.Errorf("service not found: %s", name)
}

func loadServiceDefinition(
	serviceDefPath string,
) (
	*ServiceDefinition,
	error,
) {
	file, err := os.ReadFile(serviceDefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load service definition: %w", err)
	}

	service := &ServiceDefinition{}
	err = json.Unmarshal(file, service)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json of '%s': %w", serviceDefPath, err)
	}
	return service, nil
}
