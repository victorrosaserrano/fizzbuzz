package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"fizzbuzz/internal/data"
)

var (
	buildTime string
	version   string
)

type application struct {
	config     config
	logger     *slog.Logger
	statistics *data.StatisticsTracker
}

type config struct {
	port int
	env  string

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

func main() {
	var cfg config

	// Command line flags (take precedence over environment variables)
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	// Environment variable configuration with fallbacks
	// API Configuration
	cfg.port = getEnvInt("API_PORT", cfg.port)
	cfg.env = getEnvString("API_ENV", cfg.env)

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

	// Rate Limiter Configuration
	cfg.limiter.enabled = getEnvBool("RATE_LIMITER_ENABLED", true)
	cfg.limiter.rps = getEnvFloat("RATE_LIMITER_RPS", 10.0)
	cfg.limiter.burst = getEnvInt("RATE_LIMITER_BURST", 20)

	var logger *slog.Logger

	if cfg.env == "development" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	app := &application{
		config:     cfg,
		logger:     logger,
		statistics: data.NewStatisticsTracker(),
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

	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed to start or crashed",
			"error", err,
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
