package app

import (
	"fmt"
	"net/http"
)

func (a *App) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceName := r.URL.Query().Get("service")
		if serviceName == "" {
			logAppErr(r, "missing required query param 'service'")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(badRequestHTML)
			return
		}

		svcDef, err := a.service.GetServiceByName(serviceName)
		if err != nil {
			logAppErr(r, fmt.Sprintf("invalid service: %s", serviceName))
			w.WriteHeader(http.StatusBadRequest)
			w.Write(badRequestHTML)
			return
		}

		data := map[string]string{
			"Display": svcDef.Display,
			"Name":    serviceName,
		}

		a.returnTemplate("login.html", data, w, r)
	}
}
