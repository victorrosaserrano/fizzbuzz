package main

import (
	"context"
	"net/http"
	"time"

	"fizzbuzz/internal/data"
	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/fizzbuzz", app.fizzbuzzHandler)
	router.HandlerFunc(http.MethodGet, "/v1/statistics", app.statisticsHandler)

	return app.correlationID(app.logRequest(app.rateLimit(app.rateLimiter)(app.recoverPanic(router))))
}

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// Method validation - only GET allowed (AC: Monitoring integration)
	if r.Method != http.MethodGet {
		app.methodNotAllowedResponse(w, r)
		return
	}

	start := time.Now()
	app.logger.InfoWithContext(r.Context(), "health check requested")

	// Create context with timeout for database validation
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	// Build system info
	systemInfo := data.SystemInfo{
		Environment: app.config.env,
		Version:     version, // Use build-time injected version
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	// Initialize health response with basic status
	healthResponse := data.HealthCheckResponse{
		Status:     "available",
		SystemInfo: systemInfo,
	}

	// Get database health information (AC: Operational insights)
	dbHealth, err := app.statistics.GetDatabaseHealth(ctx)
	if err != nil {
		// Degraded state when database unavailable but API functional (AC: System status)
		app.logger.WarnWithContext(ctx, "database health check failed",
			"error", err,
			"response_time_ms", time.Since(start).Milliseconds())

		healthResponse.Status = "degraded"
		healthResponse.Database = &data.DatabaseHealthInfo{
			Status:         "disconnected",
			ResponseTimeMs: -1,
		}
	} else {
		// Database is healthy - extract connection metrics
		healthResponse.Database = &data.DatabaseHealthInfo{
			Status:         "connected",
			ResponseTimeMs: time.Since(start).Milliseconds(),
		}

		// Extract connection pool metrics if available
		if status, ok := dbHealth["status"].(string); ok && status == "healthy" {
			if totalConns, ok := dbHealth["total_connections"].(int32); ok {
				healthResponse.Database.MaxConns = totalConns
			}
			if idleConns, ok := dbHealth["idle_connections"].(int32); ok {
				healthResponse.Database.IdleConns = idleConns
			}
			if acquiredConns, ok := dbHealth["acquired_connections"].(int32); ok {
				healthResponse.Database.ActiveConns = acquiredConns
			}
		}
	}

	// Determine HTTP status code based on health status
	statusCode := http.StatusOK
	if healthResponse.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	} else if healthResponse.Status == "unavailable" {
		statusCode = http.StatusServiceUnavailable
	}

	// Return response using JSON envelope format (AC: System information)
	responseData := envelope{"data": healthResponse}
	err = app.writeJSON(w, statusCode, responseData, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	responseTime := time.Since(start).Milliseconds()
	app.logger.InfoWithContext(ctx, "health check completed",
		"status", healthResponse.Status,
		"response_time_ms", responseTime,
		"database_status", healthResponse.Database.Status)
}
