package logger

import (
	"context"
	"log"
	"time"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Info(ctx context.Context, message string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", message, fields)
}

func (l *Logger) Error(ctx context.Context, message string, err error, fields ...interface{}) {
	log.Printf("[ERROR] %s - %v %v", message, err, fields)
}

func (l *Logger) Warn(ctx context.Context, message string, fields ...interface{}) {
	log.Printf("[WARN] %s %v", message, fields)
}

func (l *Logger) Debug(ctx context.Context, message string, fields ...interface{}) {
	log.Printf("[DEBUG] %s %v", message, fields)
}

func (l *Logger) WithFields(fields ...interface{}) *Logger {
	// In a real implementation, this would return a logger with structured fields
	return l
}

func (l *Logger) WithRequestID(requestID string) *Logger {
	// In a real implementation, this would include request ID in all logs
	return l
}

func (l *Logger) WithUserID(userID string) *Logger {
	// In a real implementation, this would include user ID in all logs
	return l
}

type Middleware struct {
	logger *Logger
}

func NewMiddleware(logger *Logger) *Middleware {
	return &Middleware{logger: logger}
}

func (m *Middleware) LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	m.logger.Info(ctx, "HTTP Request",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration_ms", duration.Milliseconds())
}

func (m *Middleware) LogError(ctx context.Context, operation string, err error) {
	m.logger.Error(ctx, "Operation failed", err, "operation", operation)
}

func (m *Middleware) LogDatabaseQuery(ctx context.Context, query string, duration time.Duration, err error) {
	if err != nil {
		m.logger.Error(ctx, "Database query failed", err, "query", query, "duration_ms", duration.Milliseconds())
	} else {
		m.logger.Debug(ctx, "Database query executed", "query", query, "duration_ms", duration.Milliseconds())
	}
}

func (m *Middleware) LogExternalAPICall(ctx context.Context, service, endpoint string, duration time.Duration, err error) {
	if err != nil {
		m.logger.Error(ctx, "External API call failed", err, "service", service, "endpoint", endpoint, "duration_ms", duration.Milliseconds())
	} else {
		m.logger.Info(ctx, "External API call successful", "service", service, "endpoint", endpoint, "duration_ms", duration.Milliseconds())
	}
}
