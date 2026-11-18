package auth

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10, 1.0)
	if tb == nil {
		t.Fatal("NewTokenBucket() returned nil")
	}
}

func TestTokenBucketAllow(t *testing.T) {
	tb := NewTokenBucket(5, 10.0) // 5 tokens, refill 10/sec

	// Should allow 5 requests initially
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Errorf("Expected Allow() = true at attempt %d", i+1)
		}
	}

	// Should be exhausted now
	if tb.Allow() {
		t.Error("Expected Allow() = false when bucket is empty")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	tb := NewTokenBucket(2, 100.0) // 2 tokens, fast refill

	// Consume all tokens
	tb.Allow()
	tb.Allow()

	// Give time for refill
	time.Sleep(20 * time.Millisecond)

	// Should have refilled at least 1 token
	if !tb.Allow() {
		t.Error("Expected Allow() = true after refill")
	}
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(true)
	if rl == nil {
		t.Fatal("NewRateLimiter() returned nil")
	}
	if !rl.enabled {
		t.Error("Expected enabled = true")
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	rl := NewRateLimiter(false)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 when disabled, got %d", rec.Code)
	}
}

func TestRateLimiterWithKey(t *testing.T) {
	rl := NewRateLimiter(true)

	var mu sync.Mutex
	count := 0

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))

	// Send a request with a specific key (default 60/min, so first one should succeed)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer test-key-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusTooManyRequests {
		t.Error("Expected first request with new key to succeed")
	}
}

func TestSetRateLimit(t *testing.T) {
	rl := NewRateLimiter(true)
	rl.SetRateLimit("restricted-key", 1)

	// First request should succeed
	bucket := rl.getBucket("restricted-key")
	if !bucket.Allow() {
		t.Error("Expected first request with restricted key to be allowed")
	}

	// Second request should be denied (only 1/min)
	if bucket.Allow() {
		t.Error("Expected second request to be rate limited (1 req/min)")
	}
}

func TestRateLimiterMiddlewareBlocks(t *testing.T) {
	rl := NewRateLimiter(true)
	// Configure a key with 0 capacity (effectively blocked)
	rl.SetRateLimit("blocked-key", 0)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer blocked-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429 for blocked key, got %d", rec.Code)
	}
}
