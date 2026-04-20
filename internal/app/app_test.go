package app

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReturnStatusPage_RenderFailureFallsBackToInternalServerError(t *testing.T) {
	brokenStatus := template.Must(template.New("status.html").Parse(`{{template "missing.html" .}}`))
	appServer := &App{
		templates: &Templates{
			pages: map[string]*template.Template{
				"status.html": brokenStatus,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/authorize", nil)
	rr := httptest.NewRecorder()

	page := statusPageData{
		Title:   "Action Expired",
		Message: "This approval form is no longer valid.",
	}
	appServer.returnTemplate(rr, req, http.StatusForbidden, "status.html", page)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rr.Body.String(), http.StatusText(http.StatusInternalServerError)) {
		t.Fatalf("expected standard internal server error body")
	}
}
