package app

import (
	"bytes"
	"fmt"
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

func returnTemplate(name string, data any, w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, name, data)
	if err != nil {
		logAppErr(r, fmt.Sprintf("couldn't render template: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(serverErrorHTML))
		return
	}
	w.Write(buf.Bytes())
}

func validateQuery(r *http.Request, fields []string) (map[string]string, error) {
	query := r.URL.Query()
	model := make(map[string]string)
	for _, field := range fields {
		if _, ok := query[field]; !ok {
			return nil, fmt.Errorf("model missing field '%s'", field)
		}
		value := query.Get(field)
		if value == "" {
			return nil, fmt.Errorf("model has empty field '%s'", field)
		}
		model[field] = value
	}
	return model, nil
}

func logAppErr(r *http.Request, msg string) {
	log.Printf("%s %s: %s\n", r.Method, r.URL.String(), msg)
}

var badRequestHTML = `<!DOCTYPE html>
<html>
<head><style>:root{text-align:center;font-family:sans-serif;}</style></head>
<body>
<h1>Bad Request</h1>
<hr />
<p>You're using this page wrong.</p>
</body>
</html>
`

var serverErrorHTML = `<!DOCTYPE html>
<html>
<head>
<style>:root{text-align:center;font-family:sans-serif;}</style>
</head>
<body>
<h1>Server Error</h1>
<hr />
<p>The server ran into an issue; try again later.</p>
</body>
</html>
`
