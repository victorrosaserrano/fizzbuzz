package jsonlog

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"
)

type Level int8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

func (l Level) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
	slogger  *slog.Logger
}

func New(out io.Writer, minLevel Level, env string) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: minLevel.ToSlogLevel(),
	}

	if env == "development" {
		handler = slog.NewTextHandler(out, opts)
	} else {
		handler = slog.NewJSONHandler(out, opts)
	}

	return &Logger{
		out:      out,
		minLevel: minLevel,
		slogger:  slog.New(handler),
	}
}

func (l *Logger) Info(msg string, attrs ...any) {
	l.slogger.Info(msg, attrs...)
}

func (l *Logger) Error(msg string, attrs ...any) {
	l.slogger.Error(msg, attrs...)
}

func (l *Logger) Debug(msg string, attrs ...any) {
	l.slogger.Debug(msg, attrs...)
}

func (l *Logger) Warn(msg string, attrs ...any) {
	l.slogger.Warn(msg, attrs...)
}

func (l *Logger) Write(message []byte) (int, error) {
	var entry map[string]any

	err := json.Unmarshal(message, &entry)
	if err != nil {
		l.Error("failed to unmarshal log entry", "error", err, "raw_message", string(message))
		return len(message), nil
	}

	level, exists := entry["level"]
	if !exists {
		level = "INFO"
	}

	msg, exists := entry["msg"]
	if !exists {
		msg = "log entry"
	}

	delete(entry, "level")
	delete(entry, "msg")

	attrs := make([]any, 0, len(entry)*2)
	for key, value := range entry {
		attrs = append(attrs, key, value)
	}

	switch level {
	case "DEBUG":
		l.Debug(msg.(string), attrs...)
	case "INFO":
		l.Info(msg.(string), attrs...)
	case "WARN":
		l.Warn(msg.(string), attrs...)
	case "ERROR":
		l.Error(msg.(string), attrs...)
	default:
		l.Info(msg.(string), attrs...)
	}

	return len(message), nil
}

func (l *Logger) InfoWithContext(ctx context.Context, msg string, attrs ...any) {
	if corrID := ctx.Value("correlation_id"); corrID != nil {
		attrs = append(attrs, "correlation_id", corrID)
	}
	l.Info(msg, attrs...)
}

func (l *Logger) ErrorWithContext(ctx context.Context, msg string, attrs ...any) {
	if corrID := ctx.Value("correlation_id"); corrID != nil {
		attrs = append(attrs, "correlation_id", corrID)
	}
	l.Error(msg, attrs...)
}

func (l *Logger) DebugWithContext(ctx context.Context, msg string, attrs ...any) {
	if corrID := ctx.Value("correlation_id"); corrID != nil {
		attrs = append(attrs, "correlation_id", corrID)
	}
	l.Debug(msg, attrs...)
}

func (l *Logger) WarnWithContext(ctx context.Context, msg string, attrs ...any) {
	if corrID := ctx.Value("correlation_id"); corrID != nil {
		attrs = append(attrs, "correlation_id", corrID)
	}
	l.Warn(msg, attrs...)
}

func (l *Logger) PrintInfo(msg string, properties map[string]string) {
	attrs := make([]any, 0, len(properties)*2)
	for key, value := range properties {
		attrs = append(attrs, key, value)
	}
	l.Info(msg, attrs...)
}

func (l *Logger) PrintError(err error, properties map[string]string) {
	attrs := make([]any, 0, len(properties)*2+2)
	attrs = append(attrs, "error", err.Error())

	for key, value := range properties {
		attrs = append(attrs, key, value)
	}

	l.Error("application error", attrs...)
}

func (l *Logger) PrintFatal(err error, properties map[string]string) {
	attrs := make([]any, 0, len(properties)*2+4)
	attrs = append(attrs, "error", err.Error(), "stack", string(debug.Stack()))

	for key, value := range properties {
		attrs = append(attrs, key, value)
	}

	l.Error("fatal error", attrs...)
	os.Exit(1)
}
