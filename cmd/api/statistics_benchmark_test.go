package main

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"fizzbuzz/internal/data"
)

func BenchmarkFizzBuzzWithStatistics(b *testing.B) {
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
		"limit": 100,
		"str1": "fizz",
		"str2": "buzz"
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/fizzbuzz", bytes.NewReader([]byte(payload)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		app.fizzbuzzHandler(rr, req)
	}
}

func BenchmarkFizzBuzzWithoutStatistics(b *testing.B) {
	// Create a version without statistics for comparison
	app := &application{
		config: config{
			port: 4000,
			env:  "test",
		},
		logger: testLogger(),
		// No statistics tracker
	}

	payload := `{
		"int1": 3,
		"int2": 5,
		"limit": 100,
		"str1": "fizz",
		"str2": "buzz"
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/fizzbuzz", bytes.NewReader([]byte(payload)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// We need a special handler without statistics for comparison
		app.fizzbuzzHandlerNoStats(rr, req)
	}
}
