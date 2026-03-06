// Package app provides web UI handlers with HTML templates for the consent server.
package app

import (
	"fmt"
	"log"
	"net/http"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/pkg/client"
)

// AuthConfig configures authentication behavior for the app UI.
type AuthConfig struct {
	Verifier  client.Verifier
	LoginURL  string
	LogoutURL string
	Routes    map[string]http.HandlerFunc
}

type AppOptions struct {
	Service *service.Service
	Auth    AuthConfig
}

type App struct {
	service   *service.Service
	auth      AuthConfig
	templates *Templates
}

func New(
	options AppOptions,
) (*App, error) {
	if options.Service == nil {
		return nil, fmt.Errorf("service is required")
	}
	if options.Auth.Verifier == nil {
		return nil, fmt.Errorf("auth verifier is required")
	}
	if options.Auth.LoginURL == "" {
		return nil, fmt.Errorf("auth login URL is required")
	}
	if options.Auth.LogoutURL == "" {
		return nil, fmt.Errorf("auth logout URL is required")
	}
	if options.Auth.Routes == nil {
		return nil, fmt.Errorf("auth routes are required")
	}

	routes := make(map[string]http.HandlerFunc, len(options.Auth.Routes))
	for pattern, handler := range options.Auth.Routes {
		if pattern == "" {
			return nil, fmt.Errorf("auth route pattern cannot be empty")
		}
		if handler == nil {
			return nil, fmt.Errorf("auth route handler cannot be nil: %s", pattern)
		}
		routes[pattern] = handler
	}

	templates, err := NewTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &App{
		service: options.Service,
		auth: AuthConfig{
			Verifier:  options.Auth.Verifier,
			LoginURL:  options.Auth.LoginURL,
			LogoutURL: options.Auth.LogoutURL,
			Routes:    routes,
		},
		templates: templates,
	}, nil
}

func (a *App) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.Home())
	mux.HandleFunc("/login", a.Login())
	for pattern, handler := range a.auth.Routes {
		mux.HandleFunc(pattern, handler)
	}
	return mux
}

func (a *App) returnTemplate(
	name string,
	data any,
	w http.ResponseWriter,
	r *http.Request,
) {
	bytes, err := a.templates.RenderTemplate(name, data)
	if err != nil {
		logAppErr(r, fmt.Sprintf("couldn't render template: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(serverErrorHTML)
		return
	}
	w.Write(bytes)
}

func logAppErr(r *http.Request, msg string) {
	log.Printf("%s %s: %s\n", r.Method, r.URL.String(), msg)
}

var badRequestHTML = []byte(`<!DOCTYPE html><html>
<head><style>:root{text-align:center;font-family:sans-serif;}</style></head>
<body><h1>Bad Request</h1><hr /><p>You're using this page wrong.</p></body>
</html>`)

var serverErrorHTML = []byte(`<!DOCTYPE html><html>
<head><style>:root{text-align:center;font-family:sans-serif;}</style></head>
<body><h1>Server Error</h1><hr /><p>The server ran into an issue; try again later.</p></body>
</html>`)
