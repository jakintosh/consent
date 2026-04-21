package service

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
)

func (s *Service) BuildRouter() http.Handler {
	root := http.NewServeMux()

	wire.Subrouter(root, "/auth", s.buildAuthRouter())
	wire.Subrouter(root, "/admin", s.keys.WithAuth(s.buildAdminRouter(), &PermissionAdmin))

	return root
}

func (s *Service) buildAuthRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.HandleFunc("POST /refresh", s.handleRefresh)
	mux.HandleFunc("GET /me", s.handleMe)

	return mux
}

func (s *Service) buildServicesRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.handleListServices)
	mux.HandleFunc("POST /", s.handleCreateService)
	mux.HandleFunc("GET /{name}", s.handleGetService)
	mux.HandleFunc("PUT /{name}", s.handleUpdateService)
	mux.HandleFunc("DELETE /{name}", s.handleDeleteService)

	return mux
}

func (s *Service) buildAdminRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", s.handleRegister)
	wire.Subrouter(mux, "/services", s.buildServicesRouter())
	wire.Subrouter(mux, "/keys", s.keys.Handler())

	return mux
}
