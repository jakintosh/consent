package app

import (
	"fmt"
	"net/http"
)

var loginPageSchema = []string{
	"audience",
	"redirect_url",
}

func Login(w http.ResponseWriter, r *http.Request) {
	if templates == nil {
		logAppErr(r, "templates are invalid")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(serverErrorHTML))
		return
	}

	model, err := validateModel(r, loginPageSchema)
	if err != nil {
		logAppErr(r, fmt.Sprintf("couldn't build page model: %v", err))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(badRequestHTML))
		return
	}

	err = templates.ExecuteTemplate(w, "login.html", model)
	if err != nil {
		logAppErr(r, fmt.Sprintf("couldn't render template: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(serverErrorHTML))
		return
	}
}
