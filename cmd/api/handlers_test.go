package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFizzbuzzHandler(t *testing.T) {
	app := newTestApplication(t)

	t.Run("successful POST request with valid input", func(t *testing.T) {
		jsonBody := `{
			"int1": 3,
			"int2": 5,
			"limit": 15,
			"str1": "fizz",
			"str2": "buzz"
		}`

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

		expected := `{
	"data": {
		"result": [
			"1",
			"2",
			"fizz",
			"4",
			"buzz",
			"fizz",
			"7",
			"8",
			"fizz",
			"buzz",
			"11",
			"fizz",
			"13",
			"14",
			"fizzbuzz"
		]
	}
}`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expected) {
			t.Errorf("expected body %s, got %s", expected, rr.Body.String())
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("custom string replacement", func(t *testing.T) {
		jsonBody := `{
			"int1": 2,
			"int2": 3,
			"limit": 6,
			"str1": "foo",
			"str2": "bar"
		}`

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

		expected := `{
	"data": {
		"result": [
			"1",
			"foo",
			"bar",
			"foo",
			"5",
			"foobar"
		]
	}
}`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expected) {
			t.Errorf("expected body %s, got %s", expected, rr.Body.String())
		}
	})

	t.Run("edge case - limit 1", func(t *testing.T) {
		jsonBody := `{
			"int1": 3,
			"int2": 5,
			"limit": 1,
			"str1": "fizz",
			"str2": "buzz"
		}`

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

		expected := `{
	"data": {
		"result": [
			"1"
		]
	}
}`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expected) {
			t.Errorf("expected body %s, got %s", expected, rr.Body.String())
		}
	})

	t.Run("GET request returns 405 Method Not Allowed", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/v1/fizzbuzz", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}

		expectedError := `{
	"error": "the GET method is not supported for this resource"
}`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedError) {
			t.Errorf("expected body %s, got %s", expectedError, rr.Body.String())
		}
	})

	t.Run("PUT request returns 405 Method Not Allowed", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, "/v1/fizzbuzz", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}

		expectedError := `{
	"error": "the PUT method is not supported for this resource"
}`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedError) {
			t.Errorf("expected body %s, got %s", expectedError, rr.Body.String())
		}
	})

	t.Run("DELETE request returns 405 Method Not Allowed", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, "/v1/fizzbuzz", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}

		expectedError := `{
	"error": "the DELETE method is not supported for this resource"
}`
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedError) {
			t.Errorf("expected body %s, got %s", expectedError, rr.Body.String())
		}
	})

	t.Run("malformed JSON returns 400 Bad Request", func(t *testing.T) {
		malformedJSON := `{"int1": invalid, "int2": 5}`

		req, err := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(malformedJSON))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		// Should return error in envelope format
		if !strings.Contains(rr.Body.String(), `"error"`) {
			t.Errorf("expected error response with error field, got %s", rr.Body.String())
		}
	})

	t.Run("empty request body returns 400 Bad Request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), `"error"`) {
			t.Errorf("expected error response with error field, got %s", rr.Body.String())
		}
	})

	t.Run("invalid JSON data types", func(t *testing.T) {
		invalidJSON := `{
			"int1": "not-a-number",
			"int2": 5,
			"limit": 10,
			"str1": "fizz",
			"str2": "buzz"
		}`

		req, err := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(invalidJSON))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), `"error"`) {
			t.Errorf("expected error response with error field, got %s", rr.Body.String())
		}
	})

	t.Run("large limit value handles efficiently", func(t *testing.T) {
		jsonBody := `{
			"int1": 3,
			"int2": 5,
			"limit": 10000,
			"str1": "fizz",
			"str2": "buzz"
		}`

		req, err := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(jsonBody))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := app.routes()

		// Measure response time
		start := time.Now()
		handler.ServeHTTP(rr, req)
		duration := time.Since(start)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		// Performance requirement: should complete reasonably quickly
		if duration > 1*time.Second {
			t.Errorf("request took too long: %v", duration)
		}

		// Verify response structure is correct
		if !strings.Contains(rr.Body.String(), `"data"`) {
			t.Errorf("expected data field in response, got %s", rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), `"result"`) {
			t.Errorf("expected result field in response, got %s", rr.Body.String())
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkFizzbuzzHandler(b *testing.B) {
	app := newTestApplication(&testing.T{})

	jsonBody := `{
		"int1": 3,
		"int2": 5,
		"limit": 1000,
		"str1": "fizz",
		"str2": "buzz"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			b.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	}
}

func BenchmarkFizzbuzzHandlerLargeLimit(b *testing.B) {
	app := newTestApplication(&testing.T{})

	jsonBody := `{
		"int1": 3,
		"int2": 5,
		"limit": 100000,
		"str1": "fizz",
		"str2": "buzz"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/v1/fizzbuzz", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := app.routes()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			b.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	}
}
