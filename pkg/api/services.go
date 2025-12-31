package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

type Service struct {
	Display  string   `json:"display"`
	Audience string   `json:"audience"`
	Redirect *url.URL `json:"redirect"`
}

func (s *Service) UnmarshalJSON(data []byte) error {
	type Alias Service
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

type Services struct {
	services map[string]*Service
}

func NewServices(dir string) *Services {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Failed to read services directory '%s': %v", dir, err)
	}

	svcs := make(map[string]*Service)
	for _, file := range files {
		if !file.Type().IsRegular() {
			continue
		}
		name := file.Name()
		service, err := loadService(filepath.Join(dir, name))
		if err != nil {
			log.Fatalf("Failed to load service '%s': %v", name, err)
		}
		svcs[name] = service
	}

	log.Printf("Loaded %d services from %s", len(svcs), dir)
	return &Services{services: svcs}
}

func (s *Services) GetService(name string) (*Service, error) {
	if service, ok := s.services[name]; ok {
		return service, nil
	}
	return nil, fmt.Errorf("service not found: %s", name)
}

func loadService(serviceDefPath string) (*Service, error) {
	file, err := os.ReadFile(serviceDefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load service definition: %w", err)
	}

	service := &Service{}
	err = json.Unmarshal(file, service)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json of '%s': %w", serviceDefPath, err)
	}
	return service, nil
}
