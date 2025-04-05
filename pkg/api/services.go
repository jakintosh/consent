package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"git.sr.ht/~jakintosh/consent/internal/resources"
)

type Services interface {
	GetService(name string) (*Service, error)
}

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

//
// Dynamic file-based services provider

type DynamicServicesDirectory struct {
	servicesDir string
	services    map[string]*Service
}

func NewDynamicServicesDirectory(dir string) *DynamicServicesDirectory {

	s := &DynamicServicesDirectory{
		servicesDir: dir,
		services:    make(map[string]*Service),
	}

	s.Load()

	// TODO: This needs to not use the internal 'resources' package
	err := resources.WatchDir(s.servicesDir, func() { s.Load() })
	if err != nil {
		// TODO: maybe better error handling
		log.Fatalf("Failed to start service watcher: %v", err)
	}

	return s
}

func (s *DynamicServicesDirectory) Load() {

	files, err := os.ReadDir(s.servicesDir)
	if err != nil {
		// TODO: maybe better error handling
		log.Printf("services: failed to read service defs dir: %v\n", err)
		return
	}

	clear(s.services)
	for _, file := range files {
		if !file.Type().IsRegular() {
			continue
		}
		name := file.Name()
		service, err := loadService(filepath.Join(s.servicesDir, name))
		if err != nil {
			// TODO: maybe better error handling
			log.Printf("services: failed to read service def for '%s': %v\n", name, err)
			return
		}

		if _, ok := s.services[name]; ok {
			// TODO: maybe better error handling
			log.Printf("services: duplicate definition for '%s'; overwriting\n", name)
			return
		}
		s.services[name] = service
	}

	log.Printf("Loaded services from %s\n", s.servicesDir)
}

func loadService(
	serviceDefPath string,
) (
	*Service,
	error,
) {

	file, err := os.ReadFile(serviceDefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load services definitions: %w\n", err)
	}

	service := &Service{}
	err = json.Unmarshal(file, service)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json of '%s': %w", serviceDefPath, err)
	}
	return service, nil
}

//
// Services interface

func (s *DynamicServicesDirectory) GetService(
	name string,
) (*Service, error) {

	if service, ok := s.services[name]; ok {
		return service, nil
	} else {
		return nil, fmt.Errorf("service not found")
	}
}
