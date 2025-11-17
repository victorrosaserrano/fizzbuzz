package data

import (
	"testing"
)

// TestFizzBuzzIntegration validates that our data models work correctly with the existing FizzBuzz algorithm
func TestFizzBuzzIntegration(t *testing.T) {
	// Test basic integration with data models
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Use the existing FizzBuzz function with our model data
	result := FizzBuzz(input.Int1, input.Int2, input.Limit, input.Str1, input.Str2)

	// Create output model
	output := FizzBuzzOutput{
		Result: result,
	}

	// Validate expected results
	expected := []string{
		"1", "2", "fizz", "4", "buzz", "fizz", "7", "8", "fizz", "buzz", "11", "fizz", "13", "14", "fizzbuzz",
	}

	if len(output.Result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(output.Result))
	}

	for i, v := range expected {
		if i < len(output.Result) && output.Result[i] != v {
			t.Errorf("At index %d: expected %q, got %q", i, v, output.Result[i])
		}
	}

	// Test that String() method works
	inputStr := input.String()
	if inputStr == "" {
		t.Error("String() method should not return empty string")
	}

	outputStr := output.String()
	if outputStr == "" {
		t.Error("String() method should not return empty string")
	}

	// Test stats key generation
	key := input.GenerateStatsKey()
	if key == "" {
		t.Error("GenerateStatsKey() should not return empty string")
	}

	// Test that same input generates same key
	key2 := input.GenerateStatsKey()
	if key != key2 {
		t.Error("Same input should generate same stats key")
	}

	// Test that different input generates different key
	input2 := input
	input2.Int1 = 7
	key3 := input2.GenerateStatsKey()
	if key == key3 {
		t.Error("Different input should generate different stats key")
	}
}
