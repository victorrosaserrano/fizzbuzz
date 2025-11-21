package validator

import (
	"regexp"
	"testing"
)

// TestValidatorBasicFunctionality tests core Validator functionality
func TestValidatorBasicFunctionality(t *testing.T) {
	t.Run("new validator is valid", func(t *testing.T) {
		v := New()
		if !v.Valid() {
			t.Error("New validator should be valid")
		}

		errorMap := v.ErrorMap()
		if errorMap != nil {
			t.Errorf("New validator should have nil error map, got %v", errorMap)
		}
	})

	t.Run("adding error makes validator invalid", func(t *testing.T) {
		v := New()
		v.AddError("field1", "error message")

		if v.Valid() {
			t.Error("Validator should be invalid after adding error")
		}

		errorMap := v.ErrorMap()
		if errorMap == nil {
			t.Error("Error map should not be nil after adding error")
		}

		if len(errorMap) != 1 {
			t.Errorf("Expected 1 error, got %d", len(errorMap))
		}

		if errorMap["field1"] != "error message" {
			t.Errorf("Expected 'error message', got %v", errorMap["field1"])
		}
	})

	t.Run("check with false condition adds error", func(t *testing.T) {
		v := New()
		v.Check(false, "field1", "condition failed")

		if v.Valid() {
			t.Error("Validator should be invalid after failed check")
		}

		errorMap := v.ErrorMap()
		if errorMap["field1"] != "condition failed" {
			t.Errorf("Expected 'condition failed', got %v", errorMap["field1"])
		}
	})

	t.Run("check with true condition does not add error", func(t *testing.T) {
		v := New()
		v.Check(true, "field1", "should not appear")

		if !v.Valid() {
			t.Error("Validator should remain valid after successful check")
		}

		errorMap := v.ErrorMap()
		if errorMap != nil {
			t.Errorf("Error map should be nil, got %v", errorMap)
		}
	})
}

// TestValidatorErrorCollection tests multiple error accumulation
func TestValidatorErrorCollection(t *testing.T) {
	t.Run("multiple errors are accumulated", func(t *testing.T) {
		v := New()
		v.AddError("field1", "error1")
		v.AddError("field2", "error2")
		v.Check(false, "field3", "error3")

		if v.Valid() {
			t.Error("Validator should be invalid with multiple errors")
		}

		errorMap := v.ErrorMap()
		if len(errorMap) != 3 {
			t.Errorf("Expected 3 errors, got %d", len(errorMap))
		}

		expectedErrors := map[string]string{
			"field1": "error1",
			"field2": "error2",
			"field3": "error3",
		}

		for field, expectedMsg := range expectedErrors {
			if errorMap[field] != expectedMsg {
				t.Errorf("Field %s: expected %v, got %v", field, expectedMsg, errorMap[field])
			}
		}
	})

	t.Run("overwriting error replaces message", func(t *testing.T) {
		v := New()
		v.AddError("field1", "original error")
		v.AddError("field1", "updated error")

		errorMap := v.ErrorMap()
		if len(errorMap) != 1 {
			t.Errorf("Expected 1 error after overwrite, got %d", len(errorMap))
		}

		if errorMap["field1"] != "updated error" {
			t.Errorf("Expected 'updated error', got %v", errorMap["field1"])
		}
	})

	t.Run("error map is a copy", func(t *testing.T) {
		v := New()
		v.AddError("field1", "original")

		errorMap1 := v.ErrorMap()
		errorMap1["field1"] = "modified"
		errorMap1["field2"] = "added"

		errorMap2 := v.ErrorMap()
		if errorMap2["field1"] != "original" {
			t.Error("Original error should not be modified through returned map")
		}

		if _, exists := errorMap2["field2"]; exists {
			t.Error("New field should not exist in validator after external modification")
		}
	})
}

// TestValidatorStateManagement tests Clear and reuse functionality
func TestValidatorStateManagement(t *testing.T) {
	t.Run("clear resets validator state", func(t *testing.T) {
		v := New()
		v.AddError("field1", "error1")
		v.AddError("field2", "error2")

		if v.Valid() {
			t.Error("Validator should be invalid before clear")
		}

		v.Clear()

		if !v.Valid() {
			t.Error("Validator should be valid after clear")
		}

		errorMap := v.ErrorMap()
		if errorMap != nil {
			t.Errorf("Error map should be nil after clear, got %v", errorMap)
		}
	})

	t.Run("validator reuse after clear", func(t *testing.T) {
		v := New()

		// First use
		v.AddError("field1", "error1")
		if v.Valid() {
			t.Error("Should be invalid after first use")
		}

		// Clear and reuse
		v.Clear()
		v.AddError("field2", "error2")

		if v.Valid() {
			t.Error("Should be invalid after reuse")
		}

		errorMap := v.ErrorMap()
		if len(errorMap) != 1 {
			t.Errorf("Expected 1 error after reuse, got %d", len(errorMap))
		}

		if errorMap["field2"] != "error2" {
			t.Errorf("Expected 'error2', got %v", errorMap["field2"])
		}

		// Should not contain old errors
		if _, exists := errorMap["field1"]; exists {
			t.Error("Old errors should not exist after clear and reuse")
		}
	})
}

// TestPermittedValue tests the generic PermittedValue function
func TestPermittedValue(t *testing.T) {
	tests := []struct {
		name            string
		value           any
		permittedValues any
		expected        bool
	}{
		{
			name:            "string in permitted list",
			value:           "apple",
			permittedValues: []string{"apple", "banana", "cherry"},
			expected:        true,
		},
		{
			name:            "string not in permitted list",
			value:           "grape",
			permittedValues: []string{"apple", "banana", "cherry"},
			expected:        false,
		},
		{
			name:            "int in permitted list",
			value:           5,
			permittedValues: []int{1, 3, 5, 7, 9},
			expected:        true,
		},
		{
			name:            "int not in permitted list",
			value:           4,
			permittedValues: []int{1, 3, 5, 7, 9},
			expected:        false,
		},
		{
			name:            "empty permitted list",
			value:           "test",
			permittedValues: []string{},
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool

			switch v := tt.value.(type) {
			case string:
				permitted := tt.permittedValues.([]string)
				result = PermittedValue(v, permitted...)
			case int:
				permitted := tt.permittedValues.([]int)
				result = PermittedValue(v, permitted...)
			}

			if result != tt.expected {
				t.Errorf("PermittedValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMatches tests regex pattern matching
func TestMatches(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		pattern  string
		expected bool
	}{
		{
			name:     "simple match",
			value:    "hello",
			pattern:  "^hello$",
			expected: true,
		},
		{
			name:     "simple no match",
			value:    "hello",
			pattern:  "^world$",
			expected: false,
		},
		{
			name:     "email pattern match",
			value:    "test@example.com",
			pattern:  `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			expected: true,
		},
		{
			name:     "email pattern no match",
			value:    "invalid-email",
			pattern:  `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			expected: false,
		},
		{
			name:     "empty string with any pattern",
			value:    "",
			pattern:  ".*",
			expected: true,
		},
		{
			name:     "numeric pattern",
			value:    "12345",
			pattern:  `^\d+$`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := regexp.MustCompile(tt.pattern)
			result := Matches(tt.value, pattern)

			if result != tt.expected {
				t.Errorf("Matches(%q, %q) = %v, want %v", tt.value, tt.pattern, result, tt.expected)
			}
		})
	}

	t.Run("nil pattern returns false", func(t *testing.T) {
		result := Matches("test", nil)
		if result != false {
			t.Errorf("Matches with nil pattern should return false, got %v", result)
		}
	})
}

// TestUnique tests slice uniqueness validation
func TestUnique(t *testing.T) {
	tests := []struct {
		name     string
		values   any
		expected bool
	}{
		{
			name:     "unique strings",
			values:   []string{"apple", "banana", "cherry"},
			expected: true,
		},
		{
			name:     "duplicate strings",
			values:   []string{"apple", "banana", "apple"},
			expected: false,
		},
		{
			name:     "unique integers",
			values:   []int{1, 2, 3, 4, 5},
			expected: true,
		},
		{
			name:     "duplicate integers",
			values:   []int{1, 2, 3, 2, 5},
			expected: false,
		},
		{
			name:     "empty slice",
			values:   []string{},
			expected: true,
		},
		{
			name:     "single element",
			values:   []string{"single"},
			expected: true,
		},
		{
			name:     "all same elements",
			values:   []int{5, 5, 5, 5},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool

			switch v := tt.values.(type) {
			case []string:
				result = Unique(v)
			case []int:
				result = Unique(v)
			}

			if result != tt.expected {
				t.Errorf("Unique() = %v, want %v for %v", result, tt.expected, tt.values)
			}
		})
	}
}

// TestIn tests membership checking
func TestIn(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		list     any
		expected bool
	}{
		{
			name:     "string in list",
			value:    "banana",
			list:     []string{"apple", "banana", "cherry"},
			expected: true,
		},
		{
			name:     "string not in list",
			value:    "grape",
			list:     []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "int in list",
			value:    3,
			list:     []int{1, 2, 3, 4, 5},
			expected: true,
		},
		{
			name:     "int not in list",
			value:    10,
			list:     []int{1, 2, 3, 4, 5},
			expected: false,
		},
		{
			name:     "empty list",
			value:    "test",
			list:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool

			switch v := tt.value.(type) {
			case string:
				result = In(v, tt.list.([]string))
			case int:
				result = In(v, tt.list.([]int))
			}

			if result != tt.expected {
				t.Errorf("In() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestChainableValidation tests validation patterns for clean API usage
func TestChainableValidation(t *testing.T) {
	t.Run("realistic FizzBuzz validation scenario", func(t *testing.T) {
		v := New()

		// Simulate FizzBuzz parameter validation
		int1, int2, limit := 3, 5, 100
		str1, str2 := "fizz", "buzz"

		v.Check(int1 > 0, "int1", "must be a positive integer")
		v.Check(int1 <= 10000, "int1", "must not be more than 10,000")
		v.Check(int2 > 0, "int2", "must be a positive integer")
		v.Check(int2 <= 10000, "int2", "must not be more than 10,000")
		v.Check(int1 != int2, "int1", "must be different from int2")
		v.Check(limit > 0, "limit", "must be a positive integer")
		v.Check(limit <= 100000, "limit", "must not be more than 100,000")
		v.Check(str1 != "", "str1", "must be provided")
		v.Check(len(str1) <= 50, "str1", "must not be more than 50 characters")
		v.Check(str2 != "", "str2", "must be provided")
		v.Check(len(str2) <= 50, "str2", "must not be more than 50 characters")

		if !v.Valid() {
			t.Error("Validation should pass for valid FizzBuzz parameters")
		}
	})

	t.Run("validation failure accumulation", func(t *testing.T) {
		v := New()

		// Invalid parameters
		int1, int2, limit := -1, -1, 0
		str1, str2 := "", string(make([]byte, 60))

		v.Check(int1 > 0, "int1", "must be a positive integer")
		v.Check(int2 > 0, "int2", "must be a positive integer")
		v.Check(limit > 0, "limit", "must be a positive integer")
		v.Check(str1 != "", "str1", "must be provided")
		v.Check(len(str2) <= 50, "str2", "must not be more than 50 characters")

		if v.Valid() {
			t.Error("Validation should fail for invalid parameters")
		}

		errorMap := v.ErrorMap()
		expectedErrors := 5

		if len(errorMap) != expectedErrors {
			t.Errorf("Expected %d validation errors, got %d", expectedErrors, len(errorMap))
		}

		// Check specific error messages
		if errorMap["int1"] != "must be a positive integer" {
			t.Errorf("Unexpected error for int1: %v", errorMap["int1"])
		}

		if errorMap["str2"] != "must not be more than 50 characters" {
			t.Errorf("Unexpected error for str2: %v", errorMap["str2"])
		}
	})
}
