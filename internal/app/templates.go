package app

import (
	"bytes"
	"html/template"
	"log"
	"path/filepath"

	"git.sr.ht/~jakintosh/consent/internal/resources"
)

type Templates interface {
	RenderTemplate(name string, data any) ([]byte, error)
}

type DynamicTemplateDirectory struct {
	Directory string
	Templates *template.Template
}

func NewDynamicTemplatesDirectory(dir string) Templates {

	t := &DynamicTemplateDirectory{
		Directory: dir,
		Templates: &template.Template{},
	}

	t.Load()

	err := resources.WatchDir(t.Directory, func() { t.Load() })
	if err != nil {
		// TODO: maybe better error handling
		log.Fatalf("Failed to start template watcher: %v", err)
	}

	return t
}

func (t *DynamicTemplateDirectory) RenderTemplate(
	name string,
	data any,
) (
	[]byte,
	error,
) {

	var buf bytes.Buffer
	if err := t.Templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *DynamicTemplateDirectory) Load() {

	var err error
	t.Templates, err = template.ParseGlob(filepath.Join(t.Directory, "*"))
	if err != nil {
		// TODO: maybe better error handling
		templates = nil
		log.Printf("Failed to parse templates from '%s': %v", t.Directory, err)
		return
	}

	log.Printf("Loaded templates from %v\n", t.Directory)
}
