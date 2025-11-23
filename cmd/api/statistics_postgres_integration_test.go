// Package main integration tests for Story 4.6: PostgreSQL Connection Pooling
// Tests database connection pooling and real PostgreSQL operations
package main

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"fizzbuzz/internal/data"
	"fizzbuzz/internal/jsonlog"
)

// TestPostgreSQLStatisticsInitialization tests the PostgreSQL statistics initialization
func TestPostgreSQLStatisticsInitialization(t *testing.T) {
	// Skip if no database environment configured
	if !isDatabaseTestingEnabled() {
		t.Skip("Database testing not enabled - set DB_TEST_ENABLED=true to run")
	}

	cfg := getTestConfig()
	logger := getTestLogger()

	handlerInterface, err := initializePostgreSQLStatistics(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to initialize PostgreSQL statistics: %v", err)
	}

	handler, ok := handlerInterface.(*statisticsHandler)
	if !ok {
		t.Fatal("Expected handler to be *statisticsHandler")
	}

	if handler.service == nil {
		t.Fatal("Expected service to be initialized but got nil")
	}

	// Test basic operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Test Record operation
	err = handler.Record(ctx, input)
	if err != nil {
		t.Errorf("Failed to record statistics: %v", err)
	}

	// Test GetMostFrequent operation
	entry, err := handler.GetMostFrequent(ctx)
	if err != nil {
		t.Errorf("Failed to get most frequent: %v", err)
	}

	if entry == nil {
		t.Error("Expected entry but got nil")
	}

	// Test service cleanup
	err = handler.Close()
	if err != nil {
		t.Errorf("Failed to close service: %v", err)
	}
}

// TestConnectionPoolingPerformance tests connection pool performance under load
func TestConnectionPoolingPerformance(t *testing.T) {
	if !isDatabaseTestingEnabled() {
		t.Skip("Database testing not enabled - set DB_TEST_ENABLED=true to run")
	}

	cfg := getTestConfig()
	logger := getTestLogger()

	handler, err := initializePostgreSQLStatistics(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to initialize PostgreSQL statistics: %v", err)
	}
	defer handler.Close()

	// Test concurrent operations to validate connection pooling
	const numOperations = 50
	resultChan := make(chan error, numOperations)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	// Launch concurrent operations
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			input := &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 15 + id, // Make each request unique
				Str1:  "fizz",
				Str2:  "buzz",
			}

			err := handler.Record(ctx, input)
			resultChan <- err
		}(i)
	}

	// Collect results
	var errorCount int
	for i := 0; i < numOperations; i++ {
		if err := <-resultChan; err != nil {
			t.Logf("Operation %d failed: %v", i, err)
			errorCount++
		}
	}

	duration := time.Since(start)
	t.Logf("Completed %d operations in %v (%.2f ops/sec)",
		numOperations, duration, float64(numOperations)/duration.Seconds())

	if errorCount > 0 {
		t.Errorf("Had %d errors out of %d operations", errorCount, numOperations)
	}

	// Verify reasonable performance (should complete well within timeout)
	if duration > 10*time.Second {
		t.Errorf("Operations took too long: %v", duration)
	}
}

// TestContextTimeoutIntegration tests real timeout behavior with database
func TestContextTimeoutIntegration(t *testing.T) {
	if !isDatabaseTestingEnabled() {
		t.Skip("Database testing not enabled - set DB_TEST_ENABLED=true to run")
	}

	cfg := getTestConfig()
	logger := getTestLogger()

	handler, err := initializePostgreSQLStatistics(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to initialize PostgreSQL statistics: %v", err)
	}
	defer handler.Close()

	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Test with very short timeout to ensure timeout handling works
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	err = handler.Record(ctx, input)
	if err == nil {
		t.Error("Expected timeout error but operation succeeded")
	}

	if err != context.DeadlineExceeded && err.Error() != "context deadline exceeded" {
		t.Errorf("Expected timeout error but got: %v", err)
	}
}

// Helper functions for testing

func isDatabaseTestingEnabled() bool {
	return os.Getenv("DB_TEST_ENABLED") == "true"
}

func getTestConfig() config {
	return config{
		db: struct {
			host         string
			port         int
			name         string
			user         string
			password     string
			sslMode      string
			maxConns     int
			maxIdleConns int
			maxLifetime  time.Duration
		}{
			host:         getEnvString("DB_HOST", "localhost"),
			port:         getEnvInt("DB_PORT", 5432),
			name:         getEnvString("DB_NAME", "fizzbuzz_test"),
			user:         getEnvString("DB_USER", "fizzbuzz_user"),
			password:     getEnvString("DB_PASSWORD", "fizzbuzz_pass"),
			sslMode:      getEnvString("DB_SSL_MODE", "disable"),
			maxConns:     getEnvInt("DB_MAX_CONNECTIONS", 10),
			maxIdleConns: 5,
			maxLifetime:  5 * time.Minute,
		},
	}
}

func getTestLogger() *jsonlog.Logger {
	return jsonlog.New(io.Discard, jsonlog.LevelDebug, "test")
}
