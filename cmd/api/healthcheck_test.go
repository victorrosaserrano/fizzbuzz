package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fizzbuzz/internal/data"
	"fizzbuzz/internal/jsonlog"
)

// mockStatisticsHandler provides a mock implementation for testing health check functionality
type mockStatisticsHandler struct {
	shouldFailHealth bool
	healthResponse   map[string]interface{}
}

func (m *mockStatisticsHandler) GetDatabaseHealth(ctx context.Context) (map[string]interface{}, error) {
	if m.shouldFailHealth {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "mock database connection failed",
		}, errors.New("mock database error")
	}

	if m.healthResponse != nil {
		return m.healthResponse, nil
	}

	// Default healthy response with connection metrics
	return map[string]interface{}{
		"status":               "healthy",
		"total_connections":    int32(10),
		"idle_connections":     int32(3),
		"acquired_connections": int32(2),
	}, nil
}

func (m *mockStatisticsHandler) Record(ctx context.Context, input *data.FizzBuzzInput) error {
	return nil
}

func (m *mockStatisticsHandler) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	return nil, nil
}

func (m *mockStatisticsHandler) GetPoolStats(ctx context.Context) (*data.PoolStats, error) {
	return &data.PoolStats{
		TotalConnections:  10,
		IdleConnections:   3,
		ActiveConnections: 2,
		MaxConnections:    25,
		Status:            "healthy",
		CollectedAt:       time.Now(),
	}, nil
}

func (m *mockStatisticsHandler) RecordLegacy(input *data.FizzBuzzInput, logger *jsonlog.Logger) {
}

func (m *mockStatisticsHandler) GetMostFrequentLegacy(logger *jsonlog.Logger) *data.StatisticsEntry {
	return nil
}

func (m *mockStatisticsHandler) Close() error {
	return nil
}

func TestHealthCheckHandler_GET_Success(t *testing.T) {
	// Setup test application with mock handler
	app := &application{
		config: config{
			env: "test",
		},
		logger:     jsonlog.New(io.Discard, jsonlog.LevelInfo, "test"),
		statistics: &mockStatisticsHandler{},
	}

	// Create test request
	req, err := http.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record response
	rr := httptest.NewRecorder()
	app.healthcheckHandler(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response body
	var response envelope
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Validate data structure
	dataInterface, exists := response["data"]
	if !exists {
		t.Errorf("Response missing 'data' field")
	}

	data, ok := dataInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Response 'data' field is not a map")
	}

	// Validate status
	status, exists := data["status"]
	if !exists {
		t.Errorf("Response data missing 'status' field")
	}
	if status != "available" {
		t.Errorf("Expected status 'available', got %v", status)
	}

	// Validate system_info
	systemInfo, exists := data["system_info"]
	if !exists {
		t.Errorf("Response data missing 'system_info' field")
	}

	sysInfo, ok := systemInfo.(map[string]interface{})
	if !ok {
		t.Errorf("system_info is not a map")
	}

	// Check environment
	env := sysInfo["environment"]
	if env != "test" {
		t.Errorf("Expected environment 'test', got %v", env)
	}

	// Check version exists
	version := sysInfo["version"]
	if version == nil {
		t.Errorf("Version field missing")
	}

	// Check timestamp exists and is valid
	timestamp := sysInfo["timestamp"]
	if timestamp == nil {
		t.Errorf("Timestamp field missing")
	}

	// Validate database field exists
	database, exists := data["database"]
	if !exists {
		t.Errorf("Response data missing 'database' field")
	}

	dbInfo, ok := database.(map[string]interface{})
	if !ok {
		t.Errorf("database field is not a map")
	}

	// Validate database status
	dbStatus := dbInfo["status"]
	if dbStatus != "connected" {
		t.Errorf("Expected database status 'connected', got %v", dbStatus)
	}
}

func TestHealthCheckHandler_DatabaseUnhealthy_Degraded(t *testing.T) {
	// Setup test application with failing database
	app := &application{
		config: config{
			env: "test",
		},
		logger:     jsonlog.New(io.Discard, jsonlog.LevelInfo, "test"),
		statistics: &mockStatisticsHandler{shouldFailHealth: true},
	}

	// Create test request
	req, err := http.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record response
	rr := httptest.NewRecorder()
	app.healthcheckHandler(rr, req)

	// Check status code - should be 503 for degraded state
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}

	// Parse response body
	var response envelope
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Validate degraded status
	dataInterface, exists := response["data"]
	if !exists {
		t.Errorf("Response missing 'data' field")
	}

	data, ok := dataInterface.(map[string]interface{})
	if !ok {
		t.Errorf("Response 'data' field is not a map")
	}

	status, exists := data["status"]
	if !exists {
		t.Errorf("Response data missing 'status' field")
	}
	if status != "degraded" {
		t.Errorf("Expected status 'degraded', got %v", status)
	}

	// Validate database is marked disconnected
	database, exists := data["database"]
	if !exists {
		t.Errorf("Response data missing 'database' field")
	}

	dbInfo, ok := database.(map[string]interface{})
	if !ok {
		t.Errorf("database field is not a map")
	}

	dbStatus := dbInfo["status"]
	if dbStatus != "disconnected" {
		t.Errorf("Expected database status 'disconnected', got %v", dbStatus)
	}
}

func TestHealthCheckHandler_MethodNotAllowed(t *testing.T) {
	app := &application{
		config: config{env: "test"},
		logger: jsonlog.New(io.Discard, jsonlog.LevelInfo, "test"),
	}

	// Test with POST method (should be rejected)
	req, err := http.NewRequest(http.MethodPost, "/v1/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	app.healthcheckHandler(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHealthCheckHandler_ResponseTime_Performance(t *testing.T) {
	// Setup test application
	app := &application{
		config: config{
			env: "test",
		},
		logger:     jsonlog.New(io.Discard, jsonlog.LevelInfo, "test"),
		statistics: &mockStatisticsHandler{},
	}

	// Measure response time
	start := time.Now()

	req, err := http.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	app.healthcheckHandler(rr, req)

	responseTime := time.Since(start)

	// Check that response time is reasonable (should be well under 1ms for mock)
	if responseTime > 50*time.Millisecond {
		t.Errorf("Health check response time too slow: %v (expected < 50ms for mock)", responseTime)
	}

	// Verify successful response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected successful response, got status %d", rr.Code)
	}
}

func TestHealthCheckHandler_ConnectionPoolMetrics(t *testing.T) {
	// Setup test application with specific connection metrics
	mockHandler := &mockStatisticsHandler{
		healthResponse: map[string]interface{}{
			"status":               "healthy",
			"total_connections":    int32(25),
			"idle_connections":     int32(8),
			"acquired_connections": int32(5),
		},
	}

	app := &application{
		config:     config{env: "test"},
		logger:     jsonlog.New(io.Discard, jsonlog.LevelInfo, "test"),
		statistics: mockHandler,
	}

	req, err := http.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	app.healthcheckHandler(rr, req)

	// Parse response
	var response envelope
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Navigate to database section
	data := response["data"].(map[string]interface{})
	database := data["database"].(map[string]interface{})

	// Validate connection pool metrics are included
	if maxConns := database["max_connections"]; maxConns != float64(25) {
		t.Errorf("Expected max_connections 25, got %v", maxConns)
	}

	if idleConns := database["idle_connections"]; idleConns != float64(8) {
		t.Errorf("Expected idle_connections 8, got %v", idleConns)
	}

	if activeConns := database["active_connections"]; activeConns != float64(5) {
		t.Errorf("Expected active_connections 5, got %v", activeConns)
	}
}
