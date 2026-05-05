package app

import (
	"embed"
	"net/http"
)

//go:embed assets/*
var assetsFS embed.FS

func (a *App) handleDefaultIntegrationLogo(
	w http.ResponseWriter,
	r *http.Request,
) {
	bytes, err := assetsFS.ReadFile("assets/default-integration-logo.png")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(bytes)
}
