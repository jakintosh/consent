package app

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"
)

//go:embed templates/*
var templatesFS embed.FS

type Templates struct {
	pages map[string]*template.Template
}

func NewTemplates() (
	*Templates,
	error,
) {
	return newTemplatesFromFS(templatesFS)
}

func newTemplatesFromFS(
	templateFS fs.FS,
) (
	*Templates,
	error,
) {
	pageFiles, err := fs.Glob(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	baseTemplate, err := template.ParseFS(templateFS, "templates/base.html")
	if err != nil {
		return nil, err
	}

	pages := map[string]*template.Template{}
	for _, pageFile := range pageFiles {
		pageName := path.Base(pageFile)
		if pageName == "base.html" {
			continue
		}

		pageTemplate, err := baseTemplate.Clone()
		if err != nil {
			return nil, err
		}

		if _, err := pageTemplate.ParseFS(templateFS, pageFile); err != nil {
			return nil, err
		}

		pages[pageName] = pageTemplate
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no renderable templates found")
	}

	return &Templates{pages: pages}, nil
}

func (t *Templates) RenderTemplate(
	name string,
	data any,
) (
	[]byte,
	error,
) {
	pageTemplate, ok := t.pages[name]
	if !ok {
		return nil, fmt.Errorf("unknown template: %s", name)
	}

	var buf bytes.Buffer
	if err := pageTemplate.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
