package jsonlog

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, LevelInfo, "production")

	logger.Info("test message", "key1", "value1", "key2", 123)

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	if err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if entry["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", entry["level"])
	}

	if entry["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got %v", entry["msg"])
	}

	if entry["key1"] != "value1" {
		t.Errorf("expected key1 'value1', got %v", entry["key1"])
	}

	if entry["key2"] != float64(123) {
		t.Errorf("expected key2 123, got %v", entry["key2"])
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, LevelError, "production")

	logger.Error("error occurred", "error", "test error", "code", 500)

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	if err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if entry["level"] != "ERROR" {
		t.Errorf("expected level ERROR, got %v", entry["level"])
	}

	if entry["msg"] != "error occurred" {
		t.Errorf("expected msg 'error occurred', got %v", entry["msg"])
	}

	if entry["error"] != "test error" {
		t.Errorf("expected error 'test error', got %v", entry["error"])
	}
}

func TestLogger_Write(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, LevelInfo, "production")

	message := `{"level":"INFO","msg":"test write","key":"value"}`
	n, err := logger.Write([]byte(message))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != len(message) {
		t.Errorf("expected bytes written %d, got %d", len(message), n)
	}

	var entry map[string]any
	err = json.Unmarshal(buf.Bytes(), &entry)
	if err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if entry["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", entry["level"])
	}

	if entry["msg"] != "test write" {
		t.Errorf("expected msg 'test write', got %v", entry["msg"])
	}

	if entry["key"] != "value" {
		t.Errorf("expected key 'value', got %v", entry["key"])
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name     string
		minLevel Level
		logLevel string
		method   func(*Logger)
		expected bool
	}{
		{"Debug with Debug level", LevelDebug, "DEBUG", func(l *Logger) { l.Debug("test") }, true},
		{"Debug with Info level", LevelInfo, "DEBUG", func(l *Logger) { l.Debug("test") }, false},
		{"Info with Debug level", LevelDebug, "INFO", func(l *Logger) { l.Info("test") }, true},
		{"Info with Info level", LevelInfo, "INFO", func(l *Logger) { l.Info("test") }, true},
		{"Info with Error level", LevelError, "INFO", func(l *Logger) { l.Info("test") }, false},
		{"Error with any level", LevelDebug, "ERROR", func(l *Logger) { l.Error("test") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf, tt.minLevel, "production")

			tt.method(logger)

			if tt.expected {
				if buf.Len() == 0 {
					t.Error("expected log output, but buffer is empty")
				}
			} else {
				if buf.Len() > 0 {
					t.Error("expected no log output, but buffer contains data")
				}
			}
		})
	}
}

func TestLogger_EnvironmentAware(t *testing.T) {
	tests := []struct {
		env      string
		expected string
	}{
		{"development", "level=INFO msg=\"test message\" key=value"},
		{"production", `{"level":"INFO","msg":"test message","key":"value"`},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf, LevelInfo, tt.env)

			logger.Info("test message", "key", "value")

			output := buf.String()
			if !strings.Contains(output, "test message") {
				t.Errorf("expected output to contain 'test message', got: %s", output)
			}

			if tt.env == "development" {
				if !strings.Contains(output, "level=INFO") {
					t.Errorf("development format should be readable text format")
				}
			} else {
				var entry map[string]any
				err := json.Unmarshal(buf.Bytes(), &entry)
				if err != nil {
					t.Errorf("production format should be valid JSON: %v", err)
				}
			}
		})
	}
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, LevelInfo, "production")

	ctx := context.WithValue(context.Background(), "correlation_id", "test-123")
	logger.InfoWithContext(ctx, "test message", "key", "value")

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	if err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if entry["correlation_id"] != "test-123" {
		t.Errorf("expected correlation_id 'test-123', got %v", entry["correlation_id"])
	}

	if entry["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got %v", entry["msg"])
	}
}

func TestLogger_WithContextNoCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, LevelInfo, "production")

	ctx := context.Background()
	logger.InfoWithContext(ctx, "test message", "key", "value")

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	if err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if _, exists := entry["correlation_id"]; exists {
		t.Error("correlation_id should not be present when not in context")
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(99), "INFO"}, // Default case
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}
