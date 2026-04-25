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
