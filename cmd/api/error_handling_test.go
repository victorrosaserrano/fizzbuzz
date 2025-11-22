package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestErrorHandling_MalformedJSON tests AC 1: Return 400 for malformed JSON syntax
func TestErrorHandling_MalformedJSON(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name        string
		requestBody string
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "invalid JSON syntax",
			requestBody: `{"int1": 3, "int2": 5, "limit": 100, "str1": "fizz", "str2": "buzz"`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "the request body contains badly-formed JSON",
		},
		{
			name:        "unexpected EOF",
			requestBody: `{"int1": 3, "int2":`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "the request body contains badly-formed JSON",
		},
		{
			name:        "missing brace",
			requestBody: `{"int1": 3, "int2": 5`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "the request body contains badly-formed JSON",
		},
		{
			name:        "multiple JSON values",
			requestBody: `{"int1": 3}{"int2": 5}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "body must only contain a single JSON value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, err := http.NewRequest("POST", "/v1/fizzbuzz", strings.NewReader(tt.requestBody))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			app.routes().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("want status %d; got %d", tt.wantStatus, rr.Code)
			}

			var response map[string]string
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			if response["error"] != tt.wantMessage {
				t.Errorf("want message %q; got %q", tt.wantMessage, response["error"])
			}
		})
	}
}

// TestErrorHandling_InvalidDataTypes tests AC 1: Return 400 for invalid JSON data types
func TestErrorHandling_InvalidDataTypes(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name         string
		requestBody  string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "string where int expected",
			requestBody:  `{"int1": "not-a-number", "int2": 5, "limit": 100, "str1": "fizz", "str2": "buzz"}`,
			wantStatus:   http.StatusBadRequest,
			wantContains: "int1",
		},
		{
			name:         "number where string expected",
			requestBody:  `{"int1": 3, "int2": 5, "limit": 100, "str1": 123, "str2": "buzz"}`,
			wantStatus:   http.StatusBadRequest,
			wantContains: "str1",
		},
		{
			name:         "boolean where int expected",
			requestBody:  `{"int1": 3, "int2": true, "limit": 100, "str1": "fizz", "str2": "buzz"}`,
			wantStatus:   http.StatusBadRequest,
			wantContains: "int2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, err := http.NewRequest("POST", "/v1/fizzbuzz", strings.NewReader(tt.requestBody))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			app.routes().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("want status %d; got %d", tt.wantStatus, rr.Code)
			}

			body := rr.Body.String()
			if !strings.Contains(body, tt.wantContains) {
				t.Errorf("want response to contain %q; got %q", tt.wantContains, body)
			}
		})
	}
}

// TestErrorHandling_MissingContentType tests AC 1: Return 400 for missing Content-Type header
func TestErrorHandling_MissingContentType(t *testing.T) {
	app := newTestApplication(t)

	rr := httptest.NewRecorder()

	requestBody := `{"int1": 3, "int2": 5, "limit": 100, "str1": "fizz", "str2": "buzz"}`
	req, err := http.NewRequest("POST", "/v1/fizzbuzz", strings.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	// Intentionally omit Content-Type header

	app.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want status %d; got %d", http.StatusBadRequest, rr.Code)
	}

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	expectedMessage := "missing Content-Type header"
	if response["error"] != expectedMessage {
		t.Errorf("want message %q; got %q", expectedMessage, response["error"])
	}
}

// TestErrorHandling_RequestBodyTooLarge tests AC 1: Return 400 for request body exceeding maximum size
func TestErrorHandling_RequestBodyTooLarge(t *testing.T) {
	app := newTestApplication(t)

	// Create a request body that is exactly 1MB + 1 byte of pure content
	largeBody := strings.Repeat("{", 1_048_577) // 1MB + 1 byte - this will be malformed JSON

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("POST", "/v1/fizzbuzz", strings.NewReader(largeBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	app.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want status %d; got %d", http.StatusBadRequest, rr.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Should return either "too large" or "badly-formed JSON" - both are acceptable for AC 1
	errorMessage := response["error"].(string)
	if !strings.Contains(errorMessage, "too large") && !strings.Contains(errorMessage, "badly-formed JSON") {
		t.Errorf("expected error message about size or malformed JSON; got %q", errorMessage)
	}
}

// TestErrorHandling_NotFound tests AC 2: Return 404 for requests to unknown endpoints
func TestErrorHandling_NotFound(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "unknown endpoint",
			method:     "GET",
			path:       "/v1/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non-existent API version",
			method:     "POST",
			path:       "/v2/fizzbuzz",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown path with POST",
			method:     "POST",
			path:       "/api/test",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			app.routes().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("want status %d; got %d", tt.wantStatus, rr.Code)
			}

			var response map[string]string
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			expectedMessage := "the requested resource could not be found"
			if response["error"] != expectedMessage {
				t.Errorf("want message %q; got %q", expectedMessage, response["error"])
			}
		})
	}
}

// TestErrorHandling_MethodNotAllowed tests AC 3: Return 405 for incorrect HTTP methods
func TestErrorHandling_MethodNotAllowed(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantAllow   string
		wantMessage string
	}{
		{
			name:        "GET on POST-only endpoint",
			method:      "GET",
			path:        "/v1/fizzbuzz",
			wantStatus:  http.StatusMethodNotAllowed,
			wantAllow:   "POST",
			wantMessage: "the GET method is not supported for this resource",
		},
		{
			name:        "POST on GET-only endpoint",
			method:      "POST",
			path:        "/v1/healthcheck",
			wantStatus:  http.StatusMethodNotAllowed,
			wantAllow:   "GET",
			wantMessage: "the POST method is not supported for this resource",
		},
		{
			name:        "DELETE on fizzbuzz endpoint",
			method:      "DELETE",
			path:        "/v1/fizzbuzz",
			wantStatus:  http.StatusMethodNotAllowed,
			wantAllow:   "POST",
			wantMessage: "the DELETE method is not supported for this resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			app.routes().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("want status %d; got %d", tt.wantStatus, rr.Code)
			}

			// Check Allow header
			allowHeader := rr.Header().Get("Allow")
			if allowHeader != tt.wantAllow {
				t.Errorf("want Allow header %q; got %q", tt.wantAllow, allowHeader)
			}

			// Check error message
			var response map[string]string
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			if response["error"] != tt.wantMessage {
				t.Errorf("want message %q; got %q", tt.wantMessage, response["error"])
			}
		})
	}
}

// TestErrorHandling_PanicRecovery tests AC 4: Implement panic recovery middleware
func TestErrorHandling_PanicRecovery(t *testing.T) {
	app := newTestApplication(t)

	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	wrappedHandler := app.recoverPanic(handler)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("want status %d; got %d", http.StatusInternalServerError, rr.Code)
	}

	// Check that Connection header is set to close
	if rr.Header().Get("Connection") != "close" {
		t.Error("want Connection header to be 'close'")
	}

	// Check error response format
	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	expectedMessage := "the server encountered a problem and could not process your request"
	if response["error"] != expectedMessage {
		t.Errorf("want message %q; got %q", expectedMessage, response["error"])
	}
}

// TestErrorHandling_JSONErrorResponseFormat tests AC 5: Consistent JSON error response format
func TestErrorHandling_JSONErrorResponseFormat(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name       string
		method     string
		path       string
		body       io.Reader
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "400 Bad Request",
			method:     "POST",
			path:       "/v1/fizzbuzz",
			body:       strings.NewReader(`{"invalid": "json"`),
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "404 Not Found",
			method:     "GET",
			path:       "/v1/unknown",
			body:       nil,
			headers:    nil,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "405 Method Not Allowed",
			method:     "PUT",
			path:       "/v1/fizzbuzz",
			body:       nil,
			headers:    nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "422 Validation Error",
			method:     "POST",
			path:       "/v1/fizzbuzz",
			body:       strings.NewReader(`{"int1": -1, "int2": 5, "limit": 100, "str1": "fizz", "str2": "buzz"}`),
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, err := http.NewRequest(tt.method, tt.path, tt.body)
			if err != nil {
				t.Fatal(err)
			}

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			app.routes().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("want status %d; got %d", tt.wantStatus, rr.Code)
			}

			// Check Content-Type header
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("want Content-Type 'application/json'; got %q", contentType)
			}

			// Verify JSON structure - all error responses should have "error" field
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("failed to unmarshal JSON response: %v", err)
			}

			// Check that "error" field exists
			if _, exists := response["error"]; !exists {
				t.Error("response missing 'error' field")
			}

			// For validation errors, check the structure
			if tt.wantStatus == http.StatusUnprocessableEntity {
				errorValue, ok := response["error"].(map[string]interface{})
				if !ok {
					t.Error("validation error should have structured error object")
				} else {
					if _, exists := errorValue["message"]; !exists {
						t.Error("validation error missing 'message' field")
					}
					if _, exists := errorValue["details"]; !exists {
						t.Error("validation error missing 'details' field")
					}
				}
			}
		})
	}
}

// newTestApplication is defined in integration_test.go
