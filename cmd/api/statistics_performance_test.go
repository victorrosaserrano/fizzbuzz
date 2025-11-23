// Package main performance tests for Story 4.6: PostgreSQL Statistics Performance
// Benchmarks connection pooling and database operation performance
package main

import (
	"context"
	"io"
	"testing"
	"time"

	"fizzbuzz/internal/data"
	"fizzbuzz/internal/jsonlog"
)

// BenchmarkStatisticsRecord benchmarks the Record operation performance
func BenchmarkStatisticsRecord(b *testing.B) {
	if !isDatabaseTestingEnabled() {
		b.Skip("Database testing not enabled - set DB_TEST_ENABLED=true to run")
	}

	cfg := getTestConfig()
	logger := jsonlog.New(io.Discard, jsonlog.LevelError, "test") // Minimize logging for benchmarks

	handler, err := initializePostgreSQLStatistics(cfg, logger)
	if err != nil {
		b.Fatalf("Failed to initialize PostgreSQL statistics: %v", err)
	}
	defer handler.Close()

	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := handler.Record(ctx, input)
			if err != nil {
				b.Errorf("Record failed: %v", err)
			}
		}
	})
}

// BenchmarkStatisticsGetMostFrequent benchmarks the GetMostFrequent operation
func BenchmarkStatisticsGetMostFrequent(b *testing.B) {
	if !isDatabaseTestingEnabled() {
		b.Skip("Database testing not enabled - set DB_TEST_ENABLED=true to run")
	}

	cfg := getTestConfig()
	logger := jsonlog.New(io.Discard, jsonlog.LevelError, "test") // Minimize logging for benchmarks

	handler, err := initializePostgreSQLStatistics(cfg, logger)
	if err != nil {
		b.Fatalf("Failed to initialize PostgreSQL statistics: %v", err)
	}
	defer handler.Close()

	// Populate some data first
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Insert some test data
	for i := 0; i < 5; i++ {
		err := handler.Record(ctx, input)
		if err != nil {
			b.Fatalf("Failed to populate test data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := handler.GetMostFrequent(ctx)
			if err != nil {
				b.Errorf("GetMostFrequent failed: %v", err)
			}
		}
	})
}

// BenchmarkConnectionPoolUtilization tests connection pool under different loads
func BenchmarkConnectionPoolUtilization(b *testing.B) {
	if !isDatabaseTestingEnabled() {
		b.Skip("Database testing not enabled - set DB_TEST_ENABLED=true to run")
	}

	tests := []struct {
		name     string
		maxConns int
	}{
		{"5-connections", 5},
		{"10-connections", 10},
		{"25-connections", 25},
		{"50-connections", 50},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			cfg := getTestConfig()
			cfg.db.maxConns = tt.maxConns

			logger := jsonlog.New(io.Discard, jsonlog.LevelError, "test")

			handler, err := initializePostgreSQLStatistics(cfg, logger)
			if err != nil {
				b.Fatalf("Failed to initialize PostgreSQL statistics: %v", err)
			}
			defer handler.Close()

			input := &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 15,
				Str1:  "fizz",
				Str2:  "buzz",
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := handler.Record(ctx, input)
					if err != nil {
						b.Errorf("Record failed: %v", err)
					}
				}
			})
		})
	}
}

// BenchmarkMixedOperations benchmarks a realistic mix of read/write operations
func BenchmarkMixedOperations(b *testing.B) {
	if !isDatabaseTestingEnabled() {
		b.Skip("Database testing not enabled - set DB_TEST_ENABLED=true to run")
	}

	cfg := getTestConfig()
	logger := jsonlog.New(io.Discard, jsonlog.LevelError, "test")

	handler, err := initializePostgreSQLStatistics(cfg, logger)
	if err != nil {
		b.Fatalf("Failed to initialize PostgreSQL statistics: %v", err)
	}
	defer handler.Close()

	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Populate some initial data
	for i := 0; i < 10; i++ {
		err := handler.Record(ctx, input)
		if err != nil {
			b.Fatalf("Failed to populate test data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			counter++

			// Simulate realistic workload: 80% writes, 20% reads
			if counter%5 == 0 {
				// Read operation
				_, err := handler.GetMostFrequent(ctx)
				if err != nil {
					b.Errorf("GetMostFrequent failed: %v", err)
				}
			} else {
				// Write operation
				err := handler.Record(ctx, input)
				if err != nil {
					b.Errorf("Record failed: %v", err)
				}
			}
		}
	})
}

// BenchmarkStatisticsWithTimeout benchmarks operations with realistic timeouts
func BenchmarkStatisticsWithTimeout(b *testing.B) {
	if !isDatabaseTestingEnabled() {
		b.Skip("Database testing not enabled - set DB_TEST_ENABLED=true to run")
	}

	cfg := getTestConfig()
	logger := jsonlog.New(io.Discard, jsonlog.LevelError, "test")

	handler, err := initializePostgreSQLStatistics(cfg, logger)
	if err != nil {
		b.Fatalf("Failed to initialize PostgreSQL statistics: %v", err)
	}
	defer handler.Close()

	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Use realistic timeout like HTTP handlers
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

			err := handler.Record(ctx, input)
			if err != nil {
				b.Errorf("Record failed: %v", err)
			}

			cancel()
		}
	})
}
