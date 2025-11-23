// Package data provides database repository implementations for persistent statistics storage.
// Implements repository pattern for clean data access abstraction and testing flexibility.
package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"fizzbuzz/internal/jsonlog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StatisticsRepository defines the interface for statistics data access operations.
// Provides clean abstraction for statistics persistence with context-aware operations.
type StatisticsRepository interface {
	// Record adds or increments statistics for the given FizzBuzz input parameters.
	// Returns the updated StatisticsEntry with current hit count.
	Record(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error)

	// GetMostFrequent retrieves the parameter combination with the highest hit count.
	// Returns nil when no statistics exist (empty database).
	GetMostFrequent(ctx context.Context) (*StatisticsEntry, error)

	// GetTopN retrieves the N most frequently requested parameter combinations.
	// Returns slice ordered by hit count descending, then creation time ascending.
	GetTopN(ctx context.Context, n int) ([]*StatisticsEntry, error)

	// GetStats returns summary statistics for monitoring and analytics.
	// Provides aggregate data about total requests, unique parameters, etc.
	GetStats(ctx context.Context) (StatsSummary, error)

	// Close releases database connections and cleanup resources.
	// Should be called during application shutdown for graceful cleanup.
	Close() error

	// GetPoolStats returns detailed connection pool statistics for monitoring.
	// Provides real-time metrics for operational monitoring and alerting.
	GetPoolStats(ctx context.Context) (*PoolStats, error)
}

// StatsSummary provides aggregate statistics for monitoring and analytics.
// Used by health checks and operational dashboards.
type StatsSummary struct {
	// TotalUniqueRequests is the count of distinct parameter combinations
	TotalUniqueRequests int64 `json:"total_unique_requests"`
	// TotalRequests is the sum of all hit counts across all parameters
	TotalRequests int64 `json:"total_requests"`
	// AvgHitsPerUniqueRequest is the average hits per parameter combination
	AvgHitsPerUniqueRequest float64 `json:"avg_hits_per_unique_request"`
	// MaxHits is the highest hit count for any parameter combination
	MaxHits int64 `json:"max_hits"`
	// FirstRequestTime is the earliest created_at timestamp
	FirstRequestTime *time.Time `json:"first_request_time,omitempty"`
	// LastRequestTime is the latest updated_at timestamp
	LastRequestTime *time.Time `json:"last_request_time,omitempty"`
}

// PoolStats provides comprehensive connection pool statistics for monitoring.
// Used by health checks and operational dashboards for database monitoring.
type PoolStats struct {
	// TotalConnections is the current total number of connections in the pool
	TotalConnections int32 `json:"total_connections"`
	// IdleConnections is the number of idle connections available for use
	IdleConnections int32 `json:"idle_connections"`
	// ActiveConnections is the number of connections currently in use
	ActiveConnections int32 `json:"active_connections"`
	// ConstructingConnections is the number of connections being established
	ConstructingConnections int32 `json:"constructing_connections"`
	// MaxConnections is the maximum allowed connections in the pool
	MaxConnections int32 `json:"max_connections"`
	// AcquireCount is the total number of connection acquisitions
	AcquireCount int64 `json:"acquire_count"`
	// AverageAcquireDurationMs is the average time to acquire a connection in milliseconds
	AverageAcquireDurationMs float64 `json:"average_acquire_duration_ms"`
	// Status indicates the overall pool health (healthy, degraded, unavailable)
	Status string `json:"status"`
	// CollectedAt timestamp when these metrics were collected
	CollectedAt time.Time `json:"collected_at"`
}

// PostgreSQLStatisticsRepository implements StatisticsRepository using PostgreSQL.
// Uses pgx driver with connection pooling for optimal performance and resource management.
type PostgreSQLStatisticsRepository struct {
	// pool provides connection pooling for database operations
	pool *pgxpool.Pool
	// timeout configures default operation timeout for database queries
	timeout time.Duration
	// logger provides structured logging for database operations
	logger *jsonlog.Logger
}

// NewPostgreSQLStatisticsRepository creates a new PostgreSQL repository instance.
// Requires an active pgxpool connection pool and optional timeout configuration.
func NewPostgreSQLStatisticsRepository(pool *pgxpool.Pool, timeout time.Duration, logger *jsonlog.Logger) *PostgreSQLStatisticsRepository {
	if timeout <= 0 {
		timeout = 5 * time.Second // Default 5s timeout for operations
	}

	return &PostgreSQLStatisticsRepository{
		pool:    pool,
		timeout: timeout,
		logger:  logger,
	}
}

// Record implements StatisticsRepository.Record using atomic upsert operations.
// Uses the increment_statistics database function for thread-safe hit counting.
func (r *PostgreSQLStatisticsRepository) Record(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error) {
	start := time.Now()

	// Create context with timeout for operation
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Generate parameter hash for unique identification
	hash := input.GenerateStatsKey()

	// Log operation start
	if r.logger != nil {
		r.logger.DebugWithContext(ctx, "starting database record operation",
			"operation", "Record",
			"hash", hash[:12]+"...", // Only log first 12 chars of hash for brevity
			"timeout", r.timeout)
	}

	// Execute atomic upsert using database function
	var currentHits int64
	err := r.pool.QueryRow(ctx, `
		SELECT increment_statistics($1, $2, $3, $4, $5, $6)
	`, hash, input.Int1, input.Int2, input.Limit, input.Str1, input.Str2).Scan(&currentHits)

	duration := time.Since(start)

	// Log connection pool stats after operation
	if r.logger != nil {
		stat := r.pool.Stat()
		if err != nil {
			r.logger.WarnWithContext(ctx, "database record operation failed",
				"operation", "Record",
				"error", err,
				"duration_ms", duration.Milliseconds(),
				"pool_total_conns", stat.TotalConns(),
				"pool_idle_conns", stat.IdleConns(),
				"pool_active_conns", stat.AcquiredConns())
		} else {
			r.logger.DebugWithContext(ctx, "database record operation completed",
				"operation", "Record",
				"hits", currentHits,
				"duration_ms", duration.Milliseconds(),
				"pool_total_conns", stat.TotalConns(),
				"pool_idle_conns", stat.IdleConns(),
				"pool_active_conns", stat.AcquiredConns())
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to record statistics: %w", err)
	}

	// Create StatisticsEntry with updated hit count
	entry := &StatisticsEntry{
		ParametersHash: hash,
		Parameters:     input,
		Hits:           int(currentHits),
		UpdatedAt:      time.Now(),
	}

	return entry, nil
}

// GetMostFrequent implements StatisticsRepository.GetMostFrequent.
// Uses optimized database function for efficient query with proper indexing.
func (r *PostgreSQLStatisticsRepository) GetMostFrequent(ctx context.Context) (*StatisticsEntry, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Query most frequent using database function
	rows, err := r.pool.Query(ctx, "SELECT * FROM get_most_frequent_request()")
	if err != nil {
		return nil, fmt.Errorf("failed to query most frequent: %w", err)
	}
	defer rows.Close()

	// Check if any results exist
	if !rows.Next() {
		return nil, nil // No statistics exist yet
	}

	// Scan result into StatisticsEntry
	entry, err := r.scanStatisticsEntry(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan most frequent result: %w", err)
	}

	return entry, nil
}

// GetTopN implements StatisticsRepository.GetTopN.
// Returns the N most frequently requested parameter combinations.
func (r *PostgreSQLStatisticsRepository) GetTopN(ctx context.Context, n int) ([]*StatisticsEntry, error) {
	// Validate input parameter
	if n <= 0 {
		return []*StatisticsEntry{}, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Query top N using database function
	rows, err := r.pool.Query(ctx, "SELECT * FROM get_top_requests($1)", n)
	if err != nil {
		return nil, fmt.Errorf("failed to query top %d requests: %w", n, err)
	}
	defer rows.Close()

	// Collect results
	var entries []*StatisticsEntry
	for rows.Next() {
		entry, err := r.scanStatisticsEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top request result: %w", err)
		}
		entries = append(entries, entry)
	}

	// Check for row iteration errors
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return entries, nil
}

// GetStats implements StatisticsRepository.GetStats.
// Provides aggregate statistics for monitoring and operational dashboards.
func (r *PostgreSQLStatisticsRepository) GetStats(ctx context.Context) (StatsSummary, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var summary StatsSummary
	var firstRequest, lastRequest sql.NullTime

	// Query aggregate statistics from view
	err := r.pool.QueryRow(ctx, `
		SELECT 
			total_unique_requests,
			total_requests,
			avg_hits_per_unique_request,
			max_hits,
			first_request_time,
			last_request_time
		FROM v_statistics_summary
	`).Scan(
		&summary.TotalUniqueRequests,
		&summary.TotalRequests,
		&summary.AvgHitsPerUniqueRequest,
		&summary.MaxHits,
		&firstRequest,
		&lastRequest,
	)

	if err != nil {
		return StatsSummary{}, fmt.Errorf("failed to query statistics summary: %w", err)
	}

	// Handle nullable timestamps
	if firstRequest.Valid {
		summary.FirstRequestTime = &firstRequest.Time
	}
	if lastRequest.Valid {
		summary.LastRequestTime = &lastRequest.Time
	}

	return summary, nil
}

// Close implements StatisticsRepository.Close.
// Closes the connection pool and releases database resources.
func (r *PostgreSQLStatisticsRepository) Close() error {
	if r.pool != nil {
		r.pool.Close()
	}
	return nil
}

// Health checks the database connectivity and returns connection pool metrics.
// Used by health check endpoints for operational monitoring.
func (r *PostgreSQLStatisticsRepository) Health(ctx context.Context) (map[string]interface{}, error) {
	// Create context with shorter timeout for health checks
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Test basic connectivity with simple query
	var result int
	err := r.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return map[string]interface{}{
			"status":      "unhealthy",
			"error":       err.Error(),
			"connections": "unknown",
		}, err
	}

	// Get connection pool statistics
	stat := r.pool.Stat()

	return map[string]interface{}{
		"status":                   "healthy",
		"total_connections":        stat.TotalConns(),
		"idle_connections":         stat.IdleConns(),
		"acquired_connections":     stat.AcquiredConns(),
		"constructing_connections": stat.ConstructingConns(),
		"max_connections":          stat.MaxConns(),
		"acquire_count":            stat.AcquireCount(),
		"acquire_duration_ns":      stat.AcquireDuration().Nanoseconds(),
		"response_time_ms":         "< 2000", // Based on timeout
	}, nil
}

// GetPoolStats implements StatisticsRepository.GetPoolStats.
// Returns comprehensive connection pool statistics for monitoring and alerting.
func (r *PostgreSQLStatisticsRepository) GetPoolStats(ctx context.Context) (*PoolStats, error) {
	// Create context with timeout for pool stats collection
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// Get connection pool statistics from pgx
	stat := r.pool.Stat()

	// Calculate average acquire duration in milliseconds
	avgAcquireDurationMs := float64(stat.AcquireDuration().Nanoseconds()) / 1000000.0

	// Determine pool status based on usage patterns and thresholds
	status := r.calculatePoolStatus(stat)

	poolStats := &PoolStats{
		TotalConnections:         stat.TotalConns(),
		IdleConnections:          stat.IdleConns(),
		ActiveConnections:        stat.AcquiredConns(),
		ConstructingConnections:  stat.ConstructingConns(),
		MaxConnections:           stat.MaxConns(),
		AcquireCount:             stat.AcquireCount(),
		AverageAcquireDurationMs: avgAcquireDurationMs,
		Status:                   status,
		CollectedAt:              time.Now(),
	}

	return poolStats, nil
}

// calculatePoolStatus determines pool health status based on usage patterns.
// Returns "healthy", "degraded", or "critical" based on thresholds.
func (r *PostgreSQLStatisticsRepository) calculatePoolStatus(stat *pgxpool.Stat) string {
	maxConns := float64(stat.MaxConns())
	totalConns := float64(stat.TotalConns())
	idleConns := float64(stat.IdleConns())

	// Calculate usage percentage
	usagePercentage := totalConns / maxConns

	// Calculate idle percentage of total connections
	idlePercentage := idleConns / totalConns

	// Check for critical conditions
	if usagePercentage >= 0.95 { // 95% or more of max connections in use
		return "critical"
	}

	// Check for degraded conditions
	if usagePercentage >= 0.80 || idlePercentage < 0.1 { // 80% usage or less than 10% idle
		return "degraded"
	}

	// Check average acquire duration (if too high, might indicate contention)
	avgAcquireMs := float64(stat.AcquireDuration().Nanoseconds()) / 1000000.0
	if avgAcquireMs > 100 { // More than 100ms average acquire time
		return "degraded"
	}

	return "healthy"
}

// scanStatisticsEntry is a helper method to scan database rows into StatisticsEntry structs.
// Handles the mapping from database columns to Go struct fields.
func (r *PostgreSQLStatisticsRepository) scanStatisticsEntry(rows pgx.Rows) (*StatisticsEntry, error) {
	var entry StatisticsEntry
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&entry.ParametersHash,
		&entry.Parameters.Int1,
		&entry.Parameters.Int2,
		&entry.Parameters.Limit,
		&entry.Parameters.Str1,
		&entry.Parameters.Str2,
		&entry.Hits,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, err
	}

	entry.CreatedAt = createdAt
	entry.UpdatedAt = updatedAt

	return &entry, nil
}

// Compile-time verification that PostgreSQLStatisticsRepository implements StatisticsRepository
var _ StatisticsRepository = (*PostgreSQLStatisticsRepository)(nil)
