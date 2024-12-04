package routing

import (
	"github.com/gorilla/mux"
)

func BuildRouter() *mux.Router {
	r := mux.NewRouter()

	buildAPIRouter(r)

	return r
}
