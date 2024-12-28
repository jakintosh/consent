package resources

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
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

	err := watchServices(servicesDir)
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

	log.Printf("Loaded services from %s: %v\n", servicesDirPath, services)
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

func watchServices(directory string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.Add(directory)
	if err != nil {
		return err
	}

	reload := make(chan struct{})
	go scheduleServicesReload(reload)
	go handleWatcher(watcher, reload)

	return nil
}

func handleWatcher(watcher *fsnotify.Watcher, reload chan<- struct{}) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write | fsnotify.Remove | fsnotify.Create) {
				reload <- struct{}{}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("service watcher error: %v\n", err)
		}
	}

}

func scheduleServicesReload(reload <-chan struct{}) {
	var timer *time.Timer = nil
	var c <-chan time.Time = nil
	duration := time.Millisecond * 500
	for {
		select {
		case <-reload:
			if timer != nil {
				timer.Reset(duration)
			} else {
				timer = time.NewTimer(duration)
				c = timer.C
			}

		case <-c:
			c = nil
			timer = nil
			loadServices(servicesDir)
		}
	}
}
