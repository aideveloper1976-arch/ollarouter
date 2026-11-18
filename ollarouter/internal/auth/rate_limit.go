package auth

import (
	"net/http"
	"sync"
	"time"
)

// TokenBucket implements the token bucket rate limiting algorithm.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket.
func NewTokenBucket(capacity int, refillPerSecond float64) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(capacity),
		capacity:   float64(capacity),
		refillRate: refillPerSecond,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = tb.tokens + elapsed*tb.refillRate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// RateLimiter implements per-key rate limiting middleware.
type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
	enabled bool
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(enabled bool) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		enabled: enabled,
	}
}

// Middleware returns an HTTP middleware for rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.enabled {
			next.ServeHTTP(w, r)
			return
		}

		key := "default"
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 {
			key = authHeader[7:] // Strip "Bearer "
		}

		bucket := rl.getBucket(key)
		if !bucket.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, exists = rl.buckets[key]; exists {
		return bucket
	}

	bucket = NewTokenBucket(60, 1.0) // 60 req/min default
	rl.buckets[key] = bucket
	return bucket
}

// SetRateLimit sets the rate limit for a key.
func (rl *RateLimiter) SetRateLimit(key string, capacity int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.buckets[key] = NewTokenBucket(capacity, float64(capacity)/60.0)
}
