package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"fizzbuzz/internal/validator"
)

// TestValidatorIntegrationWithErrorHelpers tests integration between validator and existing error helpers
func TestValidatorIntegrationWithErrorHelpers(t *testing.T) {
	// Create a mock application for testing
	app := &application{}

	t.Run("validation errors integrate with errorJSON helper", func(t *testing.T) {
		// Create a validator with errors
		v := validator.New()
		v.AddError("int1", "must be a positive integer")
		v.AddError("str1", "must be provided")

		// Create test HTTP request and response
		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()

		// Create error response using existing helper with validator errors
		errorDetails := map[string]any{
			"message": "validation failed",
			"details": v.ErrorMap(),
		}

		app.errorJSON(w, req, http.StatusUnprocessableEntity, errorDetails)

		// Verify response
		res := w.Result()
		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", res.StatusCode)
		}

		if contentType := res.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected application/json content type, got %s", contentType)
		}

		// Parse response body
		var response envelope
		err := json.NewDecoder(res.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify envelope structure
		errorInfo, exists := response["error"].(map[string]any)
		if !exists {
			t.Error("Expected error section in response envelope")
		}

		message, exists := errorInfo["message"].(string)
		if !exists || message != "validation failed" {
			t.Errorf("Expected validation failed message, got %v", message)
		}

		details, exists := errorInfo["details"].(map[string]any)
		if !exists {
			t.Error("Expected details in error section")
		}

		// Verify specific validation errors are present
		if details["int1"] != "must be a positive integer" {
			t.Errorf("Expected int1 error, got %v", details["int1"])
		}

		if details["str1"] != "must be provided" {
			t.Errorf("Expected str1 error, got %v", details["str1"])
		}
	})

	t.Run("empty validator integrates cleanly", func(t *testing.T) {
		// Empty validator should not generate errors
		v := validator.New()
		errorMap := v.ErrorMap()

		if errorMap != nil {
			t.Error("Empty validator should return nil error map")
		}

		// This enables clean handler logic:
		// if errorMap := v.ErrorMap(); errorMap != nil {
		//     app.errorJSON(w, r, 422, map[string]any{"message": "validation failed", "details": errorMap})
		//     return
		// }
		// ... continue with success logic
	})

	t.Run("validation helper pattern for handlers", func(t *testing.T) {
		// Simulate typical handler validation pattern

		// Mock input data
		input := struct {
			Int1  int    `json:"int1"`
			Int2  int    `json:"int2"`
			Limit int    `json:"limit"`
			Str1  string `json:"str1"`
			Str2  string `json:"str2"`
		}{
			Int1:  -1,                       // Invalid
			Int2:  -1,                       // Invalid
			Limit: 0,                        // Invalid
			Str1:  "",                       // Invalid
			Str2:  string(make([]byte, 60)), // Invalid
		}

		// Validation using new framework
		v := validator.New()
		v.Check(input.Int1 > 0, "int1", "must be a positive integer")
		v.Check(input.Int2 > 0, "int2", "must be a positive integer")
		v.Check(input.Limit > 0, "limit", "must be a positive integer")
		v.Check(input.Str1 != "", "str1", "must be provided")
		v.Check(len(input.Str2) <= 50, "str2", "must not be more than 50 characters")

		// Check for validation errors and create response
		if !v.Valid() {
			req := httptest.NewRequest("POST", "/test", nil)
			w := httptest.NewRecorder()

			errorDetails := map[string]any{
				"message": "validation failed",
				"details": v.ErrorMap(),
			}

			app.errorJSON(w, req, http.StatusUnprocessableEntity, errorDetails)

			// Verify the response contains all expected validation errors
			res := w.Result()
			if res.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("Expected 422 status, got %d", res.StatusCode)
			}

			var response envelope
			json.NewDecoder(res.Body).Decode(&response)

			errorInfo := response["error"].(map[string]any)
			details := errorInfo["details"].(map[string]any)

			// Should have 5 validation errors
			if len(details) != 5 {
				t.Errorf("Expected 5 validation errors, got %d", len(details))
			}
		} else {
			t.Error("Validation should have failed for invalid input")
		}
	})
}

// TestValidationErrorResponseFormat tests the exact format expected by clients
func TestValidationErrorResponseFormat(t *testing.T) {
	app := &application{}

	t.Run("422 response format matches API contract", func(t *testing.T) {
		v := validator.New()
		v.AddError("int1", "must be a positive integer")
		v.AddError("limit", "must not be more than 100,000")

		req := httptest.NewRequest("POST", "/v1/fizzbuzz", nil)
		w := httptest.NewRecorder()

		errorDetails := map[string]any{
			"message": "validation failed",
			"details": v.ErrorMap(),
		}

		app.errorJSON(w, req, http.StatusUnprocessableEntity, errorDetails)

		// Parse response
		var response envelope
		json.NewDecoder(w.Result().Body).Decode(&response)

		// Verify exact format matches API documentation
		expectedStructure := map[string]any{
			"error": map[string]any{
				"message": "validation failed",
				"details": map[string]any{
					"int1":  "must be a positive integer",
					"limit": "must not be more than 100,000",
				},
			},
		}

		// Check top-level structure
		if _, exists := response["error"]; !exists {
			t.Error("Response should have error section")
		}

		errorSection := response["error"].(map[string]any)
		if errorSection["message"] != "validation failed" {
			t.Errorf("Expected validation failed message, got %v", errorSection["message"])
		}

		details := errorSection["details"].(map[string]any)
		expectedDetails := expectedStructure["error"].(map[string]any)["details"].(map[string]any)

		for key, expectedMsg := range expectedDetails {
			if details[key] != expectedMsg {
				t.Errorf("Field %s: expected %v, got %v", key, expectedMsg, details[key])
			}
		}
	})
}
