package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// HTTPResult captures HTTP response details for test assertions
type HTTPResult struct {
	Code    int
	Error   error
	Headers http.Header
	Body    []byte
}

// Header represents an HTTP header key-value pair
type Header struct {
	Key   string
	Value string
}

// ContentTypeJSON returns a header for JSON content type
func ContentTypeJSON() Header {
	return Header{
		Key:   "Content-Type",
		Value: "application/json",
	}
}

// ContentTypeForm returns a header for form-urlencoded content type
func ContentTypeForm() Header {
	return Header{
		Key:   "Content-Type",
		Value: "application/x-www-form-urlencoded",
	}
}

// ExpectStatus validates the HTTP status code and fails the test if it doesn't match
func ExpectStatus(
	t *testing.T,
	expected int,
	result HTTPResult,
) {
	t.Helper()
	if result.Error != nil {
		t.Fatalf("request error: %v", result.Error)
	}
	if result.Code != expected {
		t.Fatalf("expected status %d, got %d. Body: %s", expected, result.Code, string(result.Body))
	}
}

// ExpectRedirect validates a redirect response and returns the Location header
func ExpectRedirect(
	t *testing.T,
	result HTTPResult,
) string {
	t.Helper()
	if result.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect (303), got %d. Body: %s", result.Code, string(result.Body))
	}
	location := result.Headers.Get("Location")
	if location == "" {
		t.Fatal("expected Location header in redirect")
	}
	return location
}

// Get performs a GET request and optionally decodes JSON response
func Get(
	router http.Handler,
	url string,
	response any,
	headers ...Header,
) HTTPResult {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	res := httptest.NewRecorder()
	for _, h := range headers {
		req.Header.Set(h.Key, h.Value)
	}
	router.ServeHTTP(res, req)

	if response != nil && res.Body.Len() > 0 {
		if err := json.Unmarshal(res.Body.Bytes(), response); err != nil {
			return HTTPResult{
				Code:    res.Code,
				Error:   fmt.Errorf("failed to decode JSON: %v\n%s", err, res.Body.String()),
				Headers: res.Header(),
				Body:    res.Body.Bytes(),
			}
		}
	}

	return HTTPResult{Code: res.Code, Headers: res.Header(), Body: res.Body.Bytes()}
}

// Post performs a POST request and optionally decodes JSON response
func Post(
	router http.Handler,
	url string,
	body string,
	response any,
	headers ...Header,
) HTTPResult {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	res := httptest.NewRecorder()
	for _, h := range headers {
		req.Header.Set(h.Key, h.Value)
	}
	router.ServeHTTP(res, req)

	if response != nil && res.Body.Len() > 0 {
		if err := json.Unmarshal(res.Body.Bytes(), response); err != nil {
			return HTTPResult{
				Code:    res.Code,
				Error:   fmt.Errorf("failed to decode JSON: %v\n%s", err, res.Body.String()),
				Headers: res.Header(),
				Body:    res.Body.Bytes(),
			}
		}
	}

	return HTTPResult{Code: res.Code, Headers: res.Header(), Body: res.Body.Bytes()}
}

// PostForm performs a POST with form-urlencoded body
func PostForm(
	router http.Handler,
	urlPath string,
	values url.Values,
	response any,
) HTTPResult {
	return Post(router, urlPath, values.Encode(), response, ContentTypeForm())
}

// PostJSON performs a POST with JSON body
func PostJSON(
	router http.Handler,
	urlPath string,
	body string,
	response any,
) HTTPResult {
	return Post(router, urlPath, body, response, ContentTypeJSON())
}
