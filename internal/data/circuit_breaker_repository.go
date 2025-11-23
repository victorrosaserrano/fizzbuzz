// Package data provides circuit breaker wrapper for StatisticsRepository.
// Implements database resilience with circuit breaker pattern and graceful degradation.
package data

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"fizzbuzz/internal/jsonlog"
)

// CircuitBreakerRepository wraps StatisticsRepository with circuit breaker protection
type CircuitBreakerRepository struct {
	repository     StatisticsRepository
	circuitBreaker *CircuitBreaker
	cache          *cacheLayer
	logger         *jsonlog.Logger
	mu             sync.RWMutex
}

// cacheLayer provides fallback data when database is unavailable
type cacheLayer struct {
	mostFrequent      *StatisticsEntry
	lastRefresh       time.Time
	cacheTTL          time.Duration
	fallbackStats     StatsSummary
	fallbackPoolStats *PoolStats
	mu                sync.RWMutex
}

// NewCircuitBreakerRepository creates a new circuit breaker protected repository
func NewCircuitBreakerRepository(repository StatisticsRepository, logger *jsonlog.Logger) *CircuitBreakerRepository {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config)

	cache := &cacheLayer{
		cacheTTL: 5 * time.Minute, // Cache data for 5 minutes in degraded mode
		fallbackStats: StatsSummary{
			TotalUniqueRequests:     0,
			TotalRequests:           0,
			AvgHitsPerUniqueRequest: 0,
			MaxHits:                 0,
		},
		fallbackPoolStats: &PoolStats{
			Status:      "unavailable",
			CollectedAt: time.Now(),
		},
	}

	cbRepo := &CircuitBreakerRepository{
		repository:     repository,
		circuitBreaker: cb,
		cache:          cache,
		logger:         logger,
	}

	// Set health checker function
	cb.SetHealthChecker(func(ctx context.Context) error {
		if healthRepo, ok := repository.(*PostgreSQLStatisticsRepository); ok {
			_, err := healthRepo.Health(ctx)
			return err
		}
		_, err := repository.GetMostFrequent(ctx)
		return err
	})

	// Set fallback function for cache-only mode
	cb.SetFallbackFunc(func(ctx context.Context) (interface{}, error) {
		cbRepo.logger.Warn("using cache-only mode due to database unavailability")
		return cbRepo.cache.getMostFrequentCached(), nil
	})

	return cbRepo
}

// Record implements StatisticsRepository.Record with circuit breaker protection
func (cbr *CircuitBreakerRepository) Record(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error) {
	result, err := cbr.circuitBreaker.Call(ctx, func(ctx context.Context) (interface{}, error) {
		return cbr.repository.Record(ctx, input)
	})

	if err != nil {
		// Log circuit breaker events
		state := cbr.circuitBreaker.GetStats()
		cbr.logger.WarnWithContext(ctx, "database record operation failed",
			"error", err,
			"circuit_breaker_state", state.State.String(),
			"failures", state.Failures,
			"operation", "Record")

		// For Record operations, we can't provide meaningful fallback
		// Just return the error to let the caller handle it gracefully
		return nil, err
	}

	if entry, ok := result.(*StatisticsEntry); ok {
		// Update cache with successful result
		cbr.cache.updateMostFrequent(entry)
		return entry, nil
	}

	return nil, fmt.Errorf("unexpected result type from Record operation")
}

// GetMostFrequent implements StatisticsRepository.GetMostFrequent with circuit breaker protection
func (cbr *CircuitBreakerRepository) GetMostFrequent(ctx context.Context) (*StatisticsEntry, error) {
	result, err := cbr.circuitBreaker.Call(ctx, func(ctx context.Context) (interface{}, error) {
		return cbr.repository.GetMostFrequent(ctx)
	})

	if err != nil {
		// Log circuit breaker events
		state := cbr.circuitBreaker.GetStats()
		cbr.logger.WarnWithContext(ctx, "database read operation failed",
			"error", err,
			"circuit_breaker_state", state.State.String(),
			"failures", state.Failures,
			"operation", "GetMostFrequent")

		// Check if we used fallback (cache-only mode)
		if err == ErrCircuitBreakerOpenWithFallback {
			cbr.logger.InfoWithContext(ctx, "using cached statistics due to database unavailability")
			return cbr.cache.getMostFrequentCached(), nil
		}

		return nil, err
	}

	if entry, ok := result.(*StatisticsEntry); ok {
		// Update cache with successful result
		cbr.cache.updateMostFrequent(entry)
		return entry, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetMostFrequent operation")
}

// GetTopN implements StatisticsRepository.GetTopN with circuit breaker protection
func (cbr *CircuitBreakerRepository) GetTopN(ctx context.Context, n int) ([]*StatisticsEntry, error) {
	result, err := cbr.circuitBreaker.Call(ctx, func(ctx context.Context) (interface{}, error) {
		return cbr.repository.GetTopN(ctx, n)
	})

	if err != nil {
		state := cbr.circuitBreaker.GetStats()
		cbr.logger.WarnWithContext(ctx, "database GetTopN operation failed",
			"error", err,
			"circuit_breaker_state", state.State.String(),
			"n", n,
			"operation", "GetTopN")

		// For GetTopN, provide fallback with cached most frequent
		if err == ErrCircuitBreakerOpenWithFallback {
			cached := cbr.cache.getMostFrequentCached()
			if cached != nil {
				return []*StatisticsEntry{cached}, nil
			}
			return []*StatisticsEntry{}, nil
		}

		return nil, err
	}

	if entries, ok := result.([]*StatisticsEntry); ok {
		return entries, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetTopN operation")
}

// GetStats implements StatisticsRepository.GetStats with circuit breaker protection
func (cbr *CircuitBreakerRepository) GetStats(ctx context.Context) (StatsSummary, error) {
	result, err := cbr.circuitBreaker.Call(ctx, func(ctx context.Context) (interface{}, error) {
		return cbr.repository.GetStats(ctx)
	})

	if err != nil {
		state := cbr.circuitBreaker.GetStats()
		cbr.logger.WarnWithContext(ctx, "database GetStats operation failed",
			"error", err,
			"circuit_breaker_state", state.State.String(),
			"operation", "GetStats")

		// Return fallback stats when database unavailable
		if err == ErrCircuitBreakerOpenWithFallback {
			return cbr.cache.getFallbackStats(), nil
		}

		return StatsSummary{}, err
	}

	if stats, ok := result.(StatsSummary); ok {
		return stats, nil
	}

	return StatsSummary{}, fmt.Errorf("unexpected result type from GetStats operation")
}

// GetPoolStats implements StatisticsRepository.GetPoolStats with circuit breaker protection
func (cbr *CircuitBreakerRepository) GetPoolStats(ctx context.Context) (*PoolStats, error) {
	result, err := cbr.circuitBreaker.Call(ctx, func(ctx context.Context) (interface{}, error) {
		return cbr.repository.GetPoolStats(ctx)
	})

	if err != nil {
		state := cbr.circuitBreaker.GetStats()
		cbr.logger.WarnWithContext(ctx, "database GetPoolStats operation failed",
			"error", err,
			"circuit_breaker_state", state.State.String(),
			"operation", "GetPoolStats")

		// Return fallback pool stats when database unavailable
		if err == ErrCircuitBreakerOpenWithFallback {
			return cbr.cache.getFallbackPoolStats(), nil
		}

		return nil, err
	}

	if poolStats, ok := result.(*PoolStats); ok {
		return poolStats, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetPoolStats operation")
}

// Close implements StatisticsRepository.Close
func (cbr *CircuitBreakerRepository) Close() error {
	return cbr.repository.Close()
}

// GetCircuitBreakerStats returns current circuit breaker statistics for monitoring
func (cbr *CircuitBreakerRepository) GetCircuitBreakerStats() CircuitBreakerStats {
	return cbr.circuitBreaker.GetStats()
}

// cache layer methods

// updateMostFrequent updates the cached most frequent entry
func (c *cacheLayer) updateMostFrequent(entry *StatisticsEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.mostFrequent = entry
	c.lastRefresh = time.Now()
}

// getMostFrequentCached returns cached most frequent entry if still valid
func (c *cacheLayer) getMostFrequentCached() *StatisticsEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if cache is still valid
	if time.Since(c.lastRefresh) > c.cacheTTL {
		return nil // Cache expired
	}

	return c.mostFrequent
}

// getFallbackStats returns fallback statistics when database unavailable
func (c *cacheLayer) getFallbackStats() StatsSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.fallbackStats
}

// getFallbackPoolStats returns fallback pool stats when database unavailable
func (c *cacheLayer) getFallbackPoolStats() *PoolStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Update timestamp for current status
	fallback := *c.fallbackPoolStats
	fallback.CollectedAt = time.Now()

	return &fallback
}

// String implements fmt.Stringer for debugging
func (cbr *CircuitBreakerRepository) String() string {
	state := cbr.circuitBreaker.GetStats()
	stateJSON, _ := json.Marshal(state)
	return fmt.Sprintf("CircuitBreakerRepository{state=%s}", string(stateJSON))
}

// Compile-time verification that CircuitBreakerRepository implements StatisticsRepository
var _ StatisticsRepository = (*CircuitBreakerRepository)(nil)
