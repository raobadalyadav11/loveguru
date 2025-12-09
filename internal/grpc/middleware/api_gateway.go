package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"loveguru/internal/cache"
	"loveguru/internal/logger"
	"loveguru/internal/ratelimit"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// API Gateway Router handles routing between different microservices
type GatewayRouter struct {
	logger      *logger.Logger
	cache       *cache.Cache
	rateLimiter *ratelimit.RateLimiter
}

// NewGatewayRouter creates a new API gateway router
func NewGatewayRouter() *GatewayRouter {
	return &GatewayRouter{
		logger:      logger.NewLogger(),
		cache:       cache.NewCache("localhost:6379", "", 0),
		rateLimiter: ratelimit.NewRateLimiter(cache.NewCache("localhost:6379", "", 0)),
	}
}

// HTTPHandler handles HTTP requests and routes them to appropriate services
func (g *GatewayRouter) HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apply rate limiting
		clientIP := g.getClientIP(r)
		if !g.allowRequest(clientIP) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Add logging
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		// Log the request
		g.logger.Info(context.Background(), "Request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", clientIP,
		)
	})
}

// HealthCheckHandler provides health check endpoint
func (g *GatewayRouter) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
}

// MetricsHandler provides basic metrics
func (g *GatewayRouter) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metrics := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"status":    "operational",
		"version":   "1.0.0",
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"metrics": %v}`, metrics)
}

// ErrorHandler customizes error responses
func (g *GatewayRouter) ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	g.logger.Error(context.Background(), "Gateway error", err, "path", r.URL.Path, "method", r.Method)

	// Set appropriate HTTP status code
	statusCode := g.mapErrorToStatusCode(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	fmt.Fprintf(w, `{"error": "%s", "code": %d, "timestamp": "%s"}`,
		err.Error(), statusCode, time.Now().Format(time.RFC3339))
}

// mapErrorToStatusCode maps errors to HTTP status codes
func (g *GatewayRouter) mapErrorToStatusCode(err error) int {
	errorStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errorStr, "unauthenticated"):
		return http.StatusUnauthorized
	case strings.Contains(errorStr, "unauthorized"):
		return http.StatusForbidden
	case strings.Contains(errorStr, "not found"):
		return http.StatusNotFound
	case strings.Contains(errorStr, "invalid"):
		return http.StatusBadRequest
	case strings.Contains(errorStr, "deadline exceeded"):
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// allowRequest checks if the request is allowed based on rate limiting
func (g *GatewayRouter) allowRequest(clientIP string) bool {
	config := ratelimit.Config{
		RequestsPerMinute: 100,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
	}

	allowed, err := g.rateLimiter.Allow(clientIP, config)
	if err != nil {
		g.logger.Error(context.Background(), "Rate limiter error", err, "client_ip", clientIP)
		return true // Allow on error to avoid blocking
	}

	return allowed
}

// getClientIP extracts client IP from request
func (g *GatewayRouter) getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header first
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}

	return ip
}

// Helper types for middleware
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.statusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

// CreateGRPCCaller creates a gRPC client connection
func (g *GatewayRouter) CreateGRPCCaller(serviceAddr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, serviceAddr, grpc.WithInsecure())
	if err != nil {
		g.logger.Error(context.Background(), "Failed to connect to service", err, "service_addr", serviceAddr)
		return nil, err
	}

	return conn, nil
}

// AddMetadata adds metadata to gRPC context
func AddMetadata(ctx context.Context, key, value string) context.Context {
	md := metadata.New(map[string]string{key: value})
	return metadata.NewOutgoingContext(ctx, md)
}

// GetMetadata retrieves metadata from gRPC context
func GetMetadata(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get(key)
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}

// Middleware functions for HTTP handlers

// CorsMiddleware adds CORS headers
func (g *GatewayRouter) CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers
func (g *GatewayRouter) SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}
