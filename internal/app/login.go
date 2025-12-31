package app

import (
	"fmt"
	"net/http"
)

func Login(
	w http.ResponseWriter,
	r *http.Request,
) {

	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		logAppErr(r, "missing required query param 'service'")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(badRequestHTML)
		return
	}

	service, err := services.GetService(serviceName)
	if err != nil {
		logAppErr(r, fmt.Sprintf("invalid service: %s", serviceName))
		w.WriteHeader(http.StatusBadRequest)
		w.Write(badRequestHTML)
		return
	}

	data := map[string]string{
		"Display": service.Display,
		"Name":    serviceName,
	}
	if service == nil {
		logAppErr(r, fmt.Sprintf("requested service '%s' not registered", serviceName))
		w.WriteHeader(http.StatusBadRequest)
		w.Write(badRequestHTML)
		return
	}

	returnTemplate("login.html", data, w, r)
}
