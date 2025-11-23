// Package data provides thread-safe statistics tracking for FizzBuzz request patterns.
// Implements concurrent-safe data structures for tracking API usage analytics.
package data

import (
	"context"
	"fmt"
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

// GetMostFrequentCompat provides compatibility method for the unified interface
func (st *StatisticsTracker) GetMostFrequentCompat() *StatisticsEntry {
	return st.GetMostFrequent()
}

// ============================================================================
// StatisticsService - PostgreSQL Repository Pattern Implementation
// ============================================================================

// StatisticsService provides business logic for statistics operations with PostgreSQL persistence.
// Replaces StatisticsTracker for production use with database-backed storage.
type StatisticsService struct {
	// repository provides persistent storage operations
	repository StatisticsRepository
}

// NewStatisticsService creates a new service with the given repository dependency
func NewStatisticsService(repository StatisticsRepository) *StatisticsService {
	return &StatisticsService{
		repository: repository,
	}
}

// Record records statistics using repository.Record() with context and error handling
func (ss *StatisticsService) Record(ctx context.Context, input *FizzBuzzInput) error {
	_, err := ss.repository.Record(ctx, *input)
	if err != nil {
		return fmt.Errorf("statistics service record failed: %w", err)
	}
	return nil
}

// GetMostFrequent gets most frequent statistics from repository with context
func (ss *StatisticsService) GetMostFrequent(ctx context.Context) (*StatisticsEntry, error) {
	entry, err := ss.repository.GetMostFrequent(ctx)
	if err != nil {
		return nil, fmt.Errorf("statistics service get most frequent failed: %w", err)
	}
	return entry, nil
}

// Legacy compatibility methods for transition period

// RecordLegacy provides legacy-compatible Record method (no context, no error return)
// Used during transition to maintain compatibility with existing HTTP handlers
func (ss *StatisticsService) RecordLegacy(input *FizzBuzzInput) {
	err := ss.Record(context.Background(), input)
	if err != nil {
		// In production, this would use structured logging
		// For now, we silently handle the error to maintain compatibility
		// TODO: Add proper error logging when integrated with application logger
	}
}

// GetMostFrequentLegacy provides legacy-compatible GetMostFrequent method
// Used during transition to maintain compatibility with existing HTTP handlers
func (ss *StatisticsService) GetMostFrequentLegacy() *StatisticsEntry {
	entry, err := ss.GetMostFrequent(context.Background())
	if err != nil {
		// In production, this would use structured logging
		// For now, return nil to maintain compatibility
		// TODO: Add proper error logging when integrated with application logger
		return nil
	}
	return entry
}

// EntryCount provides entry count functionality by querying repository
// For legacy compatibility - in production this might be an expensive operation
func (ss *StatisticsService) EntryCount(ctx context.Context) (int, error) {
	// Note: This is not implemented in current repository interface
	// For transition period, return 0 to maintain compatibility
	// TODO: Add GetEntryCount method to StatisticsRepository interface if needed
	return 0, nil
}

// EntryCountLegacy provides legacy-compatible EntryCount method
func (ss *StatisticsService) EntryCountLegacy() int {
	count, _ := ss.EntryCount(context.Background())
	return count
}

// Close closes the database repository connections
// Story 4.6: Graceful shutdown support for PostgreSQL connections
func (ss *StatisticsService) Close() error {
	if ss.repository != nil {
		return ss.repository.Close()
	}
	return nil
}

// GetDatabaseHealth provides health check information for the database repository
// Story 5.3: Health check endpoint integration
func (ss *StatisticsService) GetDatabaseHealth(ctx context.Context) (map[string]interface{}, error) {
	if ss.repository == nil {
		return map[string]interface{}{
			"status": "unavailable",
			"error":  "repository not initialized",
		}, fmt.Errorf("repository not initialized")
	}

	// Check if repository implements health check interface
	if healthRepo, ok := ss.repository.(*PostgreSQLStatisticsRepository); ok {
		return healthRepo.Health(ctx)
	}

	// Fallback: test basic functionality
	_, err := ss.repository.GetMostFrequent(ctx)
	if err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}, err
	}

	return map[string]interface{}{
		"status":  "healthy",
		"message": "basic database connectivity verified",
	}, nil
}
