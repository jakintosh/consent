package app

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestNewTemplatesFromFS_LoadsRenderablePages(t *testing.T) {
	templates, err := newTemplatesFromFS(testTemplateFS())
	if err != nil {
		t.Fatalf("newTemplatesFromFS failed: %v", err)
	}

	if got, want := len(templates.pages), 4; got != want {
		t.Fatalf("loaded page count = %d, want %d", got, want)
	}

	homeBytes, err := templates.RenderTemplate("home.html", map[string]string{"Message": "hello"})
	if err != nil {
		t.Fatalf("RenderTemplate(home.html) failed: %v", err)
	}
	if !strings.Contains(string(homeBytes), "home: hello") {
		t.Fatalf("expected home template content")
	}
	if strings.Contains(string(homeBytes), "login:") {
		t.Fatalf("home template unexpectedly rendered login content")
	}

	loginBytes, err := templates.RenderTemplate("login.html", map[string]string{"Message": "hello"})
	if err != nil {
		t.Fatalf("RenderTemplate(login.html) failed: %v", err)
	}
	if !strings.Contains(string(loginBytes), "login: hello") {
		t.Fatalf("expected login template content")
	}
	if strings.Contains(string(loginBytes), "home:") {
		t.Fatalf("login template unexpectedly rendered home content")
	}

	authorizeBytes, err := templates.RenderTemplate("authorize.html", map[string]string{"Message": "hello"})
	if err != nil {
		t.Fatalf("RenderTemplate(authorize.html) failed: %v", err)
	}
	if !strings.Contains(string(authorizeBytes), "authorize: hello") {
		t.Fatalf("expected authorize template content")
	}

	statusBytes, err := templates.RenderTemplate("status.html", map[string]string{"Message": "hello"})
	if err != nil {
		t.Fatalf("RenderTemplate(status.html) failed: %v", err)
	}
	if !strings.Contains(string(statusBytes), "status: hello") {
		t.Fatalf("expected status template content")
	}
}

func TestTemplatesRenderTemplate_UnknownTemplate(t *testing.T) {
	templates, err := newTemplatesFromFS(testTemplateFS())
	if err != nil {
		t.Fatalf("newTemplatesFromFS failed: %v", err)
	}

	_, err = templates.RenderTemplate("missing.html", nil)
	if err == nil {
		t.Fatalf("expected unknown template error")
	}
	if !strings.Contains(err.Error(), "unknown template: missing.html") {
		t.Fatalf("error = %q, want unknown template error", err.Error())
	}
}

func TestNewTemplatesFromFS_NoRenderableTemplates(t *testing.T) {
	_, err := newTemplatesFromFS(fstest.MapFS{
		"templates/base.html": {
			Data: []byte(`{{define "base.html"}}<body>{{template "content" .}}</body>{{end}}`),
		},
	})
	if err == nil {
		t.Fatalf("expected no renderable templates error")
	}
	if !strings.Contains(err.Error(), "no renderable templates found") {
		t.Fatalf("error = %q, want no renderable templates found", err.Error())
	}
}

func testTemplateFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/base.html": {
			Data: []byte(`{{define "base.html"}}<body>{{template "content" .}}</body>{{end}}`),
		},
		"templates/home.html": {
			Data: []byte(`{{define "content"}}home: {{.Message}}{{end}}{{template "base.html" .}}`),
		},
		"templates/login.html": {
			Data: []byte(`{{define "content"}}login: {{.Message}}{{end}}{{template "base.html" .}}`),
		},
		"templates/authorize.html": {
			Data: []byte(`{{define "content"}}authorize: {{.Message}}{{end}}{{template "base.html" .}}`),
		},
		"templates/status.html": {
			Data: []byte(`{{define "content"}}status: {{.Message}}{{end}}{{template "base.html" .}}`),
		},
	}
}
