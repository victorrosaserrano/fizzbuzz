package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// TestJSONHelpersValidation performs comprehensive validation of JSON helpers
// focusing on edge cases, security concerns, and production scenarios
func TestJSONHelpersValidation(t *testing.T) {
	app := newTestApplication(t)

	t.Run("writeJSON security and edge cases", func(t *testing.T) {
		t.Run("handles nil data gracefully", func(t *testing.T) {
			rr := httptest.NewRecorder()
			err := app.writeJSON(rr, http.StatusOK, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			expected := "null\n"
			if rr.Body.String() != expected {
				t.Errorf("expected %q, got %q", expected, rr.Body.String())
			}
		})

		t.Run("handles empty envelope", func(t *testing.T) {
			rr := httptest.NewRecorder()
			data := envelope{}
			err := app.writeJSON(rr, http.StatusOK, data, nil)
			if err != nil {
				t.Fatal(err)
			}

			expected := "{}\n"
			if rr.Body.String() != expected {
				t.Errorf("expected %q, got %q", expected, rr.Body.String())
			}
		})

		t.Run("properly escapes special characters", func(t *testing.T) {
			rr := httptest.NewRecorder()
			data := envelope{
				"xss":     "<script>alert('xss')</script>",
				"sql":     "'; DROP TABLE users; --",
				"unicode": "Hello 世界 🌍",
			}
			err := app.writeJSON(rr, http.StatusOK, data, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Verify JSON is properly escaped
			var result map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &result)
			if err != nil {
				t.Fatal(err)
			}

			if result["xss"] != "<script>alert('xss')</script>" {
				t.Error("XSS content should be properly preserved")
			}
			if result["unicode"] != "Hello 世界 🌍" {
				t.Error("Unicode should be properly handled")
			}
		})

		t.Run("handles large data structures", func(t *testing.T) {
			rr := httptest.NewRecorder()

			// Create a large data structure
			largeData := envelope{}
			for i := 0; i < 1000; i++ {
				key := string(rune('a'+i%26)) + strconv.Itoa(i)
				largeData[key] = strings.Repeat("test", 100)
			}

			err := app.writeJSON(rr, http.StatusOK, largeData, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Verify it's still valid JSON
			var result map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &result)
			if err != nil {
				t.Fatalf("large JSON should be valid: %v", err)
			}
		})

		t.Run("preserves custom headers", func(t *testing.T) {
			rr := httptest.NewRecorder()
			headers := http.Header{}
			headers.Set("Cache-Control", "no-cache")
			headers.Set("X-Custom", "test-value")
			headers.Add("X-Multi", "value1")
			headers.Add("X-Multi", "value2")

			data := envelope{"test": "value"}
			err := app.writeJSON(rr, http.StatusOK, data, headers)
			if err != nil {
				t.Fatal(err)
			}

			if rr.Header().Get("Cache-Control") != "no-cache" {
				t.Error("Cache-Control header not preserved")
			}
			if rr.Header().Get("X-Custom") != "test-value" {
				t.Error("X-Custom header not preserved")
			}

			multiValues := rr.Header().Values("X-Multi")
			if len(multiValues) != 2 {
				t.Errorf("expected 2 X-Multi values, got %d", len(multiValues))
			}
		})
	})

	t.Run("readJSON security and validation", func(t *testing.T) {
		t.Run("enforces size limits", func(t *testing.T) {
			// Create request larger than 1MB limit
			largeJSON := `{"data":"` + strings.Repeat("a", 1_048_577) + `"}`
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(largeJSON))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			var result map[string]interface{}
			err = app.readJSON(rr, req, &result)
			if err == nil {
				t.Error("expected error for oversized request body")
			}
		})

		t.Run("rejects unknown fields", func(t *testing.T) {
			type TestStruct struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}

			jsonWithExtra := `{"name":"test","age":25,"extra":"field"}`
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(jsonWithExtra))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			var result TestStruct
			err = app.readJSON(rr, req, &result)
			if err == nil {
				t.Error("expected error for unknown field")
			}
		})

		t.Run("handles malformed JSON variations", func(t *testing.T) {
			malformedCases := []string{
				`{"incomplete":`,
				`{"trailing":comma,}`,
				`{"wrong":"quotes'}`,
				`{broken json}`,
				`""`,  // Just a string, not object
				`123`, // Just a number, not object
			}

			for i, malformed := range malformedCases {
				req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(malformed))
				if err != nil {
					t.Fatal(err)
				}

				rr := httptest.NewRecorder()
				var result map[string]interface{}
				err = app.readJSON(rr, req, &result)
				if err == nil {
					t.Errorf("case %d: expected error for malformed JSON: %s", i, malformed)
				}
			}
		})

		t.Run("handles empty request body", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(""))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			var result map[string]interface{}
			err = app.readJSON(rr, req, &result)
			if err == nil {
				t.Error("expected error for empty request body")
			}
		})

		t.Run("validates single JSON value requirement", func(t *testing.T) {
			multiValueCases := []string{
				`{"first":1}{"second":2}`,
				`{"valid":true}extra_text`,
				`{"obj":1} [1,2,3]`,
				`null{"after":true}`,
			}

			for i, multiValue := range multiValueCases {
				req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(multiValue))
				if err != nil {
					t.Fatal(err)
				}

				rr := httptest.NewRecorder()
				var result map[string]interface{}
				err = app.readJSON(rr, req, &result)
				if err == nil {
					t.Errorf("case %d: expected error for multiple JSON values: %s", i, multiValue)
				}

				expectedErr := "body must only contain a single JSON value"
				if err != nil && err.Error() != expectedErr {
					t.Errorf("case %d: expected error %q, got %q", i, expectedErr, err.Error())
				}
			}
		})

		t.Run("handles different data types correctly", func(t *testing.T) {
			testCases := []struct {
				name     string
				json     string
				validate func(result map[string]interface{}) error
			}{
				{
					name: "string values",
					json: `{"str":"hello"}`,
					validate: func(result map[string]interface{}) error {
						if result["str"] != "hello" {
							return errors.New("string value mismatch")
						}
						return nil
					},
				},
				{
					name: "numeric values",
					json: `{"int":42,"float":3.14}`,
					validate: func(result map[string]interface{}) error {
						if result["int"] != float64(42) { // JSON numbers are float64
							return errors.New("int value mismatch")
						}
						if result["float"] != 3.14 {
							return errors.New("float value mismatch")
						}
						return nil
					},
				},
				{
					name: "boolean and null",
					json: `{"bool":true,"null_val":null}`,
					validate: func(result map[string]interface{}) error {
						if result["bool"] != true {
							return errors.New("boolean value mismatch")
						}
						if result["null_val"] != nil {
							return errors.New("null value should be nil")
						}
						return nil
					},
				},
				{
					name: "nested objects and arrays",
					json: `{"obj":{"nested":"value"},"arr":[1,2,3]}`,
					validate: func(result map[string]interface{}) error {
						obj, ok := result["obj"].(map[string]interface{})
						if !ok || obj["nested"] != "value" {
							return errors.New("nested object mismatch")
						}

						arr, ok := result["arr"].([]interface{})
						if !ok || len(arr) != 3 {
							return errors.New("array mismatch")
						}
						return nil
					},
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(tc.json))
					if err != nil {
						t.Fatal(err)
					}

					rr := httptest.NewRecorder()
					var result map[string]interface{}
					err = app.readJSON(rr, req, &result)
					if err != nil {
						t.Fatalf("expected no error, got %v", err)
					}

					if err := tc.validate(result); err != nil {
						t.Error(err)
					}
				})
			}
		})
	})

	t.Run("error handling integration", func(t *testing.T) {
		t.Run("errorJSON with complex error data", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			// Test complex error structure
			complexError := map[string]interface{}{
				"message": "Validation failed",
				"fields": map[string]string{
					"name":  "Name is required",
					"email": "Invalid email format",
				},
				"code": "VALIDATION_ERROR",
			}

			app.errorJSON(rr, req, http.StatusUnprocessableEntity, complexError)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
			}

			var response envelope
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			errorData, ok := response["error"].(map[string]interface{})
			if !ok {
				t.Fatal("error field should be an object")
			}

			if errorData["message"] != "Validation failed" {
				t.Error("error message mismatch")
			}
		})
	})
}

// TestJSONHelpersPerformance tests performance characteristics of JSON helpers
func TestJSONHelpersPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance tests in short mode")
	}

	app := newTestApplication(t)

	t.Run("writeJSON performance with large data", func(t *testing.T) {
		// Create reasonably large data structure
		data := envelope{}
		for i := 0; i < 100; i++ {
			key := "key" + strconv.Itoa(i)
			name := "Item " + strconv.Itoa(i)
			data[key] = map[string]interface{}{
				"id":          i,
				"name":        name,
				"description": strings.Repeat("Lorem ipsum ", 50),
				"metadata": map[string]interface{}{
					"created": "2025-01-01T00:00:00Z",
					"tags":    []string{"tag1", "tag2", "tag3"},
				},
			}
		}

		// Warm up
		for i := 0; i < 10; i++ {
			rr := httptest.NewRecorder()
			app.writeJSON(rr, http.StatusOK, data, nil)
		}

		// Performance measurement
		const iterations = 100
		for i := 0; i < iterations; i++ {
			rr := httptest.NewRecorder()
			err := app.writeJSON(rr, http.StatusOK, data, nil)
			if err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("readJSON performance with various sizes", func(t *testing.T) {
		sizes := []int{100, 1000, 10000}

		for _, size := range sizes {
			data := map[string]interface{}{}
			for i := 0; i < size; i++ {
				key := "key" + strconv.Itoa(i)
				value := "value" + strconv.Itoa(i)
				data[key] = value
			}

			jsonData, err := json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}

			// Measure readJSON performance
			const iterations = 50
			for i := 0; i < iterations; i++ {
				req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(jsonData))
				if err != nil {
					t.Fatal(err)
				}

				rr := httptest.NewRecorder()
				var result map[string]interface{}
				err = app.readJSON(rr, req, &result)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	})
}
