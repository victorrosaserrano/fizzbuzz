// Package main tests for Story 4.6 statisticsHandler implementation
// Tests the updated interface with context-aware operations
package main

import (
	"context"
	"fizzbuzz/internal/data"
	"testing"
	"time"
)

func TestStatisticsHandler_WithService(t *testing.T) {
	// Create mock repository
	mockRepo := &MockStatisticsRepository{
		recordFunc: func(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
			return &data.StatisticsEntry{
				Parameters: input,
				Hits:       1,
			}, nil
		},
		getMostFrequentFunc: func(ctx context.Context) (*data.StatisticsEntry, error) {
			return &data.StatisticsEntry{
				Parameters: data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"},
				Hits:       42,
			}, nil
		},
	}

	service := data.NewStatisticsService(mockRepo)
	handler := statisticsHandler{
		service: service,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test Record with context
	input := &data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	err := handler.Record(ctx, input)
	if err != nil {
		t.Errorf("Expected no error from Record, got: %v", err)
	}

	// Test GetMostFrequent with context
	result, err := handler.GetMostFrequent(ctx)
	if err != nil {
		t.Errorf("Expected no error from GetMostFrequent, got: %v", err)
	}
	if result == nil {
		t.Error("Expected result from service, got nil")
	}
	if result != nil && result.Hits != 42 {
		t.Errorf("Expected hits 42, got %d", result.Hits)
	}
}

func TestStatisticsHandler_LegacyMethods(t *testing.T) {
	// Test legacy compatibility methods
	mockRepo := &MockStatisticsRepository{
		recordFunc: func(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
			return &data.StatisticsEntry{
				Parameters: input,
				Hits:       1,
			}, nil
		},
		getMostFrequentFunc: func(ctx context.Context) (*data.StatisticsEntry, error) {
			return &data.StatisticsEntry{
				Parameters: data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"},
				Hits:       10,
			}, nil
		},
	}

	service := data.NewStatisticsService(mockRepo)
	handler := statisticsHandler{
		service: service,
	}

	// Test RecordLegacy (should not panic)
	input := &data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	handler.RecordLegacy(input, nil)

	// Test GetMostFrequentLegacy
	result := handler.GetMostFrequentLegacy(nil)
	if result == nil {
		t.Error("Expected result from legacy method, got nil")
	}
	if result != nil && result.Hits != 10 {
		t.Errorf("Expected hits 10, got %d", result.Hits)
	}
}

func TestStatisticsHandler_NilService(t *testing.T) {
	handler := statisticsHandler{
		service: nil, // Story 4.6: Only service is supported
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test Record with nil service - should return error
	input := &data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	err := handler.Record(ctx, input)
	if err == nil {
		t.Error("Expected error when service is nil, got nil")
	}

	// Test GetMostFrequent with nil service - should return error
	result, err := handler.GetMostFrequent(ctx)
	if err == nil {
		t.Error("Expected error when service is nil, got nil")
	}
	if result != nil {
		t.Error("Expected nil result when service is nil")
	}

	// Test legacy methods with nil service (should not panic)
	handler.RecordLegacy(input, nil) // Should not panic

	legacyResult := handler.GetMostFrequentLegacy(nil)
	if legacyResult != nil {
		t.Error("Expected nil result from legacy method when service is nil")
	}
}

func TestStatisticsHandler_ContextTimeout(t *testing.T) {
	// Create a slow mock repository
	mockRepo := &MockStatisticsRepository{
		recordFunc: func(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
			select {
			case <-time.After(100 * time.Millisecond):
				return &data.StatisticsEntry{Parameters: input, Hits: 1}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	service := data.NewStatisticsService(mockRepo)
	handler := statisticsHandler{
		service: service,
	}

	// Test with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	input := &data.FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	err := handler.Record(ctx, input)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

// Mock repository for testing (updated for Story 4.6)
type MockStatisticsRepository struct {
	recordFunc          func(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error)
	getMostFrequentFunc func(ctx context.Context) (*data.StatisticsEntry, error)
	closeFunc           func() error
}

func (m *MockStatisticsRepository) Record(ctx context.Context, input data.FizzBuzzInput) (*data.StatisticsEntry, error) {
	if m.recordFunc != nil {
		return m.recordFunc(ctx, input)
	}
	return &data.StatisticsEntry{Parameters: input, Hits: 1}, nil
}

func (m *MockStatisticsRepository) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	if m.getMostFrequentFunc != nil {
		return m.getMostFrequentFunc(ctx)
	}
	return nil, nil
}

func (m *MockStatisticsRepository) GetTopN(ctx context.Context, n int) ([]*data.StatisticsEntry, error) {
	return []*data.StatisticsEntry{}, nil
}

func (m *MockStatisticsRepository) GetStats(ctx context.Context) (data.StatsSummary, error) {
	return data.StatsSummary{}, nil
}

func (m *MockStatisticsRepository) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}
