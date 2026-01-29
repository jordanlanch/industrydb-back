package logger

import (
	"log/slog"
	"os"
)

// Logger defines structured logging interface
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
	With(args ...any) Logger
}

// SlogLogger implements Logger using Go's standard log/slog
type SlogLogger struct {
	logger *slog.Logger
}

// New creates a new structured logger with the specified level
func New(level string) Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	return &SlogLogger{logger: slog.New(handler)}
}

// Info logs an informational message
func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Error logs an error message
func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// Warn logs a warning message
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Debug logs a debug message
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// With returns a new logger with the specified attributes
func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{logger: l.logger.With(args...)}
}

// Default returns a default logger instance
func Default() Logger {
	return New("info")
}
