package app

import (
	"bytes"
	"html/template"
	"log"
	"path/filepath"
)

type Templates struct {
	templates *template.Template
}

func NewTemplates(dir string) *Templates {
	t, err := template.ParseGlob(filepath.Join(dir, "*"))
	if err != nil {
		log.Fatalf("Failed to load templates from '%s': %v", dir, err)
	}
	log.Printf("Loaded templates from %s", dir)
	return &Templates{templates: t}
}

func (t *Templates) RenderTemplate(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
