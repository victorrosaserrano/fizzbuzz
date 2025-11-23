// Package data provides core data structures and business logic for the FizzBuzz API.
// This includes FizzBuzz input/output models, algorithm implementations, and related utilities.
package data

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// FizzBuzzInput represents the input parameters for a FizzBuzz request.
// Contains the two divisor integers, the sequence limit, and the replacement strings.
type FizzBuzzInput struct {
	// Int1 is the first divisor integer (must be between 1 and 10,000)
	Int1 int `json:"int1"`
	// Int2 is the second divisor integer (must be between 1 and 10,000, different from Int1)
	Int2 int `json:"int2"`
	// Limit is the upper bound of the sequence (must be between 1 and 100,000)
	Limit int `json:"limit"`
	// Str1 is the replacement string for numbers divisible by Int1 (max 50 characters)
	Str1 string `json:"str1"`
	// Str2 is the replacement string for numbers divisible by Int2 (max 50 characters)
	Str2 string `json:"str2"`
}

// String returns a string representation of FizzBuzzInput for debugging and logging.
func (f FizzBuzzInput) String() string {
	return fmt.Sprintf("FizzBuzzInput{int1=%d, int2=%d, limit=%d, str1=%q, str2=%q}",
		f.Int1, f.Int2, f.Limit, f.Str1, f.Str2)
}

// GenerateStatsKey creates a unique key for statistics tracking based on the input parameters.
// Uses SHA256 hash of JSON representation to ensure collision-free unique keys.
func (f FizzBuzzInput) GenerateStatsKey() string {
	// Create consistent map representation for hashing
	data := map[string]interface{}{
		"int1":  f.Int1,
		"int2":  f.Int2,
		"limit": f.Limit,
		"str1":  f.Str1,
		"str2":  f.Str2,
	}

	// Marshal to JSON for consistent representation
	jsonBytes, _ := json.Marshal(data)

	// Generate SHA256 hash for collision-free uniqueness
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// FizzBuzzOutput represents the response from a FizzBuzz request.
// Contains the resulting sequence of strings after FizzBuzz transformation.
type FizzBuzzOutput struct {
	// Result is the array of strings containing the FizzBuzz transformation result
	Result []string `json:"result"`
}

// String returns a string representation of FizzBuzzOutput for debugging and logging.
func (f FizzBuzzOutput) String() string {
	return fmt.Sprintf("FizzBuzzOutput{result_length=%d}", len(f.Result))
}

// HealthCheckResponse represents the comprehensive health check response structure.
// Provides system status, version information, and optional database connectivity details.
type HealthCheckResponse struct {
	Status     string              `json:"status"`
	SystemInfo SystemInfo          `json:"system_info"`
	Database   *DatabaseHealthInfo `json:"database,omitempty"`
}

// SystemInfo contains basic application information for health checks.
type SystemInfo struct {
	Environment string `json:"environment"`
	Version     string `json:"version"`
	Timestamp   string `json:"timestamp"`
}

// DatabaseHealthInfo provides database connectivity and connection pool status.
type DatabaseHealthInfo struct {
	Status         string `json:"status"`
	ResponseTimeMs int64  `json:"response_time_ms"`
	ActiveConns    int32  `json:"active_connections"`
	IdleConns      int32  `json:"idle_connections"`
	MaxConns       int32  `json:"max_connections"`
}
