// Package main tests for Story 4.6: Direct PostgreSQL Statistics Access
// Tests connection pooling, context-aware operations, and error handling
package main

import (
	"context"
	"testing"
	"time"

	"fizzbuzz/internal/data"
)

// TestStatisticsHandlerRecord tests the Record method with PostgreSQL service
func TestStatisticsHandlerRecord(t *testing.T) {
	tests := []struct {
		name        string
		handler     statisticsHandler
		input       *data.FizzBuzzInput
		expectError bool
	}{
		{
			name: "successful record with service",
			handler: statisticsHandler{
				service: data.NewStatisticsService(&mockRepository{}),
			},
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 15,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectError: false,
		},
		{
			name: "error with nil service",
			handler: statisticsHandler{
				service: nil,
			},
			input: &data.FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 15,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := tt.handler.Record(ctx, tt.input)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

// TestStatisticsHandlerGetMostFrequent tests the GetMostFrequent method
func TestStatisticsHandlerGetMostFrequent(t *testing.T) {
	tests := []struct {
		name        string
		handler     statisticsHandler
		expectError bool
		expectNil   bool
	}{
		{
			name: "successful get with service",
			handler: statisticsHandler{
				service: data.NewStatisticsService(&mockRepository{}),
			},
			expectError: false,
			expectNil:   false,
		},
		{
			name: "error with nil service",
			handler: statisticsHandler{
				service: nil,
			},
			expectError: true,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			entry, err := tt.handler.GetMostFrequent(ctx)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.expectNil && entry != nil {
				t.Errorf("expected nil entry but got: %v", entry)
			}
		})
	}
}

// TestStatisticsHandlerLegacyMethods tests legacy compatibility methods
func TestStatisticsHandlerLegacyMethods(t *testing.T) {
	handler := statisticsHandler{
		service: data.NewStatisticsService(&mockRepository{}),
	}

	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Test legacy Record method (should not panic)
	handler.RecordLegacy(input, nil)

	// Test legacy GetMostFrequent method
	entry := handler.GetMostFrequentLegacy(nil)
	if entry == nil {
		t.Error("expected entry from legacy method but got nil")
	}
}

// TestContextTimeoutHandling tests timeout behavior
func TestContextTimeoutHandling(t *testing.T) {
	handler := statisticsHandler{
		service: data.NewStatisticsService(&slowMockRepository{}),
	}

	input := &data.FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Test with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err := handler.Record(ctx, input)
	if err == nil {
		t.Error("expected timeout error but got nil")
	}
}

// mockRepository implements StatisticsRepository for testing
type mockRepository struct{}

func (m *mockRepository) Record(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
	return &data.StatisticsEntry{
		Parameters: input,
		Hits:       1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

func (m *mockRepository) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	return &data.StatisticsEntry{
		Parameters: data.FizzBuzzInput{
			Int1:  3,
			Int2:  5,
			Limit: 15,
			Str1:  "fizz",
			Str2:  "buzz",
		},
		Hits:      10,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockRepository) GetTopN(ctx context.Context, n int) ([]*data.StatisticsEntry, error) {
	return []*data.StatisticsEntry{}, nil
}

func (m *mockRepository) GetStats(ctx context.Context) (data.StatsSummary, error) {
	return data.StatsSummary{}, nil
}

func (m *mockRepository) GetPoolStats(ctx context.Context) (*data.PoolStats, error) {
	return &data.PoolStats{
		TotalConnections:  5,
		IdleConnections:   3,
		ActiveConnections: 2,
		MaxConnections:    25,
		Status:            "healthy",
		CollectedAt:       time.Now(),
	}, nil
}

func (m *mockRepository) Close() error {
	return nil
}

// slowMockRepository simulates slow database operations for timeout testing
type slowMockRepository struct{}

func (s *slowMockRepository) Record(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
	select {
	case <-time.After(1 * time.Second): // Simulate slow operation
		return &data.StatisticsEntry{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *slowMockRepository) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	select {
	case <-time.After(1 * time.Second): // Simulate slow operation
		return &data.StatisticsEntry{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *slowMockRepository) GetTopN(ctx context.Context, n int) ([]*data.StatisticsEntry, error) {
	return []*data.StatisticsEntry{}, nil
}

func (s *slowMockRepository) GetStats(ctx context.Context) (data.StatsSummary, error) {
	return data.StatsSummary{}, nil
}

func (s *slowMockRepository) GetPoolStats(ctx context.Context) (*data.PoolStats, error) {
	select {
	case <-time.After(1 * time.Second): // Simulate slow operation
		return &data.PoolStats{Status: "healthy"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *slowMockRepository) Close() error {
	return nil
}
