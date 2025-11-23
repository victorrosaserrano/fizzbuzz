package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fizzbuzz/internal/data"
	"fizzbuzz/internal/jsonlog"
)

// TestCircuitBreakerFunctionality tests circuit breaker behavior during database outages
func TestCircuitBreakerFunctionality(t *testing.T) {
	// Create test logger
	logger := jsonlog.New(nil, jsonlog.LevelError, "test")

	// Create a mock failing repository
	failingRepo := &mockFailingRepository{}
	cbRepo := data.NewCircuitBreakerRepository(failingRepo, logger)

	ctx := context.Background()

	// Test 1: Circuit should be closed initially
	stats := cbRepo.GetCircuitBreakerStats()
	if stats.State.String() != "closed" {
		t.Errorf("Expected circuit to be closed initially, got %s", stats.State.String())
	}

	// Test 2: Force failures to open circuit (default threshold is 5)
	input := data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 100, Str1: "fizz", Str2: "buzz"}
	for i := 0; i < 6; i++ {
		_, err := cbRepo.Record(ctx, input)
		if err == nil {
			t.Errorf("Expected error from failing repository, got nil on attempt %d", i+1)
		}
	}

	// Test 3: Circuit should be open after failures
	stats = cbRepo.GetCircuitBreakerStats()
	if stats.State.String() != "open" {
		t.Errorf("Expected circuit to be open after failures, got %s", stats.State.String())
	}

	// Test 4: Verify fallback behavior when circuit is open
	entry, err := cbRepo.GetMostFrequent(ctx)
	// Should get cache result or empty result, but no error propagation from underlying repo
	if err != nil && err != data.ErrCircuitBreakerOpenWithFallback {
		t.Errorf("Expected circuit breaker fallback, got error: %v", err)
	}
	_ = entry // entry may be nil or cached value
}

// TestPoolStatsCollection tests detailed pool statistics collection
func TestPoolStatsCollection(t *testing.T) {

	// Create mock repository with pool stats
	mockRepo := &mockRepositoryWithPoolStats{}

	service := data.NewStatisticsService(mockRepo)
	statsHandler := &statisticsHandler{service: service}

	ctx := context.Background()

	// Test pool stats retrieval
	poolStats, err := statsHandler.GetPoolStats(ctx)
	if err != nil {
		t.Fatalf("Expected no error getting pool stats, got: %v", err)
	}

	if poolStats == nil {
		t.Fatal("Expected pool stats, got nil")
	}

	// Verify pool stats structure
	if poolStats.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", poolStats.Status)
	}

	if poolStats.MaxConnections != 25 {
		t.Errorf("Expected 25 max connections, got %d", poolStats.MaxConnections)
	}

	if poolStats.TotalConnections < 0 || poolStats.IdleConnections < 0 {
		t.Error("Connection counts should be non-negative")
	}
}

// TestHealthCheckWithDatabaseMonitoring tests health check integration with database metrics
func TestHealthCheckWithDatabaseMonitoring(t *testing.T) {
	// Create test application with mock statistics
	logger := jsonlog.New(nil, jsonlog.LevelError, "test")
	mockRepo := &mockRepositoryWithPoolStats{}
	service := data.NewStatisticsService(mockRepo)

	app := &application{
		config: config{
			env: "test",
		},
		logger:     logger,
		statistics: &statisticsHandler{service: service},
	}

	// Test health check endpoint
	req := httptest.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	rr := httptest.NewRecorder()

	app.healthcheckHandler(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Parse response JSON
	var response struct {
		Data data.HealthCheckResponse `json:"data"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify health response includes database metrics
	if response.Data.Database == nil {
		t.Error("Expected database health info in response")
	}

	if response.Data.Database.PoolStats == nil {
		t.Error("Expected pool stats in database health info")
	}

	// Verify pool stats in health response
	poolStats := response.Data.Database.PoolStats
	if poolStats.Status != "healthy" {
		t.Errorf("Expected healthy pool status, got %s", poolStats.Status)
	}
}

// TestDatabaseTimeoutBehavior tests database operation timeout handling
func TestDatabaseTimeoutBehavior(t *testing.T) {

	// Create a mock slow repository that always times out
	slowRepo := &mockSlowRepository{}

	service := data.NewStatisticsService(slowRepo)
	statsHandler := &statisticsHandler{service: service}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test that operations respect timeouts
	input := data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 100, Str1: "fizz", Str2: "buzz"}

	start := time.Now()
	err := statsHandler.Record(ctx, &input)
	duration := time.Since(start)

	// Should timeout quickly (within 200ms to account for overhead)
	if duration > 200*time.Millisecond {
		t.Errorf("Operation took too long: %v", duration)
	}

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Verify error indicates timeout
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected deadline exceeded error, got: %v", err)
	}
}

// TestConnectionLeakDetection tests that connection pools are properly managed
func TestConnectionLeakDetection(t *testing.T) {
	// This test would verify connection leak detection in a real environment
	// For unit testing, we verify the interface and configuration
	mockRepo := &mockRepositoryWithPoolStats{}

	service := data.NewStatisticsService(mockRepo)
	ctx := context.Background()

	// Simulate multiple operations
	input := data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 100, Str1: "fizz", Str2: "buzz"}

	for i := 0; i < 10; i++ {
		err := service.Record(ctx, &input)
		if err != nil {
			t.Errorf("Record operation %d failed: %v", i, err)
		}
	}

	// Verify pool stats show reasonable connection usage
	poolStats, err := service.GetPoolStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get pool stats: %v", err)
	}

	// Connections should not exceed reasonable limits
	if poolStats.TotalConnections > poolStats.MaxConnections {
		t.Errorf("Total connections (%d) exceeds max (%d)",
			poolStats.TotalConnections, poolStats.MaxConnections)
	}

	if poolStats.ActiveConnections > poolStats.TotalConnections {
		t.Errorf("Active connections (%d) exceeds total (%d)",
			poolStats.ActiveConnections, poolStats.TotalConnections)
	}
}

// Mock implementations for testing

type mockFailingRepository struct{}

func (m *mockFailingRepository) Record(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
	return nil, &mockDatabaseError{message: "simulated database connection failure"}
}

func (m *mockFailingRepository) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	return nil, &mockDatabaseError{message: "simulated database connection failure"}
}

func (m *mockFailingRepository) GetTopN(ctx context.Context, n int) ([]*data.StatisticsEntry, error) {
	return nil, &mockDatabaseError{message: "simulated database connection failure"}
}

func (m *mockFailingRepository) GetStats(ctx context.Context) (data.StatsSummary, error) {
	return data.StatsSummary{}, &mockDatabaseError{message: "simulated database connection failure"}
}

func (m *mockFailingRepository) GetPoolStats(ctx context.Context) (*data.PoolStats, error) {
	return nil, &mockDatabaseError{message: "simulated database connection failure"}
}

func (m *mockFailingRepository) Close() error {
	return nil
}

type mockRepositoryWithPoolStats struct{}

func (m *mockRepositoryWithPoolStats) Record(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
	return &data.StatisticsEntry{
		Parameters: input,
		Hits:       1,
	}, nil
}

func (m *mockRepositoryWithPoolStats) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	return &data.StatisticsEntry{
		Parameters: data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 100, Str1: "fizz", Str2: "buzz"},
		Hits:       42,
	}, nil
}

func (m *mockRepositoryWithPoolStats) GetTopN(ctx context.Context, n int) ([]*data.StatisticsEntry, error) {
	return []*data.StatisticsEntry{
		{
			Parameters: data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 100, Str1: "fizz", Str2: "buzz"},
			Hits:       42,
		},
	}, nil
}

func (m *mockRepositoryWithPoolStats) GetStats(ctx context.Context) (data.StatsSummary, error) {
	return data.StatsSummary{
		TotalUniqueRequests: 1,
		TotalRequests:       42,
		MaxHits:             42,
	}, nil
}

func (m *mockRepositoryWithPoolStats) GetPoolStats(ctx context.Context) (*data.PoolStats, error) {
	return &data.PoolStats{
		TotalConnections:         5,
		IdleConnections:          3,
		ActiveConnections:        2,
		ConstructingConnections:  0,
		MaxConnections:           25,
		AcquireCount:             100,
		AverageAcquireDurationMs: 2.5,
		Status:                   "healthy",
		CollectedAt:              time.Now(),
	}, nil
}

func (m *mockRepositoryWithPoolStats) Close() error {
	return nil
}

type mockSlowRepository struct{}

func (m *mockSlowRepository) Record(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
	// Simulate slow operation that will timeout
	select {
	case <-time.After(5 * time.Second):
		return &data.StatisticsEntry{Parameters: input, Hits: 1}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *mockSlowRepository) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	select {
	case <-time.After(5 * time.Second):
		return nil, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *mockSlowRepository) GetTopN(ctx context.Context, n int) ([]*data.StatisticsEntry, error) {
	select {
	case <-time.After(5 * time.Second):
		return []*data.StatisticsEntry{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *mockSlowRepository) GetStats(ctx context.Context) (data.StatsSummary, error) {
	select {
	case <-time.After(5 * time.Second):
		return data.StatsSummary{}, nil
	case <-ctx.Done():
		return data.StatsSummary{}, ctx.Err()
	}
}

func (m *mockSlowRepository) GetPoolStats(ctx context.Context) (*data.PoolStats, error) {
	select {
	case <-time.After(5 * time.Second):
		return &data.PoolStats{Status: "healthy"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *mockSlowRepository) Close() error {
	return nil
}

type mockDatabaseError struct {
	message string
}

func (e *mockDatabaseError) Error() string {
	return e.message
}
