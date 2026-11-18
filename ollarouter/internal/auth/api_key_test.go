package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/config"
)

func TestNewAuthenticator(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "or_test1234567890abcdef"},
		},
	}
	a := NewAuthenticator(cfg)
	if a == nil {
		t.Fatal("NewAuthenticator() returned nil")
	}
	if a.disabled {
		t.Error("Expected disabled = false when API keys are configured")
	}
}

func TestNewAuthenticatorDisabled(t *testing.T) {
	cfg := &config.Config{}
	a := NewAuthenticator(cfg)
	if a == nil {
		t.Fatal("NewAuthenticator() returned nil")
	}
	if !a.disabled {
		t.Error("Expected disabled = true when no API keys are configured")
	}
}

func TestValidateValidKey(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "or_test1234567890abcdef"},
		},
	}
	a := NewAuthenticator(cfg)

	if !a.Validate("or_test1234567890abcdef") {
		t.Error("Expected Validate() = true for valid key")
	}
}

func TestValidateInvalidKey(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "or_test1234567890abcdef"},
		},
	}
	a := NewAuthenticator(cfg)

	if a.Validate("wrong-key") {
		t.Error("Expected Validate() = false for invalid key")
	}
}

func TestMiddlewareValidKey(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "or_test1234567890abcdef"},
		},
	}
	a := NewAuthenticator(cfg)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/chat", nil)
	req.Header.Set("Authorization", "Bearer or_test1234567890abcdef")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid key, got %d", rec.Code)
	}
}

func TestMiddlewareNoAuth(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "or_test1234567890abcdef"},
		},
	}
	a := NewAuthenticator(cfg)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/chat", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for no auth, got %d", rec.Code)
	}
}

func TestMiddlewareHealthBypassesAuth(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "or_test1234567890abcdef"},
		},
	}
	a := NewAuthenticator(cfg)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Health endpoints should bypass auth
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 for /health without auth, got %d", rec.Code)
	}
}

func TestGenerateAndValidateKey(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "or_test1234567890abcdef"},
		},
	}
	a := NewAuthenticator(cfg)

	// Generate a new key
	rawKey, err := a.GenerateKey("new-key", 120, 50.0)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	if rawKey == "" {
		t.Fatal("GenerateKey() returned empty key")
	}

	// Validate the generated key
	if !a.Validate(rawKey) {
		t.Error("Expected Validate() = true for newly generated key")
	}

	// Verify the key info
	keys := a.ListKeys()
	found := false
	for _, k := range keys {
		if k.Name == "new-key" {
			found = true
			if k.RateLimit != 120 {
				t.Errorf("Expected RateLimit = 120, got %d", k.RateLimit)
			}
			if k.MonthlyBudget != 50.0 {
				t.Errorf("Expected MonthlyBudget = 50.0, got %f", k.MonthlyBudget)
			}
			break
		}
	}
	if !found {
		t.Error("Generated key not found in ListKeys()")
	}
}

func TestRevokeKey(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "or_test1234567890abcdef"},
		},
	}
	a := NewAuthenticator(cfg)

	// Get the key ID
	keys := a.ListKeys()
	if len(keys) == 0 {
		t.Fatal("Expected at least 1 key")
	}

	keyID := keys[0].ID
	a.RevokeKey(keyID)

	// Validate after revoke
	if a.Validate("or_test1234567890abcdef") {
		t.Error("Expected Validate() = false after revoke")
	}
}

func TestHashKey(t *testing.T) {
	h1 := hashKey("test-key")
	h2 := hashKey("test-key")
	h3 := hashKey("other-key")

	if h1 != h2 {
		t.Error("hashKey() should be deterministic")
	}
	if h1 == h3 {
		t.Error("Different keys should produce different hashes")
	}
	if len(h1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(h1))
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "***"},
		{"or_test1234567890abcdef", "or_t...cdef"},
		{"12345678", "***"},
		{"1234567890", "1234...7890"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := maskKey(tt.input); got != tt.want {
				t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
