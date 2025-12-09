package middleware

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limits   map[string]int
	window   time.Duration
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limits: map[string]int{
			"auth":    5,  // 5 requests per minute
			"chat":    30, // 30 requests per minute
			"ai":      10, // 10 requests per minute
			"default": 60, // 60 requests per minute
		},
		window: time.Minute,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	// Clean up old requests
	requests := r.requests[key]
	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	r.requests[key] = validRequests

	// Check if limit exceeded
	limit := r.defaultLimit(key)
	if len(validRequests) >= limit {
		return false
	}

	// Add current request
	r.requests[key] = append(validRequests, now)
	return true
}

func (r *RateLimiter) defaultLimit(key string) int {
	if limit, exists := r.limits[key]; exists {
		return limit
	}
	return r.limits["default"]
}

func (r *RateLimiter) UnaryServerInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(r.unaryInterceptor)
}

func (r *RateLimiter) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return r.streamInterceptor
}

func (r *RateLimiter) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	if !r.Allow(info.FullMethod) {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
	return handler(ctx, req)
}

func (r *RateLimiter) streamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	if !r.Allow(info.FullMethod) {
		return status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
	return handler(srv, stream)
}
