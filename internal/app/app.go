package app

import (
	"log"
	"net/http"
)

func Init(tmplDir string) {
	templateDir = tmplDir
	loadTemplates(templateDir)

	err := watchTemplates(tmplDir)
	if err != nil {
		log.Fatalf("Failed to start template watcher: %v", err)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	if templates == nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("app.Login(): templates are invalid\n")
		return
	}

	err := templates.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("app.Login(): template rendering issue: %v\n", err)
		return
	}
}
