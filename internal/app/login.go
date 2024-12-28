package app

import (
	"fmt"
	"net/http"

	"git.sr.ht/~jakintosh/consent/internal/resources"
)

func Login(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		logAppErr(r, fmt.Sprintf("missing required query param 'service'"))
		w.WriteHeader(http.StatusBadRequest)
		w.Write(badRequestHTML)
		return
	}

	service := resources.GetService(serviceName)
	if service == nil {
		logAppErr(r, fmt.Sprintf("requested service '%s' not registered", serviceName))
		w.WriteHeader(http.StatusBadRequest)
		w.Write(badRequestHTML)
		return
	}

	returnTemplate("login.html", service, w, r)
}
