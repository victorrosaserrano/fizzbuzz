package main

import (
	"bytes"
	"encoding/json"
	"fizzbuzz/internal/jsonlog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCorrelationIDMiddleware(t *testing.T) {
	app := &application{
		logger: jsonlog.New(&bytes.Buffer{}, jsonlog.LevelInfo, "production"),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := r.Context().Value("correlation_id")
		if corrID == nil {
			t.Error("correlation_id not found in context")
		}

		corrIDStr, ok := corrID.(string)
		if !ok {
			t.Error("correlation_id is not a string")
		}

		if corrIDStr == "" {
			t.Error("correlation_id is empty")
		}

		// Validate UUID format
		if _, err := uuid.Parse(corrIDStr); err != nil {
			t.Errorf("correlation_id is not a valid UUID: %s", corrIDStr)
		}

		w.WriteHeader(http.StatusOK)
	})

	middleware := app.correlationID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	corrIDHeader := w.Header().Get("X-Correlation-ID")
	if corrIDHeader == "" {
		t.Error("X-Correlation-ID header not set")
	}

	if _, err := uuid.Parse(corrIDHeader); err != nil {
		t.Errorf("X-Correlation-ID header is not a valid UUID: %s", corrIDHeader)
	}
}

func TestCorrelationIDMiddleware_ExistingHeader(t *testing.T) {
	app := &application{
		logger: jsonlog.New(&bytes.Buffer{}, jsonlog.LevelInfo, "production"),
	}

	existingID := "existing-correlation-id"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := r.Context().Value("correlation_id")
		if corrID != existingID {
			t.Errorf("expected correlation_id '%s', got '%v'", existingID, corrID)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := app.correlationID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Correlation-ID", existingID)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	corrIDHeader := w.Header().Get("X-Correlation-ID")
	if corrIDHeader != existingID {
		t.Errorf("expected X-Correlation-ID header '%s', got '%s'", existingID, corrIDHeader)
	}
}

func TestLogRequestMiddleware(t *testing.T) {
	var logBuf bytes.Buffer
	app := &application{
		logger: jsonlog.New(&logBuf, jsonlog.LevelInfo, "production"),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Set up logging first, then correlation ID
	logHandler := app.logRequest(handler)
	middleware := app.correlationID(logHandler)

	req := httptest.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader("{}"))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Parse log output - get the last log entry (request completed)
	logLines := strings.Split(strings.TrimSpace(logBuf.String()), "\n")
	if len(logLines) == 0 {
		t.Fatal("no log output found")
	}

	var logEntry map[string]any
	err := json.Unmarshal([]byte(logLines[len(logLines)-1]), &logEntry)
	if err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if logEntry["level"] != "INFO" {
		t.Errorf("expected log level INFO, got %v", logEntry["level"])
	}

	if logEntry["msg"] != "HTTP request completed" {
		t.Errorf("expected log message 'HTTP request completed', got %v", logEntry["msg"])
	}

	if logEntry["method"] != "POST" {
		t.Errorf("expected method POST, got %v", logEntry["method"])
	}

	if logEntry["uri"] != "/v1/fizzbuzz" {
		t.Errorf("expected URI /v1/fizzbuzz, got %v", logEntry["uri"])
	}

	if logEntry["status"] != float64(200) {
		t.Errorf("expected status 200, got %v", logEntry["status"])
	}

	if logEntry["correlation_id"] == nil {
		t.Error("correlation_id should be present in log")
	}

	if _, ok := logEntry["duration_ms"]; !ok {
		t.Error("duration_ms should be present in log")
	}
}

func TestRecoverPanicMiddleware(t *testing.T) {
	var logBuf bytes.Buffer
	app := &application{
		logger: jsonlog.New(&logBuf, jsonlog.LevelInfo, "production"),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Set up panic recovery first, then correlation ID
	panicHandler := app.recoverPanic(handler)
	middleware := app.correlationID(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 after panic, got %d", w.Code)
	}

	connectionHeader := w.Header().Get("Connection")
	if connectionHeader != "close" {
		t.Errorf("expected Connection header 'close', got '%s'", connectionHeader)
	}

	// Parse log output - get the first log entry (panic recovered)
	logLines := strings.Split(strings.TrimSpace(logBuf.String()), "\n")
	if len(logLines) == 0 {
		t.Fatal("no log output found")
	}

	var logEntry map[string]any
	err := json.Unmarshal([]byte(logLines[0]), &logEntry)
	if err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if logEntry["level"] != "ERROR" {
		t.Errorf("expected log level ERROR, got %v", logEntry["level"])
	}

	if logEntry["msg"] != "panic recovered" {
		t.Errorf("expected log message 'panic recovered', got %v", logEntry["msg"])
	}

	if logEntry["panic"] != "test panic" {
		t.Errorf("expected panic 'test panic', got %v", logEntry["panic"])
	}

	if logEntry["correlation_id"] == nil {
		t.Error("correlation_id should be present in log")
	}
}

func TestFullMiddlewareChain(t *testing.T) {
	var logBuf bytes.Buffer
	app := &application{
		logger: jsonlog.New(&logBuf, jsonlog.LevelInfo, "production"),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate checking correlation ID in handler
		corrID := r.Context().Value("correlation_id")
		if corrID == nil {
			t.Error("correlation_id not available in handler context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Full middleware chain as in routes.go
	middleware := app.recoverPanic(app.correlationID(app.logRequest(handler)))

	req := httptest.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that log was generated
	if logBuf.Len() == 0 {
		t.Error("expected log output from middleware chain")
	}

	// Check correlation ID header was set
	corrIDHeader := w.Header().Get("X-Correlation-ID")
	if corrIDHeader == "" {
		t.Error("X-Correlation-ID header should be set")
	}
}
