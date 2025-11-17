package data

import (
	"encoding/json"
	"testing"
)

// TestFizzBuzzInputStringMethod tests the String() method for FizzBuzzInput
func TestFizzBuzzInputStringMethod(t *testing.T) {
	tests := []struct {
		name  string
		input FizzBuzzInput
		want  string
	}{
		{
			name: "basic case",
			input: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			want: `FizzBuzzInput{int1=3, int2=5, limit=100, str1="fizz", str2="buzz"}`,
		},
		{
			name: "empty strings",
			input: FizzBuzzInput{
				Int1:  2,
				Int2:  7,
				Limit: 50,
				Str1:  "",
				Str2:  "",
			},
			want: `FizzBuzzInput{int1=2, int2=7, limit=50, str1="", str2=""}`,
		},
		{
			name: "special characters",
			input: FizzBuzzInput{
				Int1:  4,
				Int2:  6,
				Limit: 20,
				Str1:  "foo-bar",
				Str2:  "baz_qux",
			},
			want: `FizzBuzzInput{int1=4, int2=6, limit=20, str1="foo-bar", str2="baz_qux"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.String()
			if got != tt.want {
				t.Errorf("FizzBuzzInput.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFizzBuzzOutputStringMethod tests the String() method for FizzBuzzOutput
func TestFizzBuzzOutputStringMethod(t *testing.T) {
	tests := []struct {
		name   string
		output FizzBuzzOutput
		want   string
	}{
		{
			name: "empty result",
			output: FizzBuzzOutput{
				Result: []string{},
			},
			want: "FizzBuzzOutput{result_length=0}",
		},
		{
			name: "small result",
			output: FizzBuzzOutput{
				Result: []string{"1", "2", "fizz"},
			},
			want: "FizzBuzzOutput{result_length=3}",
		},
		{
			name: "large result",
			output: FizzBuzzOutput{
				Result: make([]string, 1000),
			},
			want: "FizzBuzzOutput{result_length=1000}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.output.String()
			if got != tt.want {
				t.Errorf("FizzBuzzOutput.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFizzBuzzInputGenerateStatsKey tests the statistics key generation
func TestFizzBuzzInputGenerateStatsKey(t *testing.T) {
	tests := []struct {
		name   string
		input1 FizzBuzzInput
		input2 FizzBuzzInput
		same   bool // Should the keys be the same?
	}{
		{
			name: "identical inputs generate same key",
			input1: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			input2: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			same: true,
		},
		{
			name: "different int1 generates different key",
			input1: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			input2: FizzBuzzInput{
				Int1:  7,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			same: false,
		},
		{
			name: "different str1 generates different key",
			input1: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			input2: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "foo",
				Str2:  "buzz",
			},
			same: false,
		},
		{
			name: "different limit generates different key",
			input1: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			input2: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 200,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			same: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := tt.input1.GenerateStatsKey()
			key2 := tt.input2.GenerateStatsKey()

			// Validate key format (should be 64-character hex string for SHA256)
			if len(key1) != 64 {
				t.Errorf("Expected key length 64, got %d for key1", len(key1))
			}
			if len(key2) != 64 {
				t.Errorf("Expected key length 64, got %d for key2", len(key2))
			}

			// Check if keys match expectation
			if tt.same {
				if key1 != key2 {
					t.Errorf("Expected identical keys, got %v and %v", key1, key2)
				}
			} else {
				if key1 == key2 {
					t.Errorf("Expected different keys, but both are %v", key1)
				}
			}

			// Ensure keys are not empty
			if key1 == "" || key2 == "" {
				t.Error("Keys should not be empty")
			}
		})
	}
}

// TestJSONSerialization tests JSON marshaling and unmarshaling
func TestJSONSerialization(t *testing.T) {
	t.Run("FizzBuzzInput JSON serialization", func(t *testing.T) {
		input := FizzBuzzInput{
			Int1:  3,
			Int2:  5,
			Limit: 100,
			Str1:  "fizz",
			Str2:  "buzz",
		}

		// Marshal to JSON
		jsonBytes, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("Failed to marshal FizzBuzzInput: %v", err)
		}

		// Verify JSON contains expected fields
		expected := `{"int1":3,"int2":5,"limit":100,"str1":"fizz","str2":"buzz"}`
		got := string(jsonBytes)
		if got != expected {
			t.Errorf("JSON marshal = %v, want %v", got, expected)
		}

		// Unmarshal back
		var unmarshaled FizzBuzzInput
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal FizzBuzzInput: %v", err)
		}

		// Verify round-trip consistency
		if unmarshaled != input {
			t.Errorf("Round-trip failed: got %v, want %v", unmarshaled, input)
		}
	})

	t.Run("FizzBuzzOutput JSON serialization", func(t *testing.T) {
		output := FizzBuzzOutput{
			Result: []string{"1", "2", "fizz", "4", "buzz"},
		}

		// Marshal to JSON
		jsonBytes, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("Failed to marshal FizzBuzzOutput: %v", err)
		}

		// Verify JSON contains expected fields
		expected := `{"result":["1","2","fizz","4","buzz"]}`
		got := string(jsonBytes)
		if got != expected {
			t.Errorf("JSON marshal = %v, want %v", got, expected)
		}

		// Unmarshal back
		var unmarshaled FizzBuzzOutput
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal FizzBuzzOutput: %v", err)
		}

		// Verify round-trip consistency
		if len(unmarshaled.Result) != len(output.Result) {
			t.Errorf("Round-trip failed: length mismatch got %d, want %d", len(unmarshaled.Result), len(output.Result))
		}
		for i, v := range output.Result {
			if i < len(unmarshaled.Result) && unmarshaled.Result[i] != v {
				t.Errorf("Round-trip failed at index %d: got %v, want %v", i, unmarshaled.Result[i], v)
			}
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("empty strings in input", func(t *testing.T) {
		input := FizzBuzzInput{
			Int1:  1,
			Int2:  2,
			Limit: 5,
			Str1:  "",
			Str2:  "",
		}

		key := input.GenerateStatsKey()
		if key == "" {
			t.Error("GenerateStatsKey should work with empty strings")
		}

		str := input.String()
		if str == "" {
			t.Error("String should work with empty strings")
		}
	})

	t.Run("large numbers", func(t *testing.T) {
		input := FizzBuzzInput{
			Int1:  10000,
			Int2:  9999,
			Limit: 100000,
			Str1:  "very-long-string-that-approaches-fifty-chars",
			Str2:  "another-very-long-string-that-approaches-limit",
		}

		key := input.GenerateStatsKey()
		if key == "" {
			t.Error("GenerateStatsKey should work with large values")
		}

		str := input.String()
		if str == "" {
			t.Error("String should work with large values")
		}
	})

	t.Run("empty output", func(t *testing.T) {
		output := FizzBuzzOutput{
			Result: []string{},
		}

		str := output.String()
		if str != "FizzBuzzOutput{result_length=0}" {
			t.Errorf("String for empty output = %v, want FizzBuzzOutput{result_length=0}", str)
		}
	})
}
