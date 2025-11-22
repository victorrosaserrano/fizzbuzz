package data

import (
	"context"
	"errors"
	"testing"
)

// Tests use existing MockStatisticsRepository from repository_test.go

func TestNewStatisticsService(t *testing.T) {
	mockRepo := NewMockStatisticsRepository()
	service := NewStatisticsService(mockRepo)

	if service == nil {
		t.Fatal("NewStatisticsService returned nil")
	}

	if service.repository != mockRepo {
		t.Error("StatisticsService repository not set correctly")
	}
}

func TestStatisticsService_Record(t *testing.T) {
	tests := []struct {
		name          string
		input         FizzBuzzInput
		mockBehavior  func(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error)
		expectedError bool
	}{
		{
			name: "successful record",
			input: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 15,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			mockBehavior: func(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error) {
				return &StatisticsEntry{
					Parameters: input,
					Hits:       1,
				}, nil
			},
			expectedError: false,
		},
		{
			name: "repository error",
			input: FizzBuzzInput{
				Int1:  3,
				Int2:  5,
				Limit: 15,
				Str1:  "fizz",
				Str2:  "buzz",
			},
			mockBehavior: func(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error) {
				return nil, errors.New("database connection failed")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := NewMockStatisticsRepository()
			mockRepo.recordFunc = tt.mockBehavior
			service := NewStatisticsService(mockRepo)

			err := service.Record(context.Background(), &tt.input)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestStatisticsService_GetMostFrequent(t *testing.T) {
	tests := []struct {
		name          string
		mockBehavior  func(ctx context.Context) (*StatisticsEntry, error)
		expectedEntry *StatisticsEntry
		expectedError bool
	}{
		{
			name: "successful get with result",
			mockBehavior: func(ctx context.Context) (*StatisticsEntry, error) {
				return &StatisticsEntry{
					Parameters: FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"},
					Hits:       42,
				}, nil
			},
			expectedEntry: &StatisticsEntry{
				Parameters: FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"},
				Hits:       42,
			},
			expectedError: false,
		},
		{
			name: "successful get with no results",
			mockBehavior: func(ctx context.Context) (*StatisticsEntry, error) {
				return nil, nil
			},
			expectedEntry: nil,
			expectedError: false,
		},
		{
			name: "repository error",
			mockBehavior: func(ctx context.Context) (*StatisticsEntry, error) {
				return nil, errors.New("database query failed")
			},
			expectedEntry: nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := NewMockStatisticsRepository()
			mockRepo.getMostFunc = tt.mockBehavior
			service := NewStatisticsService(mockRepo)

			entry, err := service.GetMostFrequent(context.Background())

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectedEntry != nil && entry != nil {
				if entry.Hits != tt.expectedEntry.Hits {
					t.Errorf("Expected hits %d, got %d", tt.expectedEntry.Hits, entry.Hits)
				}
				if entry.Parameters.Int1 != tt.expectedEntry.Parameters.Int1 {
					t.Errorf("Expected Int1 %d, got %d", tt.expectedEntry.Parameters.Int1, entry.Parameters.Int1)
				}
			} else if tt.expectedEntry != entry {
				t.Errorf("Expected entry %v, got %v", tt.expectedEntry, entry)
			}
		})
	}
}

func TestStatisticsService_LegacyCompatibility(t *testing.T) {
	t.Run("RecordLegacy handles errors silently", func(t *testing.T) {
		mockRepo := NewMockStatisticsRepository()
		mockRepo.recordFunc = func(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error) {
			return nil, errors.New("database error")
		}
		service := NewStatisticsService(mockRepo)

		input := &FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}

		// Should not panic or return error
		service.RecordLegacy(input)
	})

	t.Run("GetMostFrequentLegacy handles errors silently", func(t *testing.T) {
		mockRepo := NewMockStatisticsRepository()
		mockRepo.getMostFunc = func(ctx context.Context) (*StatisticsEntry, error) {
			return nil, errors.New("database error")
		}
		service := NewStatisticsService(mockRepo)

		result := service.GetMostFrequentLegacy()
		if result != nil {
			t.Error("Expected nil result when error occurs")
		}
	})

	t.Run("EntryCountLegacy returns 0 for now", func(t *testing.T) {
		mockRepo := NewMockStatisticsRepository()
		service := NewStatisticsService(mockRepo)

		count := service.EntryCountLegacy()
		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}
	})
}
