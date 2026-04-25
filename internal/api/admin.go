package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
)

func (a *API) buildAdminRouter() http.Handler {
	mux := http.NewServeMux()

	wire.Subrouter(mux, "/keys", a.keys.Handler())
	wire.Subrouter(mux, "/integrations", a.buildIntegrationsRouter())
	wire.Subrouter(mux, "/roles", a.buildRolesRouter())
	wire.Subrouter(mux, "/users", a.buildUsersRouter())

	return mux
}
