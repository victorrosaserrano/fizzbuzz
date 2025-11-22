// Package data provides database repository implementations for persistent statistics storage.
// Implements repository pattern for clean data access abstraction and testing flexibility.
package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

// PostgreSQLStatisticsRepository implements StatisticsRepository using PostgreSQL.
// Uses pgx driver with connection pooling for optimal performance and resource management.
type PostgreSQLStatisticsRepository struct {
	// pool provides connection pooling for database operations
	pool *pgxpool.Pool
	// timeout configures default operation timeout for database queries
	timeout time.Duration
}

// NewPostgreSQLStatisticsRepository creates a new PostgreSQL repository instance.
// Requires an active pgxpool connection pool and optional timeout configuration.
func NewPostgreSQLStatisticsRepository(pool *pgxpool.Pool, timeout time.Duration) *PostgreSQLStatisticsRepository {
	if timeout <= 0 {
		timeout = 5 * time.Second // Default 5s timeout for operations
	}

	return &PostgreSQLStatisticsRepository{
		pool:    pool,
		timeout: timeout,
	}
}

// Record implements StatisticsRepository.Record using atomic upsert operations.
// Uses the increment_statistics database function for thread-safe hit counting.
func (r *PostgreSQLStatisticsRepository) Record(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error) {
	// Create context with timeout for operation
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Generate parameter hash for unique identification
	hash := input.GenerateStatsKey()

	// Execute atomic upsert using database function
	var currentHits int64
	err := r.pool.QueryRow(ctx, `
		SELECT increment_statistics($1, $2, $3, $4, $5, $6)
	`, hash, input.Int1, input.Int2, input.Limit, input.Str1, input.Str2).Scan(&currentHits)

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
