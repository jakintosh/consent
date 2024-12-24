package app

import (
	"log"
	"net/http"
	"net/url"
)

func Init(tmplDir string) {
	templateDir = tmplDir
	loadTemplates(templateDir)

	err := watchTemplates(tmplDir)
	if err != nil {
		log.Fatalf("Failed to start template watcher: %v", err)
	}
}

type LoginPageModel struct {
	Audience string
	Redirect string
}

func Login(w http.ResponseWriter, r *http.Request) {
	if templates == nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("app.Login(): templates are invalid\n")
		return
	}

	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Printf("app.Login(): parse query failure: %v\n", err)
	}

	model := LoginPageModel{
		Audience: params.Get("audience"),
		Redirect: params.Get("redirect_url"),
	}

	if model.Audience == "" || model.Redirect == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = templates.ExecuteTemplate(w, "login.html", model)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("app.Login(): template rendering issue: %v\n", err)
		return
	}
}
