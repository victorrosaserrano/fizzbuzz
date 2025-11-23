package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// ipLimiter holds rate limiter and last seen time for an IP address
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimiterMap holds per-IP rate limiters with thread-safe access
type rateLimiterMap struct {
	mu       sync.RWMutex
	limiters map[string]*ipLimiter
	rps      rate.Limit
	burst    int
}

// newRateLimiterMap creates a new rate limiter map with the specified rate and burst
func newRateLimiterMap(rps float64, burst int) *rateLimiterMap {
	return &rateLimiterMap{
		limiters: make(map[string]*ipLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// getLimiter returns the rate limiter for the given IP, creating one if it doesn't exist
func (rlm *rateLimiterMap) getLimiter(ip string) *rate.Limiter {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	limiter, exists := rlm.limiters[ip]
	if !exists {
		limiter = &ipLimiter{
			limiter:  rate.NewLimiter(rlm.rps, rlm.burst),
			lastSeen: time.Now(),
		}
		rlm.limiters[ip] = limiter
	} else {
		limiter.lastSeen = time.Now()
	}

	return limiter.limiter
}

// cleanupOldEntries removes IP entries that haven't been seen for the specified duration
func (rlm *rateLimiterMap) cleanupOldEntries(maxAge time.Duration) int {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var deletedCount int

	for ip, limiter := range rlm.limiters {
		if limiter.lastSeen.Before(cutoff) {
			delete(rlm.limiters, ip)
			deletedCount++
		}
	}

	return deletedCount
}

// getStats returns statistics about the rate limiter map
func (rlm *rateLimiterMap) getStats() (totalEntries int, rps float64, burst int) {
	rlm.mu.RLock()
	defer rlm.mu.RUnlock()

	return len(rlm.limiters), float64(rlm.rps), rlm.burst
}

// getClientIP extracts the real client IP from the request
// Handles X-Forwarded-For headers properly for proxied requests
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (comma-separated list)
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// Take the first IP in the chain (original client)
		ips := strings.Split(xForwardedFor, ",")
		ip := strings.TrimSpace(ips[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Check X-Real-IP header
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		if net.ParseIP(xRealIP) != nil {
			return xRealIP
		}
	}

	// Fall back to RemoteAddr (remove port if present)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.logger.ErrorWithContext(r.Context(), "panic recovered",
					"panic", err,
					"method", r.Method,
					"uri", r.URL.RequestURI(),
					"addr", r.RemoteAddr)
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) correlationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := r.Header.Get("X-Correlation-ID")
		if corrID == "" {
			corrID = uuid.New().String()
		}

		w.Header().Set("X-Correlation-ID", corrID)

		ctx := context.WithValue(r.Context(), "correlation_id", corrID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response recorder to capture the status code
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rr, r)

		duration := time.Since(start)

		corrID := r.Context().Value("correlation_id")

		app.logger.Info("HTTP request completed",
			"method", r.Method,
			"uri", r.URL.RequestURI(),
			"addr", r.RemoteAddr,
			"proto", r.Proto,
			"status", rr.statusCode,
			"duration_ms", duration.Milliseconds(),
			"correlation_id", corrID,
			"user_agent", r.Header.Get("User-Agent"))
	})
}

// responseRecorder wraps http.ResponseWriter to capture the status code
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// rateLimit middleware enforces per-IP rate limiting using token bucket algorithm
func (app *application) rateLimit(rateLimiterMap *rateLimiterMap) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting if disabled
			if !app.config.limiter.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Extract client IP
			ip := getClientIP(r)

			// Get rate limiter for this IP
			limiter := rateLimiterMap.getLimiter(ip)

			// Check if request is allowed
			if !limiter.Allow() {
				// Rate limit exceeded - calculate retry after time
				retryAfter := time.Second / time.Duration(rateLimiterMap.rps)

				// Log rate limit violation with correlation ID
				corrID := r.Context().Value("correlation_id")
				app.logger.WarnWithContext(r.Context(), "rate limit exceeded",
					"ip", ip,
					"correlation_id", corrID,
					"rps_limit", float64(rateLimiterMap.rps),
					"burst_limit", rateLimiterMap.burst,
					"method", r.Method,
					"uri", r.URL.RequestURI())

				// Send 429 rate limit exceeded response
				app.rateLimitExceededResponse(w, r, retryAfter)
				return
			}

			// Request allowed - continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}
