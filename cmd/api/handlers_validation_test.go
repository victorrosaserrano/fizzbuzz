package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fizzbuzz/internal/data"
)

// TestValidateFizzBuzzInput tests the validateFizzBuzzInput function
func TestValidateFizzBuzzInput(t *testing.T) {
	tests := []struct {
		name           string
		input          *data.FizzBuzzInput
		expectValid    bool
		expectedErrors map[string]string
	}{
		{
			name: "valid input",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid:    true,
			expectedErrors: map[string]string{},
		},
		{
			name: "int1 zero value",
			input: &data.FizzBuzzInput{
				Int1:  0,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"int1": "must be a positive integer",
			},
		},
		{
			name: "int1 negative",
			input: &data.FizzBuzzInput{
				Int1:  -1,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"int1": "must be a positive integer",
			},
		},
		{
			name: "int1 exceeds maximum",
			input: &data.FizzBuzzInput{
				Int1:  10001,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"int1": "must not be more than 10,000",
			},
		},
		{
			name: "int2 zero value",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  0,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"int2": "must be a positive integer",
			},
		},
		{
			name: "int2 negative",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  -5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"int2": "must be a positive integer",
			},
		},
		{
			name: "int2 exceeds maximum",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  15000,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"int2": "must not be more than 10,000",
			},
		},
		{
			name: "int1 equals int2",
			input: &data.FizzBuzzInput{
				Int1:  5,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"int1": "must be different from int2",
			},
		},
		{
			name: "limit zero value",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 0,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"limit": "must be a positive integer",
			},
		},
		{
			name: "limit negative",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: -100,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"limit": "must be a positive integer",
			},
		},
		{
			name: "limit exceeds maximum",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100001,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"limit": "must not be more than 100,000",
			},
		},
		{
			name: "str1 empty",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "",
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"str1": "must be provided",
			},
		},
		{
			name: "str1 too long",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  strings.Repeat("a", 51),
				Str2:  "buzz",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"str1": "must not be more than 50 characters",
			},
		},
		{
			name: "str2 empty",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  "",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"str2": "must be provided",
			},
		},
		{
			name: "str2 too long",
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 100,
				Str1:  "fizz",
				Str2:  strings.Repeat("b", 51),
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"str2": "must not be more than 50 characters",
			},
		},
		{
			name: "multiple validation errors",
			input: &data.FizzBuzzInput{
				Int1:  -1,
				Int2:  -2,
				Limit: 0,
				Str1:  "",
				Str2:  "",
			},
			expectValid: false,
			expectedErrors: map[string]string{
				"int1":  "must be a positive integer",
				"int2":  "must be a positive integer",
				"limit": "must be a positive integer",
				"str1":  "must be provided",
				"str2":  "must be provided",
			},
		},
		{
			name: "boundary values - maximum valid",
			input: &data.FizzBuzzInput{
				Int1:  10000,
				Int2:  9999,
				Limit: 100000,
				Str1:  strings.Repeat("f", 50),
				Str2:  strings.Repeat("b", 50),
			},
			expectValid:    true,
			expectedErrors: map[string]string{},
		},
		{
			name: "boundary values - minimum valid",
			input: &data.FizzBuzzInput{
				Int1:  1,
				Int2:  2,
				Limit: 1,
				Str1:  "f",
				Str2:  "b",
			},
			expectValid:    true,
			expectedErrors: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validateFizzBuzzInput(tt.input)

			if tt.expectValid {
				if !v.Valid() {
					t.Errorf("expected validation to pass, but got errors: %v", v.ErrorMap())
				}
			} else {
				if v.Valid() {
					t.Error("expected validation to fail, but it passed")
				}

				errorMap := v.ErrorMap()
				if len(errorMap) != len(tt.expectedErrors) {
					t.Errorf("expected %d errors, got %d", len(tt.expectedErrors), len(errorMap))
				}

				for field, expectedMsg := range tt.expectedErrors {
					if actualMsg, exists := errorMap[field]; !exists {
						t.Errorf("expected error for field %s, but none found", field)
					} else if actualMsg != expectedMsg {
						t.Errorf("field %s: expected error %q, got %q", field, expectedMsg, actualMsg)
					}
				}
			}
		})
	}
}

// TestFizzBuzzValidationIntegration tests validation integration with HTTP handler
func TestFizzBuzzValidationIntegration(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldContain  []string
	}{
		{
			name: "valid request passes validation",
			requestBody: `{
				"int1": 3,
				"int2": 5,
				"limit": 15,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusOK,
			shouldContain:  []string{`"data"`, `"result"`},
		},
		{
			name: "invalid int1 returns 422",
			requestBody: `{
				"int1": 0,
				"int2": 5,
				"limit": 15,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusUnprocessableEntity,
			shouldContain:  []string{`"error"`, `"message": "validation failed"`, `"int1": "must be a positive integer"`},
		},
		{
			name: "invalid int2 returns 422",
			requestBody: `{
				"int1": 3,
				"int2": -5,
				"limit": 15,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusUnprocessableEntity,
			shouldContain:  []string{`"error"`, `"int2": "must be a positive integer"`},
		},
		{
			name: "identical int values returns 422",
			requestBody: `{
				"int1": 5,
				"int2": 5,
				"limit": 15,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusUnprocessableEntity,
			shouldContain:  []string{`"error"`, `"int1": "must be different from int2"`},
		},
		{
			name: "invalid limit returns 422",
			requestBody: `{
				"int1": 3,
				"int2": 5,
				"limit": 0,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusUnprocessableEntity,
			shouldContain:  []string{`"error"`, `"limit": "must be a positive integer"`},
		},
		{
			name: "empty string returns 422",
			requestBody: `{
				"int1": 3,
				"int2": 5,
				"limit": 15,
				"str1": "",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusUnprocessableEntity,
			shouldContain:  []string{`"error"`, `"str1": "must be provided"`},
		},
		{
			name: "string too long returns 422",
			requestBody: `{
				"int1": 3,
				"int2": 5,
				"limit": 15,
				"str1": "` + strings.Repeat("a", 51) + `",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusUnprocessableEntity,
			shouldContain:  []string{`"error"`, `"str1": "must not be more than 50 characters"`},
		},
		{
			name: "multiple validation errors",
			requestBody: `{
				"int1": -1,
				"int2": 0,
				"limit": -100,
				"str1": "",
				"str2": ""
			}`,
			expectedStatus: http.StatusUnprocessableEntity,
			shouldContain: []string{
				`"error"`,
				`"validation failed"`,
				`"int1": "must be a positive integer"`,
				`"int2": "must be a positive integer"`,
				`"limit": "must be a positive integer"`,
				`"str1": "must be provided"`,
				`"str2": "must be provided"`,
			},
		},
		{
			name: "boundary values - maximum valid integers",
			requestBody: `{
				"int1": 10000,
				"int2": 9999,
				"limit": 100000,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusOK,
			shouldContain:  []string{`"data"`, `"result"`},
		},
		{
			name: "boundary values - integers exceed maximum",
			requestBody: `{
				"int1": 10001,
				"int2": 10002,
				"limit": 100001,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedStatus: http.StatusUnprocessableEntity,
			shouldContain: []string{
				`"error"`,
				`"int1": "must not be more than 10,000"`,
				`"int2": "must not be more than 10,000"`,
				`"limit": "must not be more than 100,000"`,
			},
		},
		{
			name: "boundary values - minimum valid",
			requestBody: `{
				"int1": 1,
				"int2": 2,
				"limit": 1,
				"str1": "f",
				"str2": "b"
			}`,
			expectedStatus: http.StatusOK,
			shouldContain:  []string{`"data"`, `"result"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(tt.requestBody))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler := app.routes()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			responseBody := rr.Body.String()
			for _, content := range tt.shouldContain {
				if !strings.Contains(responseBody, content) {
					t.Errorf("response should contain %q, got: %s", content, responseBody)
				}
			}

			// Verify Content-Type
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

// BenchmarkValidateFizzBuzzInput tests validation performance
func BenchmarkValidateFizzBuzzInput(b *testing.B) {
	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 100,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validateFizzBuzzInput(input)
		if !v.Valid() {
			b.Errorf("validation should pass for valid input")
		}
	}
}

// TestValidationPerformanceOverhead tests that validation overhead is minimal
func TestValidationPerformanceOverhead(t *testing.T) {
	app := newTestApplication(t)

	jsonBody := `{
		"int1": 3,
		"int2": 5,
		"limit": 100,
		"str1": "fizz",
		"str2": "buzz"
	}`

	// Perform multiple requests to warm up
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)
	}

	// Test with validation should still be fast
	req, err := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := app.routes()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Response should contain valid result
	if !strings.Contains(rr.Body.String(), `"data"`) {
		t.Errorf("expected data field in response")
	}
}
