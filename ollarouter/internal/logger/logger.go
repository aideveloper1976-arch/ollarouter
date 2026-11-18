package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with convenience methods.
type Logger struct {
	*slog.Logger
	level  slog.Level
	format string
}

// New creates a new Logger.
func New(level string, format string) *Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: l}

	if format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
		level:  l,
		format: format,
	}
}

// WithRequestID returns a logger with request_id attribute.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.With("request_id", requestID),
		level:  l.level,
		format: l.format,
	}
}

// MaskKey masks an API key for safe logging.
// Shows first 3 and last 4 characters, e.g., "sk-...xxxx"
func MaskKey(key string) string {
	if len(key) <= 10 {
		return "***"
	}
	return key[:3] + "..." + key[len(key)-4:]
}

// ContextKey is a type for context keys used by the logger.
type ContextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey ContextKey = "request_id"
	// ProviderKey is the context key for provider name.
	ProviderKey ContextKey = "provider"
	// ModelKey is the context key for model name.
	ModelKey ContextKey = "model"
)

// WithContext returns a logger with fields extracted from context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}
	attrs := []any{}
	if rid, ok := ctx.Value(RequestIDKey).(string); ok && rid != "" {
		attrs = append(attrs, slog.String("request_id", rid))
	}
	if prov, ok := ctx.Value(ProviderKey).(string); ok && prov != "" {
		attrs = append(attrs, slog.String("provider", prov))
	}
	if model, ok := ctx.Value(ModelKey).(string); ok && model != "" {
		attrs = append(attrs, slog.String("model", model))
	}
	if len(attrs) == 0 {
		return l
	}
	return &Logger{
		Logger: l.With(attrs...),
		level:  l.level,
		format: l.format,
	}
}
