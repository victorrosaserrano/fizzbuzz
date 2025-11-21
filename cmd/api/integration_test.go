package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestApplication creates a test application instance with a test logger
func newTestApplication(t *testing.T) *application {
	// Create a test logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return &application{
		config: config{
			port: 4000,
			env:  "test",
		},
		logger: logger,
	}
}

func TestHealthcheckHandler(t *testing.T) {
	app := newTestApplication(t)

	t.Run("GET /v1/healthcheck returns correct response", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		expected := `{
	"status": "available",
	"system_info": {
		"environment": "test",
		"version": "1.0.0"
	}
}
`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expected) {
			t.Errorf("expected body %s, got %s", expected, rr.Body.String())
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("POST /v1/healthcheck returns method not allowed", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/v1/healthcheck", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}

		expectedError := `{
	"error": "the POST method is not supported for this resource"
}
`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedError) {
			t.Errorf("expected body %s, got %s", expectedError, rr.Body.String())
		}
	})
}

func TestNotFoundHandler(t *testing.T) {
	app := newTestApplication(t)

	req, err := http.NewRequest(http.MethodGet, "/v1/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := app.routes()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	expectedError := `{
	"error": "the requested resource could not be found"
}
`
	if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedError) {
		t.Errorf("expected body %s, got %s", expectedError, rr.Body.String())
	}
}

func TestMiddleware(t *testing.T) {
	app := newTestApplication(t)

	t.Run("panic recovery middleware works", func(t *testing.T) {
		// Create a handler that panics
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		// Apply middleware
		handler := app.recoverPanic(panicHandler)

		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d after panic recovery, got %d", http.StatusInternalServerError, rr.Code)
		}

		expectedError := `{
	"error": "the server encountered a problem and could not process your request"
}
`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedError) {
			t.Errorf("expected body %s, got %s", expectedError, rr.Body.String())
		}

		// Verify Connection: close header is set
		connectionHeader := rr.Header().Get("Connection")
		if connectionHeader != "close" {
			t.Errorf("expected Connection: close header after panic, got %s", connectionHeader)
		}
	})

	t.Run("request logging middleware captures response", func(t *testing.T) {
		// Create a simple handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte("test response"))
		})

		// Apply logging middleware
		handler := app.logRequest(testHandler)

		req, err := http.NewRequest(http.MethodGet, "/test", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusTeapot {
			t.Errorf("expected status %d, got %d", http.StatusTeapot, rr.Code)
		}

		if rr.Body.String() != "test response" {
			t.Errorf("expected body 'test response', got %s", rr.Body.String())
		}
	})
}

func TestJSONHelpers(t *testing.T) {
	app := newTestApplication(t)

	t.Run("writeJSON helper creates proper JSON response", func(t *testing.T) {
		rr := httptest.NewRecorder()

		data := envelope{
			"test":   "value",
			"number": 42,
		}

		err := app.writeJSON(rr, http.StatusCreated, data, nil)
		if err != nil {
			t.Fatal(err)
		}

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}

		expectedJSON := `{
	"number": 42,
	"test": "value"
}
`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedJSON) {
			t.Errorf("expected JSON %s, got %s", expectedJSON, rr.Body.String())
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("writeJSON with custom headers", func(t *testing.T) {
		rr := httptest.NewRecorder()

		data := envelope{"message": "test"}
		headers := http.Header{}
		headers.Set("X-Custom-Header", "test-value")

		err := app.writeJSON(rr, http.StatusOK, data, headers)
		if err != nil {
			t.Fatal(err)
		}

		customHeader := rr.Header().Get("X-Custom-Header")
		if customHeader != "test-value" {
			t.Errorf("expected X-Custom-Header test-value, got %s", customHeader)
		}
	})

	t.Run("readJSON helper parses valid JSON", func(t *testing.T) {
		jsonBody := `{"test": "value", "number": 42}`
		req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(jsonBody))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		var result map[string]interface{}
		err = app.readJSON(rr, req, &result)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if result["test"] != "value" {
			t.Errorf("expected test field to be 'value', got %v", result["test"])
		}

		if result["number"] != float64(42) {
			t.Errorf("expected number field to be 42, got %v", result["number"])
		}
	})

	t.Run("readJSON rejects malformed JSON", func(t *testing.T) {
		malformedJSON := `{"test": value}` // missing quotes around value
		req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(malformedJSON))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		var result map[string]interface{}
		err = app.readJSON(rr, req, &result)
		if err == nil {
			t.Error("expected error for malformed JSON, got nil")
		}
	})

	t.Run("readJSON rejects multiple JSON values", func(t *testing.T) {
		multipleJSON := `{"first": "value"}{"second": "value"}`
		req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(multipleJSON))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		var result map[string]interface{}
		err = app.readJSON(rr, req, &result)
		if err == nil {
			t.Error("expected error for multiple JSON values, got nil")
		}

		expectedErr := "body must only contain a single JSON value"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
	})
}

func TestErrorResponses(t *testing.T) {
	app := newTestApplication(t)

	t.Run("errorJSON creates proper error response", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		app.errorJSON(rr, req, http.StatusBadRequest, "test error message")

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		expectedError := `{
	"error": "test error message"
}
`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedError) {
			t.Errorf("expected body %s, got %s", expectedError, rr.Body.String())
		}
	})

	t.Run("badRequestResponse works correctly", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		testErr := errors.New("bad request error")
		app.badRequestResponse(rr, req, testErr)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		expectedError := `{
	"error": "bad request error"
}
`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedError) {
			t.Errorf("expected body %s, got %s", expectedError, rr.Body.String())
		}
	})
}

func TestResponseRecorder(t *testing.T) {
	t.Run("responseRecorder captures status code", func(t *testing.T) {
		rr := httptest.NewRecorder()
		recorder := &responseRecorder{ResponseWriter: rr, statusCode: http.StatusOK}

		// Test default status code
		if recorder.statusCode != http.StatusOK {
			t.Errorf("expected default status %d, got %d", http.StatusOK, recorder.statusCode)
		}

		// Test WriteHeader
		recorder.WriteHeader(http.StatusCreated)
		if recorder.statusCode != http.StatusCreated {
			t.Errorf("expected status %d after WriteHeader, got %d", http.StatusCreated, recorder.statusCode)
		}
	})
}

// TestServerIntegration tests the full HTTP server integration
func TestServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	app := newTestApplication(t)

	// Create a test server
	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	t.Run("full HTTP client integration test", func(t *testing.T) {
		// Test healthcheck endpoint
		resp, err := http.Get(ts.URL + "/v1/healthcheck")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		expected := `{
	"status": "available",
	"system_info": {
		"environment": "test",
		"version": "1.0.0"
	}
}
`
		if strings.TrimSpace(string(body)) != strings.TrimSpace(expected) {
			t.Errorf("expected body %s, got %s", expected, string(body))
		}

		// Verify headers
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("404 for unknown endpoints", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/unknown/endpoint")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
	})

	t.Run("405 for wrong methods", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/v1/healthcheck", "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
		}
	})
}

func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	app := newTestApplication(t)
	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	// Test concurrent requests to ensure thread safety
	numRequests := 50
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get(ts.URL + "/v1/healthcheck")
			if err != nil {
				t.Errorf("request failed: %v", err)
				done <- false
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
				done <- false
				return
			}

			done <- true
		}()
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != numRequests {
		t.Errorf("expected %d successful requests, got %d", numRequests, successCount)
	}
}
