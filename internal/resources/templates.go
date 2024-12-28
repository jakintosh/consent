package resources

import (
	"bytes"
	"html/template"
	"log"
	"path/filepath"
)

var templateDir string
var templates *template.Template

func RenderTemplate(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, name, data)
	return buf.Bytes(), err
}

func initTemplates(directory string) {
	templateDir = directory
	loadTemplates(templateDir)

	err := watchDir(templateDir, func() {
		loadTemplates(templateDir)
	})
	if err != nil {
		log.Fatalf("Failed to start template watcher: %v", err)
	}

}

func loadTemplates(directory string) {
	var err error
	templates, err = template.ParseGlob(filepath.Join(templateDir, "*"))
	if err != nil {
		templates = nil
		log.Printf("Failed to parse templates from '%s': %v", directory, err)
	}

	log.Printf("Loaded templates from %v\n", directory)
}
