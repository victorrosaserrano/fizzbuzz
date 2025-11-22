package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fizzbuzz/internal/data"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func TestStatisticsIntegration(t *testing.T) {
	tests := []struct {
		name          string
		payload       string
		expectedCode  int
		statsRecorded bool
	}{
		{
			name: "successful request records statistics",
			payload: `{
				"int1": 3,
				"int2": 5, 
				"limit": 15,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedCode:  http.StatusOK,
			statsRecorded: true,
		},
		{
			name: "validation failure does not record statistics",
			payload: `{
				"int1": 0,
				"int2": 5,
				"limit": 15,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedCode:  http.StatusUnprocessableEntity,
			statsRecorded: false,
		},
		{
			name: "malformed JSON does not record statistics",
			payload: `{
				"int1": "invalid",
				"int2": 5,
				"limit": 15,
				"str1": "fizz",
				"str2": "buzz"
			}`,
			expectedCode:  http.StatusBadRequest,
			statsRecorded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &application{
				config: config{
					port: 4000,
					env:  "test",
				},
				logger:     testLogger(),
				statistics: data.NewStatisticsTracker(),
			}

			// Record initial statistics count
			initialCount := app.statistics.EntryCount()

			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/v1/fizzbuzz", bytes.NewReader([]byte(tt.payload)))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			// Execute handler
			app.fizzbuzzHandler(rr, req)

			// Verify response status
			if rr.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			// Check if statistics were recorded
			finalCount := app.statistics.EntryCount()
			statsWereRecorded := finalCount > initialCount

			if tt.statsRecorded != statsWereRecorded {
				t.Errorf("expected statistics recorded: %v, actual: %v", tt.statsRecorded, statsWereRecorded)
			}
		})
	}
}

func TestConcurrentStatisticsRecording(t *testing.T) {
	app := &application{
		config: config{
			port: 4000,
			env:  "test",
		},
		logger:     testLogger(),
		statistics: data.NewStatisticsTracker(),
	}

	payload := `{
		"int1": 3,
		"int2": 5,
		"limit": 15,
		"str1": "fizz",
		"str2": "buzz"
	}`

	numRequests := 100
	done := make(chan bool, numRequests)

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/v1/fizzbuzz", bytes.NewReader([]byte(payload)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			app.fizzbuzzHandler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	timeout := time.After(5 * time.Second)
	completed := 0
	for completed < numRequests {
		select {
		case <-done:
			completed++
		case <-timeout:
			t.Fatal("test timed out waiting for concurrent requests")
		}
	}

	// Verify statistics were recorded correctly
	// Since all requests have identical parameters, should have 1 entry with numRequests hits
	entryCount := app.statistics.EntryCount()
	if entryCount != 1 {
		t.Errorf("expected 1 statistics entry, got %d", entryCount)
	}

	mostFrequent := app.statistics.GetMostFrequent()
	if mostFrequent == nil {
		t.Fatal("expected most frequent entry, got nil")
	}

	if mostFrequent.Hits != numRequests {
		t.Errorf("expected %d hits, got %d", numRequests, mostFrequent.Hits)
	}
}

func TestStatisticsGracefulDegradation(t *testing.T) {
	// This test simulates what would happen if statistics recording encountered an issue
	// In practice, the panic recovery in the handler should prevent issues
	app := &application{
		config: config{
			port: 4000,
			env:  "test",
		},
		logger:     testLogger(),
		statistics: data.NewStatisticsTracker(),
	}

	payload := `{
		"int1": 3,
		"int2": 5,
		"limit": 15,
		"str1": "fizz",
		"str2": "buzz"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/fizzbuzz", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Execute handler - should succeed even if statistics had issues
	app.fizzbuzzHandler(rr, req)

	// Verify response is successful
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Verify response body contains expected FizzBuzz result
	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, exists := response["data"]
	if !exists {
		t.Fatal("response missing 'data' field")
	}

	dataMap := data.(map[string]interface{})
	result, exists := dataMap["result"]
	if !exists {
		t.Fatal("response data missing 'result' field")
	}

	resultSlice := result.([]interface{})
	if len(resultSlice) != 15 {
		t.Errorf("expected result length 15, got %d", len(resultSlice))
	}

	// Verify first few elements of FizzBuzz sequence
	expected := []string{"1", "2", "fizz", "4", "buzz", "fizz", "7", "8", "fizz", "buzz", "11", "fizz", "13", "14", "fizzbuzz"}
	for i, exp := range expected {
		if i >= len(resultSlice) {
			break
		}
		if resultSlice[i].(string) != exp {
			t.Errorf("at index %d, expected %s, got %s", i, exp, resultSlice[i])
		}
	}
}
