package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"fizzbuzz/internal/data"
	"fizzbuzz/internal/jsonlog"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	buildTime string
	version   string
)

type application struct {
	config      config
	logger      *jsonlog.Logger
	statistics  statisticsHandler
	rateLimiter *rateLimiterMap
}

// statisticsHandler provides interface for statistics operations
// Story 4.6: Direct PostgreSQL access with context-aware operations
type statisticsHandler struct {
	service *data.StatisticsService // PostgreSQL-backed implementation only
}

// Record records statistics using PostgreSQL service with context and timeout
func (sh *statisticsHandler) Record(ctx context.Context, input *data.FizzBuzzInput) error {
	if sh.service == nil {
		return errors.New("statistics service not initialized")
	}
	return sh.service.Record(ctx, input)
}

// GetMostFrequent gets most frequent statistics from PostgreSQL with context
func (sh *statisticsHandler) GetMostFrequent(ctx context.Context) (*data.StatisticsEntry, error) {
	if sh.service == nil {
		return nil, errors.New("statistics service not initialized")
	}
	return sh.service.GetMostFrequent(ctx)
}

// Legacy compatibility methods for transition period
// RecordLegacy provides legacy-compatible Record method (no context, no error return)
func (sh *statisticsHandler) RecordLegacy(input *data.FizzBuzzInput, logger *jsonlog.Logger) {
	// Create context with reasonable timeout for legacy API compatibility
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := sh.Record(ctx, input)
	if err != nil {
		if logger != nil {
			logger.Warn("legacy statistics recording failed",
				"error", err,
				"operation", "RecordLegacy",
				"parameters", input)
		}
	}
}

// GetMostFrequentLegacy provides legacy-compatible GetMostFrequent method
func (sh *statisticsHandler) GetMostFrequentLegacy(logger *jsonlog.Logger) *data.StatisticsEntry {
	// Create context with reasonable timeout for legacy API compatibility
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	entry, err := sh.GetMostFrequent(ctx)
	if err != nil {
		if logger != nil {
			logger.Warn("legacy statistics retrieval failed",
				"error", err,
				"operation", "GetMostFrequentLegacy")
		}
		return nil
	}
	return entry
}

type config struct {
	port     int
	env      string
	logLevel string

	db struct {
		host         string
		port         int
		name         string
		user         string
		password     string
		sslMode      string
		maxConns     int
		maxIdleConns int
		maxLifetime  time.Duration
	}

	limiter struct {
		enabled bool
		rps     float64
		burst   int
	}
}

// getEnvString returns environment variable value or default if not set
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default if not set/invalid
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvFloat returns environment variable as float64 or default if not set/invalid
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// getEnvBool returns environment variable as bool or default if not set/invalid
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// getEnvDuration returns environment variable as duration or default if not set/invalid
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// initializePostgreSQLStatistics initializes PostgreSQL connection pool and statistics service
// Story 4.6: Direct PostgreSQL access with connection pooling and context-aware operations
func initializePostgreSQLStatistics(cfg config, logger *jsonlog.Logger) (statisticsHandler, error) {
	// Build PostgreSQL connection string
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.db.host, cfg.db.port, cfg.db.user, cfg.db.password, cfg.db.name, cfg.db.sslMode)

	// Configure connection pool with optimized settings for FizzBuzz workload
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return statisticsHandler{}, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Connection pooling configuration for optimal performance
	poolConfig.MaxConns = int32(cfg.db.maxConns)
	poolConfig.MinConns = 2                         // Maintain minimum connections for immediate availability
	poolConfig.MaxConnLifetime = cfg.db.maxLifetime // Connection refresh for long-running applications
	poolConfig.MaxConnIdleTime = 10 * time.Minute   // Close idle connections to free resources
	poolConfig.HealthCheckPeriod = 1 * time.Minute  // Regular health checks for connection validity

	// Context with timeout for initial connection setup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create connection pool with timeout
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return statisticsHandler{}, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connectivity with ping
	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return statisticsHandler{}, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize repository with optimized timeout for FizzBuzz operations
	operationTimeout := 3 * time.Second // Fast response times for API endpoints
	repository := data.NewPostgreSQLStatisticsRepository(pool, operationTimeout)

	// Create statistics service with repository
	service := data.NewStatisticsService(repository)

	logger.Info("PostgreSQL statistics initialized successfully",
		"db_host", cfg.db.host,
		"db_port", cfg.db.port,
		"db_name", cfg.db.name,
		"max_connections", cfg.db.maxConns,
		"operation_timeout", operationTimeout,
		"pool_health_check_period", poolConfig.HealthCheckPeriod)

	return statisticsHandler{
		service: service,
	}, nil
}

// initializeRateLimiter creates a new rate limiter with cleanup goroutine
func initializeRateLimiter(cfg config, logger *jsonlog.Logger) *rateLimiterMap {
	// Create rate limiter map
	rlm := newRateLimiterMap(cfg.limiter.rps, cfg.limiter.burst)

	// Start background cleanup goroutine if rate limiting is enabled
	if cfg.limiter.enabled {
		go func() {
			cleanupInterval := 1 * time.Minute // Run cleanup every minute
			maxAge := 1 * time.Hour            // Remove entries older than 1 hour
			ticker := time.NewTicker(cleanupInterval)
			defer ticker.Stop()

			logger.Info("rate limiter cleanup goroutine started",
				"cleanup_interval", cleanupInterval,
				"max_age", maxAge)

			for {
				select {
				case <-ticker.C:
					deletedCount := rlm.cleanupOldEntries(maxAge)
					totalEntries, rps, burst := rlm.getStats()

					if deletedCount > 0 {
						logger.Debug("rate limiter cleanup completed",
							"deleted_entries", deletedCount,
							"remaining_entries", totalEntries,
							"rps_limit", rps,
							"burst_limit", burst)
					}
				}
			}
		}()
	}

	logger.Info("rate limiter initialized",
		"enabled", cfg.limiter.enabled,
		"rps", cfg.limiter.rps,
		"burst", cfg.limiter.burst)

	return rlm
}

func main() {
	var cfg config

	// Command line flags (take precedence over environment variables)
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.logLevel, "log-level", "info", "Log level (debug|info|warn|error)")

	// Rate limiter flags
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiting")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2.0, "Rate limiter requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst size")

	flag.Parse()

	// Environment variable configuration with fallbacks
	// API Configuration
	cfg.port = getEnvInt("API_PORT", cfg.port)
	cfg.env = getEnvString("API_ENV", cfg.env)
	cfg.logLevel = getEnvString("LOG_LEVEL", cfg.logLevel)

	// Database Configuration
	cfg.db.host = getEnvString("DB_HOST", "localhost")
	cfg.db.port = getEnvInt("DB_PORT", 5432)
	cfg.db.name = getEnvString("DB_NAME", "fizzbuzz")
	cfg.db.user = getEnvString("DB_USER", "fizzbuzz_user")
	cfg.db.password = getEnvString("DB_PASSWORD", "fizzbuzz_pass")
	cfg.db.sslMode = getEnvString("DB_SSL_MODE", "disable")
	cfg.db.maxConns = getEnvInt("DB_MAX_CONNECTIONS", 25)
	cfg.db.maxIdleConns = getEnvInt("DB_MAX_IDLE_CONNECTIONS", 5)
	cfg.db.maxLifetime = getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)

	// Rate Limiter Configuration (use flag values as defaults)
	cfg.limiter.enabled = getEnvBool("RATE_LIMITER_ENABLED", cfg.limiter.enabled)
	cfg.limiter.rps = getEnvFloat("RATE_LIMITER_RPS", cfg.limiter.rps)
	cfg.limiter.burst = getEnvInt("RATE_LIMITER_BURST", cfg.limiter.burst)

	// Parse log level
	var level jsonlog.Level
	switch cfg.logLevel {
	case "debug":
		level = jsonlog.LevelDebug
	case "info":
		level = jsonlog.LevelInfo
	case "warn":
		level = jsonlog.LevelWarn
	case "error":
		level = jsonlog.LevelError
	default:
		level = jsonlog.LevelInfo
	}

	logger := jsonlog.New(os.Stdout, level, cfg.env)

	// Story 4.6: Initialize PostgreSQL Statistics with Connection Pooling
	// Direct PostgreSQL access approach with proper connection pooling and context-aware operations
	statsHandler, err := initializePostgreSQLStatistics(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize PostgreSQL statistics, terminating application",
			"error", err,
			"db_host", cfg.db.host,
			"db_port", cfg.db.port)
		os.Exit(1)
	}

	// Story 5.2: Initialize Rate Limiter with IP-based controls
	rateLimiter := initializeRateLimiter(cfg, logger)

	app := &application{
		config:      cfg,
		logger:      logger,
		statistics:  statsHandler,
		rateLimiter: rateLimiter,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		logger.Info("shutdown initiated",
			"signal", s,
			"timeout", "5s",
			"addr", srv.Addr)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		// Story 4.6: Cleanup PostgreSQL connections on shutdown
		if app.statistics.service != nil {
			logger.Info("closing database connections")

			err := app.statistics.service.Close()
			if err != nil {
				logger.Error("failed to close database connections", "error", err)
			} else {
				logger.Info("database connections closed successfully")
			}
		}

		logger.Info("background tasks completed",
			"shutdown_timeout", "5s")

		shutdownError <- nil
	}()

	logger.Info("starting server",
		"addr", srv.Addr,
		"env", cfg.env,
		"version", version,
		"buildTime", buildTime,
		"db_host", cfg.db.host,
		"db_port", cfg.db.port,
		"db_name", cfg.db.name,
		"db_ssl_mode", cfg.db.sslMode,
		"rate_limiter_enabled", cfg.limiter.enabled,
		"rate_limiter_rps", cfg.limiter.rps)

	listenErr := srv.ListenAndServe()
	if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
		logger.Error("server failed to start or crashed",
			"error", listenErr,
			"addr", srv.Addr,
			"env", cfg.env)
		os.Exit(1)
	}

	err = <-shutdownError
	if err != nil {
		logger.Error("graceful shutdown failed",
			"error", err,
			"addr", srv.Addr)
		os.Exit(1)
	}

	logger.Info("server stopped gracefully",
		"addr", srv.Addr,
		"env", cfg.env)
}
