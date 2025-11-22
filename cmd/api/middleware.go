package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

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
