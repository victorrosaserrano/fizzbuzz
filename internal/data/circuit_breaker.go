// Package data provides circuit breaker implementation for database operations.
// Implements circuit breaker pattern for database call resilience and graceful degradation.
package data

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitBreakerState represents the current state of the circuit breaker
type CircuitBreakerState int

const (
	// CircuitClosed means the circuit is closed and calls are passing through
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen means the circuit is open and calls are being rejected
	CircuitOpen
	// CircuitHalfOpen means the circuit is allowing limited calls to test recovery
	CircuitHalfOpen
)

// String returns string representation of circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int
	// RecoveryTimeout is the time to wait before transitioning to half-open
	RecoveryTimeout time.Duration
	// SuccessThreshold is the number of successes needed in half-open to close
	SuccessThreshold int
	// Timeout is the maximum time to wait for an operation
	Timeout time.Duration
}

// DefaultCircuitBreakerConfig returns a sensible default configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,                // Open after 5 failures
		RecoveryTimeout:  30 * time.Second, // Wait 30s before trying again
		SuccessThreshold: 3,                // Need 3 successes to close
		Timeout:          5 * time.Second,  // 5s timeout for operations
	}
}

// CircuitBreaker implements the circuit breaker pattern for database operations
type CircuitBreaker struct {
	config        CircuitBreakerConfig
	state         CircuitBreakerState
	failures      int
	successes     int
	lastFailTime  time.Time
	mu            sync.RWMutex
	fallbackFunc  func(ctx context.Context) (interface{}, error)
	healthChecker func(ctx context.Context) error
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// SetFallbackFunc sets the fallback function to call when circuit is open
func (cb *CircuitBreaker) SetFallbackFunc(fallback func(ctx context.Context) (interface{}, error)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.fallbackFunc = fallback
}

// SetHealthChecker sets the health check function to test database connectivity
func (cb *CircuitBreaker) SetHealthChecker(healthCheck func(ctx context.Context) error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.healthChecker = healthCheck
}

// Call executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, cb.config.Timeout)
	defer cancel()

	state := cb.getState()

	switch state {
	case CircuitClosed:
		return cb.callClosed(ctx, fn)
	case CircuitOpen:
		return cb.callOpen(ctx)
	case CircuitHalfOpen:
		return cb.callHalfOpen(ctx, fn)
	default:
		return nil, errors.New("unknown circuit breaker state")
	}
}

// callClosed handles calls when circuit is closed
func (cb *CircuitBreaker) callClosed(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	result, err := fn(ctx)

	if err != nil {
		cb.recordFailure()
		return nil, err
	}

	cb.recordSuccess()
	return result, nil
}

// callOpen handles calls when circuit is open
func (cb *CircuitBreaker) callOpen(ctx context.Context) (interface{}, error) {
	// Check if enough time has passed to try recovery
	cb.mu.RLock()
	shouldTryRecovery := time.Since(cb.lastFailTime) >= cb.config.RecoveryTimeout
	cb.mu.RUnlock()

	if shouldTryRecovery {
		cb.transitionToHalfOpen()
		// Perform a health check to see if we should transition to half-open
		if cb.healthChecker != nil {
			if err := cb.healthChecker(ctx); err == nil {
				// Health check passed, transition to half-open
				return nil, ErrCircuitBreakerTransitioning
			}
		}
	}

	// Circuit is still open, try fallback if available
	cb.mu.RLock()
	fallback := cb.fallbackFunc
	cb.mu.RUnlock()

	if fallback != nil {
		result, err := fallback(ctx)
		if err != nil {
			return nil, ErrCircuitBreakerOpenWithFallbackError{Err: err}
		}
		return result, ErrCircuitBreakerOpenWithFallback
	}

	return nil, ErrCircuitBreakerOpen
}

// callHalfOpen handles calls when circuit is half-open
func (cb *CircuitBreaker) callHalfOpen(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	result, err := fn(ctx)

	if err != nil {
		cb.recordFailure()
		return nil, err
	}

	cb.recordSuccess()

	// Check if we have enough successes to close the circuit
	cb.mu.RLock()
	successes := cb.successes
	threshold := cb.config.SuccessThreshold
	cb.mu.RUnlock()

	if successes >= threshold {
		cb.transitionToClosed()
	}

	return result, nil
}

// recordFailure increments failure count and transitions to open if threshold reached
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.successes = 0 // Reset success counter
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.config.FailureThreshold {
		cb.state = CircuitOpen
	}
}

// recordSuccess increments success count and resets failure count
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes++
	cb.failures = 0 // Reset failure counter on success
}

// transitionToHalfOpen changes state to half-open
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitHalfOpen
	cb.successes = 0
}

// transitionToClosed changes state to closed
func (cb *CircuitBreaker) transitionToClosed() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
}

// getState returns the current circuit breaker state
func (cb *CircuitBreaker) getState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns current circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:        cb.state,
		Failures:     cb.failures,
		Successes:    cb.successes,
		LastFailTime: cb.lastFailTime,
	}
}

// CircuitBreakerStats provides statistics about the circuit breaker
type CircuitBreakerStats struct {
	State        CircuitBreakerState `json:"state"`
	Failures     int                 `json:"failures"`
	Successes    int                 `json:"successes"`
	LastFailTime time.Time           `json:"last_fail_time,omitempty"`
}

// Custom errors for circuit breaker
var (
	ErrCircuitBreakerOpen             = errors.New("circuit breaker is open")
	ErrCircuitBreakerOpenWithFallback = errors.New("circuit breaker is open, fallback used")
	ErrCircuitBreakerTransitioning    = errors.New("circuit breaker transitioning to half-open")
)

// ErrCircuitBreakerOpenWithFallbackError wraps fallback errors
type ErrCircuitBreakerOpenWithFallbackError struct {
	Err error
}

func (e ErrCircuitBreakerOpenWithFallbackError) Error() string {
	return "circuit breaker fallback error: " + e.Err.Error()
}

func (e ErrCircuitBreakerOpenWithFallbackError) Unwrap() error {
	return e.Err
}
