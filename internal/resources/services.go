package resources

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type ServiceDefinition struct {
	Display  string `json:"display"`
	Audience string `json:"audience"`
	Redirect string `json:"redirect"`
}

var servicesDir string = ""
var services = make(map[string]*ServiceDefinition)

func GetService(name string) *ServiceDefinition {
	if service, ok := services[name]; ok {
		return service
	} else {
		return nil
	}
}

func initServices(servicesDirPath string) {
	servicesDir = servicesDirPath
	loadServices(servicesDir)

	err := watchDir(servicesDir, func() {
		loadServices(servicesDir)
	})
	if err != nil {
		log.Fatalf("Failed to start service watcher: %v", err)
	}
}

func loadServices(servicesDirPath string) {
	files, err := os.ReadDir(servicesDirPath)
	if err != nil {
		log.Printf("services: failed to read service defs dir: %v\n", err)
		return
	}

	clear(services)
	for _, file := range files {
		if !file.Type().IsRegular() {
			continue
		}
		name := file.Name()
		service, err := loadService(filepath.Join(servicesDirPath, name))
		if err != nil {
			log.Printf("services: failed to read service def for '%s': %v\n", name, err)
		}

		if _, ok := services[name]; ok {
			log.Printf("services: duplicate definition for '%s'; overwriting\n", name)
		}
		services[name] = service
	}

	log.Printf("Loaded services from %s\n", servicesDirPath)
}

func loadService(serviceDefPath string) (*ServiceDefinition, error) {
	file, err := os.ReadFile(serviceDefPath)
	if err != nil {
		log.Fatalf("failed to load services definitions: %v\n", err)
	}

	service := &ServiceDefinition{}
	err = json.Unmarshal(file, service)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json of '%s': %v", serviceDefPath, err)
	}
	return service, nil
}
