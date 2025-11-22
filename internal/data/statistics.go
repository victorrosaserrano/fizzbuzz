// Package data provides thread-safe statistics tracking for FizzBuzz request patterns.
// Implements concurrent-safe data structures for tracking API usage analytics.
package data

import (
	"sync"
	"time"
)

// StatisticsEntry represents a tracked parameter combination with its hit frequency.
// Enhanced for database persistence with all necessary fields for data access and JSON marshaling.
type StatisticsEntry struct {
	// ID is the database primary key (not exposed in JSON responses)
	ID int64 `db:"id" json:"-"`
	// ParametersHash is the SHA256 hash of parameters for unique identification (not exposed)
	ParametersHash string `db:"parameters_hash" json:"-"`
	// Parameters contains the original FizzBuzz input parameters for this entry
	Parameters FizzBuzzInput `json:"parameters"`
	// Hits tracks the frequency count of requests for this parameter combination
	Hits int `db:"hits" json:"hits"`
	// CreatedAt tracks when this parameter combination was first seen
	CreatedAt time.Time `db:"created_at" json:"created_at,omitempty"`
	// UpdatedAt tracks when this parameter combination was last requested
	UpdatedAt time.Time `db:"updated_at" json:"updated_at,omitempty"`
}

// StatisticsTracker provides thread-safe tracking of FizzBuzz request parameters.
// Uses RWMutex for concurrent read/write access optimization and efficient storage.
type StatisticsTracker struct {
	// mu provides thread-safe access to the entries map
	mu sync.RWMutex
	// entries stores parameter combinations mapped by their unique hash keys
	entries map[string]*StatisticsEntry
	// mostFrequent tracks the current most frequent entry for O(1) access
	mostFrequent *StatisticsEntry
}

// NewStatisticsTracker creates a new thread-safe statistics tracker.
// Returns an initialized tracker ready for concurrent use.
func NewStatisticsTracker() *StatisticsTracker {
	return &StatisticsTracker{
		entries: make(map[string]*StatisticsEntry),
	}
}

// Record adds or updates statistics for the given FizzBuzz input parameters.
// Thread-safe method that handles concurrent writes using write locks.
// Creates new entries for first-time parameters or increments existing hit counts.
func (st *StatisticsTracker) Record(input *FizzBuzzInput) {
	// Generate unique key for parameter combination
	key := input.GenerateStatsKey()

	// Acquire write lock for concurrent safety
	st.mu.Lock()
	defer st.mu.Unlock()

	// Check if entry already exists
	if entry, exists := st.entries[key]; exists {
		// Increment hit count for existing entry
		entry.Hits++

		// Update most frequent if this entry now has the highest count
		if st.mostFrequent == nil || entry.Hits > st.mostFrequent.Hits {
			st.mostFrequent = entry
		}
	} else {
		// Create new entry for first-time parameter combination
		newEntry := &StatisticsEntry{
			Parameters: *input,
			Hits:       1,
		}
		st.entries[key] = newEntry

		// Update most frequent if this is the first entry or has higher count
		if st.mostFrequent == nil || newEntry.Hits > st.mostFrequent.Hits {
			st.mostFrequent = newEntry
		}
	}
}

// GetMostFrequent returns the parameter combination with the highest hit count.
// Thread-safe method using read locks to allow concurrent reads without blocking.
// Returns nil when no statistics exist (empty tracker).
// In case of ties, returns any of the most frequent entries consistently.
func (st *StatisticsTracker) GetMostFrequent() *StatisticsEntry {
	// Acquire read lock for concurrent safety
	st.mu.RLock()
	defer st.mu.RUnlock()

	// Handle edge case: empty tracker
	if st.mostFrequent == nil {
		return nil
	}

	// Validate mostFrequent is still accurate (edge case for concurrent modifications)
	// This ensures consistency in rare edge cases where mostFrequent might be stale
	maxHits := 0
	var result *StatisticsEntry

	for _, entry := range st.entries {
		if entry.Hits > maxHits {
			maxHits = entry.Hits
			result = entry
		}
	}

	// Update cached most frequent if validation found a different result
	// This handles rare edge cases but maintains O(1) performance in normal cases
	if result != nil && (st.mostFrequent == nil || result.Hits > st.mostFrequent.Hits) {
		// Note: We can't update mostFrequent here due to read lock
		// This is acceptable as the next Record() call will fix it
		return result
	}

	return st.mostFrequent
}

// GetStats returns a copy of all statistics entries for debugging/inspection.
// Thread-safe method that returns current statistics without exposing internal state.
func (st *StatisticsTracker) GetStats() map[string]*StatisticsEntry {
	st.mu.RLock()
	defer st.mu.RUnlock()

	// Create a copy to avoid exposing internal map
	result := make(map[string]*StatisticsEntry, len(st.entries))
	for key, entry := range st.entries {
		// Create copy of entry to prevent external modification
		entryCopy := &StatisticsEntry{
			Parameters: entry.Parameters,
			Hits:       entry.Hits,
		}
		result[key] = entryCopy
	}

	return result
}

// EntryCount returns the total number of unique parameter combinations tracked.
// Thread-safe method for monitoring and debugging purposes.
func (st *StatisticsTracker) EntryCount() int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return len(st.entries)
}
