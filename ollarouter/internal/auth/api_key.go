package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/config"
)

// contextKey is used for storing auth info in request context.
type contextKey string

const (
	// ContextKeyKeyInfo stores the authenticated API key info in the request context.
	ContextKeyKeyInfo contextKey = "auth:key_info"
)

// APIKeyInfo holds information about an API key.
type APIKeyInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	KeyPrefix     string    `json:"key_masked"`
	Hash          string    `json:"-"` // SHA-256 hash of the full key
	RateLimit     int       `json:"rate_limit"`
	MonthlyBudget float64   `json:"monthly_budget_usd"`
	CreatedAt     time.Time `json:"created_at"`
	Revoked       bool      `json:"revoked"`
}

// Authenticator validates API keys from HTTP requests.
type Authenticator struct {
	keys     map[string]*APIKeyInfo // key hash -> info
	byID     map[string]*APIKeyInfo // id -> info
	disabled bool
	mu       sync.RWMutex
}

// NewAuthenticator creates a new authenticator from config.
func NewAuthenticator(cfg *config.Config) *Authenticator {
	a := &Authenticator{
		keys: make(map[string]*APIKeyInfo),
		byID: make(map[string]*APIKeyInfo),
	}

	if len(cfg.APIKeys) == 0 {
		a.disabled = true
		return a
	}

	for _, kc := range cfg.APIKeys {
		hash := hashKey(kc.Key)
		id := generateID()
		keyInfo := &APIKeyInfo{
			ID:            id,
			Name:          kc.Name,
			KeyPrefix:     maskKey(kc.Key),
			Hash:          hash,
			RateLimit:     kc.RateLimit,
			MonthlyBudget: kc.MonthlyBudget,
			CreatedAt:     time.Now(),
		}
		a.keys[hash] = keyInfo
		a.byID[id] = keyInfo
	}

	return a
}

// Middleware returns an HTTP middleware that validates API keys.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.disabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for health endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/health/providers" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"authentication required"}`))
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid authorization format"}`))
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		info := a.lookup(token)
		if info == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid API key"}`))
			return
		}

		// Store key info in request context for downstream handlers
		ctx := context.WithValue(r.Context(), ContextKeyKeyInfo, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Validate checks if a raw key is valid.
func (a *Authenticator) Validate(rawKey string) bool {
	return a.isValid(rawKey)
}

// isValid checks the key hash against the store.
func (a *Authenticator) isValid(rawKey string) bool {
	return a.lookup(rawKey) != nil
}

// lookup returns the API key info for a raw key, or nil if invalid/revoked.
func (a *Authenticator) lookup(rawKey string) *APIKeyInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	hash := hashKey(rawKey)
	info, exists := a.keys[hash]
	if !exists || info.Revoked {
		return nil
	}
	return info
}

// GetKeyInfoFromContext extracts the authenticated API key info from the request context.
// Returns nil if the request was not authenticated.
func GetKeyInfoFromContext(r *http.Request) *APIKeyInfo {
	info, _ := r.Context().Value(ContextKeyKeyInfo).(*APIKeyInfo)
	return info
}

// GenerateKey creates a new API key and returns the raw key.
func (a *Authenticator) GenerateKey(name string, rateLimit int, budget float64) (string, error) {
	rawKey := generateRawKey()
	hash := hashKey(rawKey)
	id := generateID()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.keys[hash] = &APIKeyInfo{
		ID:            id,
		Name:          name,
		KeyPrefix:     maskKey(rawKey),
		Hash:          hash,
		RateLimit:     rateLimit,
		MonthlyBudget: budget,
		CreatedAt:     time.Now(),
	}
	a.byID[id] = a.keys[hash]
	a.disabled = false

	return rawKey, nil
}

// GenerateKeyFromRaw adds a key with a specific raw value.
func (a *Authenticator) GenerateKeyFromRaw(name string, rawKey string, rateLimit int, budget float64) {
	hash := hashKey(rawKey)
	id := generateID()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.keys[hash] = &APIKeyInfo{
		ID:            id,
		Name:          name,
		KeyPrefix:     maskKey(rawKey),
		Hash:          hash,
		RateLimit:     rateLimit,
		MonthlyBudget: budget,
		CreatedAt:     time.Now(),
	}
	a.byID[id] = a.keys[hash]
	a.disabled = false
}

// GetKeyInfoByID returns the API key info for the given ID.
func (a *Authenticator) GetKeyInfoByID(id string) *APIKeyInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.byID[id]
}

// FindAPIKeyConfigIndex finds a matching API key config index by ID.
func (a *Authenticator) FindAPIKeyConfigIndex(cfg *config.Config, id string) int {
	info := a.GetKeyInfoByID(id)
	if info == nil {
		return -1
	}
	for i, kc := range cfg.APIKeys {
		if maskKey(kc.Key) == info.KeyPrefix {
			return i
		}
	}
	return -1
}

// RevokeKey revokes a key by ID.
func (a *Authenticator) RevokeKey(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if info, ok := a.byID[id]; ok {
		info.Revoked = true
	}
}

// ListKeys returns all keys.
func (a *Authenticator) ListKeys() []*APIKeyInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]*APIKeyInfo, 0, len(a.byID))
	for _, info := range a.byID {
		result = append(result, info)
	}
	return result
}

// hashKey creates a SHA-256 hash of a key.
func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// maskKey masks a key for safe logging.
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// generateRawKey generates a random API key.
func generateRawKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "or_" + hex.EncodeToString(bytes)
}

// generateID generates a random ID.
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
