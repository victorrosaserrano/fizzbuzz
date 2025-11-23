package data

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHealthCheckResponse_JSON_Marshaling(t *testing.T) {
	// Test complete health check response
	response := HealthCheckResponse{
		Status: "available",
		SystemInfo: SystemInfo{
			Environment: "test",
			Version:     "1.0.0",
			Timestamp:   "2025-11-23T12:00:00Z",
		},
		Database: &DatabaseHealthInfo{
			Status:         "connected",
			ResponseTimeMs: 5,
			ActiveConns:    3,
			IdleConns:      2,
			MaxConns:       10,
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal HealthCheckResponse: %v", err)
	}

	// Unmarshal back
	var unmarshaled HealthCheckResponse
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal HealthCheckResponse: %v", err)
	}

	// Verify fields
	if unmarshaled.Status != "available" {
		t.Errorf("Expected status 'available', got '%s'", unmarshaled.Status)
	}

	if unmarshaled.SystemInfo.Environment != "test" {
		t.Errorf("Expected environment 'test', got '%s'", unmarshaled.SystemInfo.Environment)
	}

	if unmarshaled.SystemInfo.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", unmarshaled.SystemInfo.Version)
	}

	if unmarshaled.Database == nil {
		t.Errorf("Expected database info, got nil")
	} else {
		if unmarshaled.Database.Status != "connected" {
			t.Errorf("Expected database status 'connected', got '%s'", unmarshaled.Database.Status)
		}

		if unmarshaled.Database.ResponseTimeMs != 5 {
			t.Errorf("Expected response time 5ms, got %d", unmarshaled.Database.ResponseTimeMs)
		}

		if unmarshaled.Database.ActiveConns != 3 {
			t.Errorf("Expected active connections 3, got %d", unmarshaled.Database.ActiveConns)
		}

		if unmarshaled.Database.MaxConns != 10 {
			t.Errorf("Expected max connections 10, got %d", unmarshaled.Database.MaxConns)
		}
	}
}

func TestHealthCheckResponse_JSON_WithoutDatabase(t *testing.T) {
	// Test health check response without database info (omitempty should work)
	response := HealthCheckResponse{
		Status: "degraded",
		SystemInfo: SystemInfo{
			Environment: "production",
			Version:     "1.2.3",
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		},
		Database: nil, // No database info
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal HealthCheckResponse: %v", err)
	}

	// Parse as map to check JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify database field is omitted
	if _, exists := jsonMap["database"]; exists {
		t.Errorf("Expected database field to be omitted when nil, but it was present")
	}

	// Verify other fields are present
	if jsonMap["status"] != "degraded" {
		t.Errorf("Expected status 'degraded', got %v", jsonMap["status"])
	}

	systemInfo, exists := jsonMap["system_info"]
	if !exists {
		t.Errorf("Expected system_info field to be present")
	} else {
		sysInfo := systemInfo.(map[string]interface{})
		if sysInfo["environment"] != "production" {
			t.Errorf("Expected environment 'production', got %v", sysInfo["environment"])
		}
	}
}

func TestSystemInfo_JSON_Structure(t *testing.T) {
	systemInfo := SystemInfo{
		Environment: "staging",
		Version:     "2.1.0",
		Timestamp:   "2025-11-23T14:30:00Z",
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(systemInfo)
	if err != nil {
		t.Fatalf("Failed to marshal SystemInfo: %v", err)
	}

	// Expected JSON structure
	expectedJSON := `{"environment":"staging","version":"2.1.0","timestamp":"2025-11-23T14:30:00Z"}`

	// Parse both as maps for comparison
	var actual, expected map[string]interface{}

	err = json.Unmarshal(jsonBytes, &actual)
	if err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}

	err = json.Unmarshal([]byte(expectedJSON), &expected)
	if err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}

	// Compare fields
	for key, expectedValue := range expected {
		if actual[key] != expectedValue {
			t.Errorf("Field %s: expected %v, got %v", key, expectedValue, actual[key])
		}
	}
}

func TestDatabaseHealthInfo_JSON_Structure(t *testing.T) {
	dbInfo := DatabaseHealthInfo{
		Status:         "connected",
		ResponseTimeMs: 12,
		ActiveConns:    7,
		IdleConns:      4,
		MaxConns:       20,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(dbInfo)
	if err != nil {
		t.Fatalf("Failed to marshal DatabaseHealthInfo: %v", err)
	}

	// Parse as map to check structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify all expected fields are present
	expectedFields := map[string]interface{}{
		"status":             "connected",
		"response_time_ms":   float64(12), // JSON numbers are float64
		"active_connections": float64(7),
		"idle_connections":   float64(4),
		"max_connections":    float64(20),
	}

	for field, expected := range expectedFields {
		if actual, exists := jsonMap[field]; !exists {
			t.Errorf("Expected field '%s' to be present", field)
		} else if actual != expected {
			t.Errorf("Field '%s': expected %v, got %v", field, expected, actual)
		}
	}
}

func TestHealthCheckResponse_AcceptanceCriteria_Available(t *testing.T) {
	// Test the exact response format specified in acceptance criteria
	response := HealthCheckResponse{
		Status: "available",
		SystemInfo: SystemInfo{
			Environment: "production",
			Version:     "1.0.0",
		},
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Parse as map to verify structure matches AC
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify structure matches acceptance criteria format
	if jsonMap["status"] != "available" {
		t.Errorf("Status should be 'available', got %v", jsonMap["status"])
	}

	systemInfo, exists := jsonMap["system_info"]
	if !exists {
		t.Errorf("system_info field should be present")
	} else {
		sysInfo := systemInfo.(map[string]interface{})
		if sysInfo["environment"] != "production" {
			t.Errorf("environment should be 'production', got %v", sysInfo["environment"])
		}
		if sysInfo["version"] != "1.0.0" {
			t.Errorf("version should be '1.0.0', got %v", sysInfo["version"])
		}
	}
}
