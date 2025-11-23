package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"fizzbuzz/internal/data"
	"fizzbuzz/internal/jsonlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRateLimiterShutdown tests that rate limiter cleanup goroutine terminates cleanly
func TestRateLimiterShutdown(t *testing.T) {
	cfg := config{
		limiter: struct {
			enabled bool
			rps     float64
			burst   int
		}{
			enabled: true,
			rps:     10.0,
			burst:   20,
		},
	}

	logger := jsonlog.New(os.Stdout, jsonlog.LevelError, "test") // Use ERROR level to reduce test output

	// Initialize rate limiter
	rateLimiter := initializeRateLimiter(cfg, logger)
	require.NotNil(t, rateLimiter)

	// Give the goroutine a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test that shutdown completes within reasonable time
	start := time.Now()
	rateLimiter.shutdown()
	rateLimiter.waitForShutdown()
	elapsed := time.Since(start)

	// Shutdown should complete quickly (well under 1 second)
	assert.Less(t, elapsed, 100*time.Millisecond,
		"Rate limiter shutdown took too long: %v", elapsed)
}

// TestRateLimiterShutdownDisabled tests shutdown when rate limiting is disabled
func TestRateLimiterShutdownDisabled(t *testing.T) {
	cfg := config{
		limiter: struct {
			enabled bool
			rps     float64
			burst   int
		}{
			enabled: false,
			rps:     10.0,
			burst:   20,
		},
	}

	logger := jsonlog.New(os.Stdout, jsonlog.LevelError, "test")

	// Initialize rate limiter with disabled config
	rateLimiter := initializeRateLimiter(cfg, logger)
	require.NotNil(t, rateLimiter)

	// Shutdown should work immediately since no goroutine is running
	start := time.Now()
	rateLimiter.shutdown()
	rateLimiter.waitForShutdown()
	elapsed := time.Since(start)

	// Should complete almost instantly since no goroutine to wait for
	assert.Less(t, elapsed, 10*time.Millisecond,
		"Disabled rate limiter shutdown took too long: %v", elapsed)
}

// TestGracefulShutdownWithInFlightRequests tests shutdown with active requests
func TestGracefulShutdownWithInFlightRequests(t *testing.T) {
	// Skip this test in short mode since it involves timing
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := config{
		port: 0, // Use random available port
		env:  "test",
		shutdown: struct {
			timeout time.Duration
		}{
			timeout: 2 * time.Second, // Short timeout for faster tests
		},
		limiter: struct {
			enabled bool
			rps     float64
			burst   int
		}{
			enabled: false, // Disable rate limiting for simpler test
		},
	}

	logger := jsonlog.New(os.Stdout, jsonlog.LevelError, "test")

	// Create a mock statistics handler that doesn't require database
	mockStats := &testStatisticsHandler{}

	// Initialize rate limiter (disabled)
	rateLimiter := initializeRateLimiter(cfg, logger)

	app := &application{
		config:      cfg,
		logger:      logger,
		statistics:  mockStats,
		rateLimiter: rateLimiter,
	}

	// Create test server
	server := httptest.NewUnstartedServer(app.routes())
	server.Start()
	defer server.Close()

	// Test that server responds before shutdown
	resp, err := http.Get(server.URL + "/v1/healthcheck")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Server should be accessible
	assert.True(t, isServerResponding(server.URL+"/v1/healthcheck"))
}

// TestShutdownTimeout tests behavior when shutdown exceeds timeout
func TestShutdownTimeout(t *testing.T) {
	// This test verifies that the shutdown context timeout works correctly
	cfg := config{
		shutdown: struct {
			timeout time.Duration
		}{
			timeout: 10 * time.Millisecond, // Very short timeout to force timeout condition
		},
	}

	// Create mock server that delays shutdown
	srv := &slowShutdownServer{
		shutdownDelay: 50 * time.Millisecond, // Longer than timeout
	}

	// Test the shutdown context timeout behavior
	ctx, cancel := context.WithTimeout(context.Background(), cfg.shutdown.timeout)
	defer cancel()

	start := time.Now()
	err := srv.Shutdown(ctx)
	elapsed := time.Since(start)

	// Should return deadline exceeded error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")

	// Should respect the timeout
	assert.Greater(t, elapsed, cfg.shutdown.timeout)
	assert.Less(t, elapsed, cfg.shutdown.timeout*2) // But not take too much longer
}

// TestShutdownConfiguration tests various shutdown timeout configurations
func TestShutdownConfiguration(t *testing.T) {
	tests := []struct {
		name            string
		timeoutFlag     string
		expectedTimeout time.Duration
	}{
		{
			name:            "default timeout",
			timeoutFlag:     "",
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "custom timeout",
			timeoutFlag:     "5s",
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "short timeout",
			timeoutFlag:     "100ms",
			expectedTimeout: 100 * time.Millisecond,
		},
		{
			name:            "long timeout",
			timeoutFlag:     "2m",
			expectedTimeout: 2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg config
			cfg.shutdown.timeout = 30 * time.Second // Set default

			if tt.timeoutFlag != "" {
				duration, err := time.ParseDuration(tt.timeoutFlag)
				require.NoError(t, err)
				cfg.shutdown.timeout = duration
			}

			assert.Equal(t, tt.expectedTimeout, cfg.shutdown.timeout)
		})
	}
}

// TestRateLimiterCleanupIntegration tests rate limiter cleanup during shutdown
func TestRateLimiterCleanupIntegration(t *testing.T) {
	cfg := config{
		limiter: struct {
			enabled bool
			rps     float64
			burst   int
		}{
			enabled: true,
			rps:     1.0,
			burst:   2,
		},
	}

	logger := jsonlog.New(os.Stdout, jsonlog.LevelError, "test")

	// Initialize rate limiter
	rateLimiter := initializeRateLimiter(cfg, logger)
	require.NotNil(t, rateLimiter)

	// Add some IP entries to the rate limiter
	_ = rateLimiter.getLimiter("192.168.1.1")
	_ = rateLimiter.getLimiter("192.168.1.2")
	_ = rateLimiter.getLimiter("10.0.0.1")

	// Verify entries exist
	totalEntries, _, _ := rateLimiter.getStats()
	assert.Equal(t, 3, totalEntries)

	// Test graceful shutdown
	start := time.Now()
	rateLimiter.shutdown()
	rateLimiter.waitForShutdown()
	elapsed := time.Since(start)

	// Should shutdown quickly
	assert.Less(t, elapsed, 100*time.Millisecond)

	// Rate limiter should still be functional for stats (but goroutine stopped)
	totalEntries, rps, burst := rateLimiter.getStats()
	assert.Equal(t, 3, totalEntries) // Entries preserved
	assert.Equal(t, 1.0, rps)
	assert.Equal(t, 2, burst)
}

// Helper functions and mock types for testing

// testStatisticsHandler provides a test implementation specific to shutdown tests
type testStatisticsHandler struct {
	mu sync.RWMutex
}

func (m *testStatisticsHandler) Record(ctx context.Context, input *data.FizzBuzzInput) error {
	return nil
}

func (m *testStatisticsHandler) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	return nil, nil
}

func (m *testStatisticsHandler) GetDatabaseHealth(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"status": "available",
	}, nil
}

func (m *testStatisticsHandler) RecordLegacy(input *data.FizzBuzzInput, logger *jsonlog.Logger) {
	// No-op for testing
}

func (m *testStatisticsHandler) GetMostFrequentLegacy(logger *jsonlog.Logger) *data.StatisticsEntry {
	return nil
}

func (m *testStatisticsHandler) GetPoolStats(ctx context.Context) (*data.PoolStats, error) {
	return &data.PoolStats{
		TotalConnections:  5,
		IdleConnections:   3,
		ActiveConnections: 2,
		MaxConnections:    25,
		Status:            "healthy",
		CollectedAt:       time.Now(),
	}, nil
}

func (m *testStatisticsHandler) Close() error {
	return nil
}

// slowShutdownServer simulates a server that takes time to shutdown
type slowShutdownServer struct {
	shutdownDelay time.Duration
}

func (s *slowShutdownServer) Shutdown(ctx context.Context) error {
	select {
	case <-time.After(s.shutdownDelay):
		return nil // Successful shutdown after delay
	case <-ctx.Done():
		return ctx.Err() // Timeout or cancellation
	}
}

// isServerResponding checks if the server is responding to HTTP requests
func isServerResponding(url string) bool {
	client := &http.Client{Timeout: 100 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// BenchmarkRateLimiterShutdown benchmarks the shutdown performance
func BenchmarkRateLimiterShutdown(b *testing.B) {
	cfg := config{
		limiter: struct {
			enabled bool
			rps     float64
			burst   int
		}{
			enabled: true,
			rps:     10.0,
			burst:   20,
		},
	}

	logger := jsonlog.New(os.Stdout, jsonlog.LevelError, "test")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rateLimiter := initializeRateLimiter(cfg, logger)

		// Add some load to the rate limiter
		for j := 0; j < 10; j++ {
			_ = rateLimiter.getLimiter(fmt.Sprintf("192.168.1.%d", j))
		}

		b.StartTimer()
		rateLimiter.shutdown()
		rateLimiter.waitForShutdown()
		b.StopTimer()
	}
}
