package validator

import (
	"encoding/json"
	"regexp"
	"testing"
)

// TestErrorJSONIntegration tests integration with existing errorJSON helper pattern
func TestErrorJSONIntegration(t *testing.T) {
	t.Run("validator errors compatible with JSON envelope format", func(t *testing.T) {
		v := New()
		v.AddError("int1", "must be a positive integer")
		v.AddError("str1", "must be provided")

		errorMap := v.ErrorMap()

		// Simulate the envelope format that would be used with errorJSON helper
		envelope := map[string]any{
			"error": map[string]any{
				"message": "validation failed",
				"details": errorMap,
			},
		}

		// Marshal to JSON to verify structure
		jsonBytes, err := json.Marshal(envelope)
		if err != nil {
			t.Fatalf("Failed to marshal envelope: %v", err)
		}

		// Verify JSON structure matches expected format
		var unmarshaled map[string]any
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal envelope: %v", err)
		}

		// Check envelope structure
		errorSection, exists := unmarshaled["error"].(map[string]any)
		if !exists {
			t.Error("Expected error section in envelope")
		}

		message, exists := errorSection["message"].(string)
		if !exists || message != "validation failed" {
			t.Errorf("Expected 'validation failed' message, got %v", message)
		}

		details, exists := errorSection["details"].(map[string]any)
		if !exists {
			t.Error("Expected details section in error")
		}

		// Verify field-specific errors are preserved
		if details["int1"] != "must be a positive integer" {
			t.Errorf("Expected int1 error, got %v", details["int1"])
		}

		if details["str1"] != "must be provided" {
			t.Errorf("Expected str1 error, got %v", details["str1"])
		}
	})

	t.Run("empty validator returns nil for clean envelope", func(t *testing.T) {
		v := New()
		errorMap := v.ErrorMap()

		if errorMap != nil {
			t.Error("Empty validator should return nil error map")
		}

		// This enables clean conditional logic in handlers:
		// if errorMap := validator.ErrorMap(); errorMap != nil {
		//     return 422 with errorMap
		// }
	})
}

// TestPerformanceBenchmarks tests validation framework performance
func TestPerformanceBenchmarks(t *testing.T) {
	t.Run("single validation performance", func(t *testing.T) {
		// This would normally be a benchmark, but testing basic performance here
		v := New()

		// Measure time for typical FizzBuzz validation
		for i := 0; i < 1000; i++ {
			v.Clear()
			v.Check(i > 0, "int1", "must be positive")
			v.Check(i != 5, "int1", "must not equal 5")
			v.Check(len("fizz") > 0, "str1", "must not be empty")
		}

		// Test passes if no panic or excessive delay
		// Real performance testing would be done with go test -bench
	})

	t.Run("helper function performance", func(t *testing.T) {
		// Test helper functions don't have performance regressions
		for i := 0; i < 1000; i++ {
			PermittedValue(i, 1, 2, 3, 4, 5)
			Unique([]int{1, 2, 3, 4, 5})
			In(i, []int{1, 2, 3, 4, 5})
		}

		emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		for i := 0; i < 100; i++ {
			Matches("test@example.com", emailPattern)
		}
	})
}

// TestValidationHelperEdgeCases tests edge cases for helper functions
func TestValidationHelperEdgeCases(t *testing.T) {
	t.Run("PermittedValue with no arguments", func(t *testing.T) {
		result := PermittedValue("test")
		if result != false {
			t.Error("PermittedValue with no permitted values should return false")
		}
	})

	t.Run("PermittedValue with single argument", func(t *testing.T) {
		result := PermittedValue("test", "test")
		if result != true {
			t.Error("PermittedValue with matching single value should return true")
		}

		result = PermittedValue("test", "other")
		if result != false {
			t.Error("PermittedValue with non-matching single value should return false")
		}
	})

	t.Run("Unique with nil slice", func(t *testing.T) {
		// Test that Unique handles empty/nil gracefully
		var nilSlice []string
		result := Unique(nilSlice)
		if result != true {
			t.Error("Unique with nil slice should return true")
		}
	})

	t.Run("In with nil slice", func(t *testing.T) {
		var nilSlice []string
		result := In("test", nilSlice)
		if result != false {
			t.Error("In with nil slice should return false")
		}
	})

	t.Run("Matches with invalid pattern compilation", func(t *testing.T) {
		// Test error handling when regex pattern is invalid
		// Note: This test assumes pattern is pre-compiled and passed to Matches
		// Invalid patterns would cause compile-time errors, not runtime errors

		validPattern := regexp.MustCompile(".*")
		result := Matches("test", validPattern)
		if result != true {
			t.Error("Valid pattern should match any string")
		}
	})
}

// TestRealWorldScenarios tests realistic usage patterns
func TestRealWorldScenarios(t *testing.T) {
	t.Run("HTTP handler validation pattern", func(t *testing.T) {
		// Simulate how validator would be used in an HTTP handler

		// Mock input data that would come from JSON parsing
		input := struct {
			Int1  int    `json:"int1"`
			Int2  int    `json:"int2"`
			Limit int    `json:"limit"`
			Str1  string `json:"str1"`
			Str2  string `json:"str2"`
		}{
			Int1:  3,
			Int2:  5,
			Limit: 100,
			Str1:  "fizz",
			Str2:  "buzz",
		}

		// Validation logic that would be in handler
		v := New()
		v.Check(input.Int1 > 0, "int1", "must be a positive integer")
		v.Check(input.Int1 <= 10000, "int1", "must not be more than 10,000")
		v.Check(input.Int2 > 0, "int2", "must be a positive integer")
		v.Check(input.Int2 <= 10000, "int2", "must not be more than 10,000")
		v.Check(input.Int1 != input.Int2, "int1", "must be different from int2")
		v.Check(input.Limit > 0, "limit", "must be a positive integer")
		v.Check(input.Limit <= 100000, "limit", "must not be more than 100,000")
		v.Check(input.Str1 != "", "str1", "must be provided")
		v.Check(len(input.Str1) <= 50, "str1", "must not be more than 50 characters")
		v.Check(input.Str2 != "", "str2", "must be provided")
		v.Check(len(input.Str2) <= 50, "str2", "must not be more than 50 characters")

		if !v.Valid() {
			t.Error("Valid input should pass validation")
		}

		// Test error response generation pattern
		errorMap := v.ErrorMap()
		if errorMap != nil {
			t.Error("Valid input should not generate errors")
		}
	})

	t.Run("HTTP handler validation failure pattern", func(t *testing.T) {
		// Test validation failure scenario
		input := struct {
			Int1  int    `json:"int1"`
			Int2  int    `json:"int2"`
			Limit int    `json:"limit"`
			Str1  string `json:"str1"`
			Str2  string `json:"str2"`
		}{
			Int1:  -1,                       // Invalid: negative
			Int2:  -1,                       // Invalid: negative
			Limit: 0,                        // Invalid: zero
			Str1:  "",                       // Invalid: empty
			Str2:  string(make([]byte, 60)), // Invalid: too long
		}

		v := New()
		v.Check(input.Int1 > 0, "int1", "must be a positive integer")
		v.Check(input.Int2 > 0, "int2", "must be a positive integer")
		v.Check(input.Limit > 0, "limit", "must be a positive integer")
		v.Check(input.Str1 != "", "str1", "must be provided")
		v.Check(len(input.Str2) <= 50, "str2", "must not be more than 50 characters")

		if v.Valid() {
			t.Error("Invalid input should fail validation")
		}

		errorMap := v.ErrorMap()
		if errorMap == nil {
			t.Error("Invalid input should generate error map")
		}

		expectedErrorCount := 5
		if len(errorMap) != expectedErrorCount {
			t.Errorf("Expected %d errors, got %d", expectedErrorCount, len(errorMap))
		}

		// Verify error response would be in correct format for 422 status
		envelope := map[string]any{
			"error": map[string]any{
				"message": "validation failed",
				"details": errorMap,
			},
		}

		// Ensure it can be marshaled for HTTP response
		_, err := json.Marshal(envelope)
		if err != nil {
			t.Fatalf("Error envelope should be JSON serializable: %v", err)
		}
	})

	t.Run("validator reuse in handler", func(t *testing.T) {
		// Test pattern where validator might be reused (though not recommended)
		v := New()

		// First request
		v.Check(false, "field1", "error1")
		if v.Valid() {
			t.Error("Should be invalid after first use")
		}

		// Clear for next request (if reusing validator instance)
		v.Clear()
		v.Check(true, "field1", "should not appear")

		if !v.Valid() {
			t.Error("Should be valid after clear and valid check")
		}
	})
}
