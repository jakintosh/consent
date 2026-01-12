package app

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*
var templatesFS embed.FS

type Templates struct {
	templates *template.Template
}

func NewTemplates() (*Templates, error) {
	t, err := template.ParseFS(templatesFS, "templates/*")
	if err != nil {
		return nil, err
	}
	return &Templates{templates: t}, nil
}

func (t *Templates) RenderTemplate(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
