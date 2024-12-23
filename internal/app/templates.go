package app

import (
	"html/template"
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

var templateDir string
var templates *template.Template

func loadTemplates(directory string) {
	var err error
	templates, err = template.ParseGlob(filepath.Join(templateDir, "*"))
	if err != nil {
		templates = nil
		log.Printf("Failed to parse templates from '%s': %v", directory, err)
	}

	log.Printf("Loaded templates from %v\n", directory)
}

func watchTemplates(directory string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.Add(directory)
	if err != nil {
		return err
	}

	reload := make(chan struct{})
	go scheduleTemplateReload(reload)
	go handleWatcher(watcher, reload)

	return nil
}

func handleWatcher(watcher *fsnotify.Watcher, reloadC chan<- struct{}) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write | fsnotify.Remove | fsnotify.Create) {
				reloadC <- struct{}{}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("template watcher error: %v\n", err)
		}
	}

}

func scheduleTemplateReload(reload <-chan struct{}) {
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
			loadTemplates(templateDir)
		}
	}
}
