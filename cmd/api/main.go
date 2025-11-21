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
	"syscall"
	"time"
)

var (
	buildTime string
	version   string
)

type application struct {
	config config
	logger *slog.Logger
}

type config struct {
	port int
	env  string
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

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
		config: cfg,
		logger: logger,
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

	logger.Info("starting server", "addr", srv.Addr, "env", cfg.env, "version", version, "buildTime", buildTime)

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
