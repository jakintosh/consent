package api

import (
	"fmt"
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

type Options struct {
	Service   *service.Service
	KeysStore keys.Store
}

type API struct {
	service *service.Service
	keys    *keys.Service
}

func New(
	options Options,
) (
	*API,
	error,
) {
	if options.Service == nil {
		return nil, fmt.Errorf("api: service required")
	}
	if options.KeysStore == nil {
		return nil, fmt.Errorf("api: keys store required")
	}

	keysOpts := keys.Options{
		Store:       options.KeysStore,
		Permissions: service.AllKeyPermissions(),
	}
	keysSvc, err := keys.New(keysOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize key service: %w", err)
	}

	return &API{
		service: options.Service,
		keys:    keysSvc,
	}, nil
}

func (a *API) Router() http.Handler {
	root := http.NewServeMux()

	wire.Subrouter(root, "/auth", a.buildAuthRouter())
	wire.Subrouter(root, "/admin", a.keys.WithAuth(a.buildAdminRouter(), &service.PermissionAdmin))

	return root
}

func (a *API) buildAuthRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /me", a.handleMe)
	mux.HandleFunc("POST /login", a.handleLogin)
	mux.HandleFunc("POST /logout", a.handleLogout)
	mux.HandleFunc("POST /refresh", a.handleRefresh)

	return mux
}

func (a *API) buildServicesRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", a.handleListServices)
	mux.HandleFunc("POST /", a.handleCreateService)
	mux.HandleFunc("GET /{name}", a.handleGetService)
	mux.HandleFunc("PUT /{name}", a.handleUpdateService)
	mux.HandleFunc("DELETE /{name}", a.handleDeleteService)

	return mux
}

func (a *API) buildAdminRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", a.handleRegister)
	wire.Subrouter(mux, "/services", a.buildServicesRouter())
	wire.Subrouter(mux, "/keys", a.keys.Handler())

	return mux
}
