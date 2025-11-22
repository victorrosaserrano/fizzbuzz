package data

import (
	"sync"
	"testing"
	"time"
)

// TestNewStatisticsTracker tests initialization of a new statistics tracker.
func TestNewStatisticsTracker(t *testing.T) {
	tracker := NewStatisticsTracker()

	// Verify proper initialization
	if tracker == nil {
		t.Fatal("Expected non-nil tracker")
	}

	// Verify empty state
	if count := tracker.EntryCount(); count != 0 {
		t.Errorf("Expected 0 entries, got %d", count)
	}

	// Verify nil most frequent for empty tracker
	if most := tracker.GetMostFrequent(); most != nil {
		t.Errorf("Expected nil most frequent for empty tracker, got %+v", most)
	}
}

// TestStatisticsTracker_Record tests basic recording functionality.
func TestStatisticsTracker_Record(t *testing.T) {
	tracker := NewStatisticsTracker()

	// Test data
	input1 := &FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	input2 := &FizzBuzzInput{
		Int1:  2,
		Int2:  7,
		Limit: 20,
		Str1:  "ping",
		Str2:  "pong",
	}

	// Record first input
	tracker.Record(input1)

	// Verify entry was created
	if count := tracker.EntryCount(); count != 1 {
		t.Errorf("Expected 1 entry after first record, got %d", count)
	}

	// Verify most frequent is set
	most := tracker.GetMostFrequent()
	if most == nil {
		t.Fatal("Expected most frequent to be set")
	}
	if most.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", most.Hits)
	}

	// Record same input again
	tracker.Record(input1)

	// Verify hit count increased
	if count := tracker.EntryCount(); count != 1 {
		t.Errorf("Expected 1 entry after duplicate record, got %d", count)
	}

	most = tracker.GetMostFrequent()
	if most.Hits != 2 {
		t.Errorf("Expected 2 hits after duplicate record, got %d", most.Hits)
	}

	// Record different input
	tracker.Record(input2)

	// Verify second entry was created
	if count := tracker.EntryCount(); count != 2 {
		t.Errorf("Expected 2 entries after different record, got %d", count)
	}

	// Verify most frequent is still the first input (2 hits vs 1 hit)
	most = tracker.GetMostFrequent()
	if most.Parameters != *input1 {
		t.Errorf("Expected most frequent to be input1, got %+v", most.Parameters)
	}
	if most.Hits != 2 {
		t.Errorf("Expected most frequent to have 2 hits, got %d", most.Hits)
	}
}

// TestStatisticsTracker_MostFrequentUpdates tests most frequent tracking.
func TestStatisticsTracker_MostFrequentUpdates(t *testing.T) {
	tracker := NewStatisticsTracker()

	input1 := &FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	input2 := &FizzBuzzInput{Int1: 2, Int2: 7, Limit: 20, Str1: "ping", Str2: "pong"}

	// Record input1 once
	tracker.Record(input1)
	most := tracker.GetMostFrequent()
	if most.Parameters != *input1 || most.Hits != 1 {
		t.Errorf("Expected input1 with 1 hit, got %+v with %d hits", most.Parameters, most.Hits)
	}

	// Record input2 twice to make it most frequent
	tracker.Record(input2)
	tracker.Record(input2)

	most = tracker.GetMostFrequent()
	if most.Parameters != *input2 || most.Hits != 2 {
		t.Errorf("Expected input2 with 2 hits, got %+v with %d hits", most.Parameters, most.Hits)
	}

	// Record input1 twice more to make it most frequent again
	tracker.Record(input1)
	tracker.Record(input1)

	most = tracker.GetMostFrequent()
	if most.Parameters != *input1 || most.Hits != 3 {
		t.Errorf("Expected input1 with 3 hits, got %+v with %d hits", most.Parameters, most.Hits)
	}
}

// TestStatisticsTracker_ThreadSafety tests concurrent access safety.
func TestStatisticsTracker_ThreadSafety(t *testing.T) {
	tracker := NewStatisticsTracker()

	// Test parameters
	const numGoroutines = 100
	const recordsPerGoroutine = 10

	input1 := &FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	input2 := &FizzBuzzInput{Int1: 2, Int2: 7, Limit: 20, Str1: "ping", Str2: "pong"}

	var wg sync.WaitGroup

	// Launch goroutines for concurrent recording
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < recordsPerGoroutine; j++ {
				// Alternate between two inputs
				if (id+j)%2 == 0 {
					tracker.Record(input1)
				} else {
					tracker.Record(input2)
				}

				// Also test concurrent reads
				tracker.GetMostFrequent()
				tracker.EntryCount()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify expected state
	expectedEntries := 2 // Two distinct inputs
	if count := tracker.EntryCount(); count != expectedEntries {
		t.Errorf("Expected %d entries after concurrent access, got %d", expectedEntries, count)
	}

	// Verify total hits are correct
	totalExpectedHits := numGoroutines * recordsPerGoroutine
	stats := tracker.GetStats()
	totalActualHits := 0
	for _, entry := range stats {
		totalActualHits += entry.Hits
	}

	if totalActualHits != totalExpectedHits {
		t.Errorf("Expected %d total hits, got %d", totalExpectedHits, totalActualHits)
	}

	// Verify most frequent is set
	most := tracker.GetMostFrequent()
	if most == nil {
		t.Error("Expected most frequent to be set after concurrent operations")
	}
}

// TestStatisticsTracker_ConcurrentReadsWrites tests mixed read/write operations.
func TestStatisticsTracker_ConcurrentReadsWrites(t *testing.T) {
	tracker := NewStatisticsTracker()

	input := &FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}

	const numReaders = 50
	const numWriters = 10
	const operationsPerRoutine = 100

	var wg sync.WaitGroup

	// Launch reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerRoutine; j++ {
				tracker.GetMostFrequent()
				tracker.EntryCount()
				time.Sleep(time.Microsecond) // Small delay to increase chance of concurrent access
			}
		}()
	}

	// Launch writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerRoutine; j++ {
				tracker.Record(input)
				time.Sleep(time.Microsecond) // Small delay to increase chance of concurrent access
			}
		}()
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify final state
	expectedHits := numWriters * operationsPerRoutine
	most := tracker.GetMostFrequent()
	if most == nil {
		t.Fatal("Expected most frequent to be set")
	}
	if most.Hits != expectedHits {
		t.Errorf("Expected %d hits, got %d", expectedHits, most.Hits)
	}
}

// TestStatisticsTracker_KeyConsistency tests parameter key generation consistency.
func TestStatisticsTracker_KeyConsistency(t *testing.T) {
	tracker := NewStatisticsTracker()

	// Same parameters in different objects should be treated as identical
	input1 := &FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	input2 := &FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}

	// Record both inputs
	tracker.Record(input1)
	tracker.Record(input2)

	// Should result in one entry with 2 hits
	if count := tracker.EntryCount(); count != 1 {
		t.Errorf("Expected 1 entry for identical parameters, got %d", count)
	}

	most := tracker.GetMostFrequent()
	if most.Hits != 2 {
		t.Errorf("Expected 2 hits for identical parameters, got %d", most.Hits)
	}
}

// TestStatisticsTracker_EdgeCases tests various edge cases.
func TestStatisticsTracker_EdgeCases(t *testing.T) {
	tracker := NewStatisticsTracker()

	testCases := []struct {
		name  string
		input FizzBuzzInput
	}{
		{
			name:  "empty strings",
			input: FizzBuzzInput{Int1: 1, Int2: 2, Limit: 5, Str1: "", Str2: ""},
		},
		{
			name:  "zero values",
			input: FizzBuzzInput{Int1: 1, Int2: 1, Limit: 1, Str1: "a", Str2: "b"},
		},
		{
			name:  "large integers within bounds",
			input: FizzBuzzInput{Int1: 9999, Int2: 10000, Limit: 100000, Str1: "large", Str2: "test"},
		},
		{
			name:  "special characters in strings",
			input: FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz!@#", Str2: "buzz$%^"},
		},
	}

	// Record all edge cases
	for _, tc := range testCases {
		tracker.Record(&tc.input)
	}

	// Verify all were recorded correctly
	expectedCount := len(testCases)
	if count := tracker.EntryCount(); count != expectedCount {
		t.Errorf("Expected %d entries for edge cases, got %d", expectedCount, count)
	}

	// Verify each case creates unique entries (no collisions)
	stats := tracker.GetStats()
	for _, tc := range testCases {
		found := false
		for _, entry := range stats {
			if entry.Parameters == tc.input {
				found = true
				if entry.Hits != 1 {
					t.Errorf("Expected 1 hit for %s, got %d", tc.name, entry.Hits)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected to find entry for %s", tc.name)
		}
	}
}

// TestStatisticsTracker_GetStats tests stats retrieval functionality.
func TestStatisticsTracker_GetStats(t *testing.T) {
	tracker := NewStatisticsTracker()

	inputs := []*FizzBuzzInput{
		{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"},
		{Int1: 2, Int2: 7, Limit: 20, Str1: "ping", Str2: "pong"},
	}

	// Record inputs with different frequencies
	tracker.Record(inputs[0])
	tracker.Record(inputs[0])
	tracker.Record(inputs[1])

	stats := tracker.GetStats()

	// Verify stats structure
	if len(stats) != 2 {
		t.Errorf("Expected 2 entries in stats, got %d", len(stats))
	}

	// Verify stats data integrity
	found0, found1 := false, false
	for _, entry := range stats {
		if entry.Parameters == *inputs[0] {
			found0 = true
			if entry.Hits != 2 {
				t.Errorf("Expected 2 hits for input[0], got %d", entry.Hits)
			}
		}
		if entry.Parameters == *inputs[1] {
			found1 = true
			if entry.Hits != 1 {
				t.Errorf("Expected 1 hit for input[1], got %d", entry.Hits)
			}
		}
	}

	if !found0 || !found1 {
		t.Error("Expected both inputs to be found in stats")
	}

	// Verify GetStats returns copies (modification doesn't affect tracker)
	for _, entry := range stats {
		entry.Hits = 999
	}

	// Verify original data is unchanged
	most := tracker.GetMostFrequent()
	if most.Hits == 999 {
		t.Error("GetStats should return copies, not references to internal data")
	}
}
