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
) (
	*App,
	error,
) {
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
	mux.HandleFunc("/", a.serve(a.handleGetHome))
	mux.HandleFunc("GET /login", a.serve(a.handleGetLogin))
	mux.HandleFunc("POST /login", a.serve(a.handlePostLogin))
	mux.HandleFunc("GET /authorize", a.serve(a.handleGetAuthorize))
	mux.HandleFunc("POST /authorize", a.serve(a.handlePostAuthorize))
	for pattern, handler := range a.auth.Routes {
		mux.HandleFunc(pattern, handler)
	}
	return mux
}

type appHandler func(http.ResponseWriter, *http.Request) *appError

type statusPageData struct {
	Title   string
	Message string
}

func (a *App) serve(handler appHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			spec := appErrorSpecs[err.kind]
			if spec.loggable {
				if err.cause != nil {
					logAppErr(r, fmt.Sprintf("%s: %v", spec.logMessage, err.cause))
				} else {
					logAppErr(r, spec.logMessage)
				}
			}
			page := statusPageData{
				Title:   spec.title,
				Message: spec.message,
			}
			a.returnTemplate(w, r, spec.status, "status.html", page)
		}
	}
}

func (a *App) returnTemplate(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	name string,
	data any,
) {
	bytes, err := a.templates.RenderTemplate(name, data)
	if err != nil {
		logAppErr(r, fmt.Sprintf("couldn't render template: %v", err))
		error := http.StatusText(http.StatusInternalServerError)
		code := http.StatusInternalServerError
		http.Error(w, error, code)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	_, _ = w.Write(bytes)
}

func logAppErr(r *http.Request, msg string) {
	log.Printf("%s %s: %s\n", r.Method, r.URL.String(), msg)
}
