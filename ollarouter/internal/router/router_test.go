package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/auth"
	"github.com/aideveloper1976-arch/ollarouter/internal/auth/upstream"
	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/logger"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

type dummyProvider struct{}

func (d *dummyProvider) Name() string { return "dummy" }
func (d *dummyProvider) Type() string { return "openai" }
func (d *dummyProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	return &types.ChatResponse{Model: req.Model, Message: types.Message{Role: "assistant", Content: "ok"}}, nil
}
func (d *dummyProvider) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	return handler(&types.StreamChunk{Content: "ok", Done: true})
}
func (d *dummyProvider) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	return &types.GenerateResponse{Model: req.Model, Response: "ok"}, nil
}
func (d *dummyProvider) GenerateStream(ctx context.Context, req *types.GenerateRequest, handler types.StreamHandler) error {
	return handler(&types.StreamChunk{Content: "ok", Done: true})
}
func (d *dummyProvider) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	return []types.ModelInfo{{Name: "dummy/gpt-4", ProviderName: "dummy"}}, nil
}
func (d *dummyProvider) CheckHealth(ctx context.Context) types.ProviderHealth {
	return types.ProviderHealth{Status: "online", LatencyMS: 1}
}

type dummyBalanceProvider struct {
	dummyProvider
}

func (d *dummyBalanceProvider) GetBalance(ctx context.Context) (*types.ProviderBalance, error) {
	return &types.ProviderBalance{
		Currency:         "USD",
		TotalCredits:     100,
		TotalUsage:       25,
		RemainingCredits: 75,
		Source:           "test",
	}, nil
}

func TestAdminProviderTestRouteExists(t *testing.T) {
	reg := provider.NewRegistry()
	if err := reg.Add(config.ProviderConfig{Name: "test-provider", Type: "openai", BaseURL: "https://api.openai.com/v1"}, &dummyProvider{}); err != nil {
		t.Fatalf("failed to add provider: %v", err)
	}

	app := &App{Registry: reg, Logger: logger.New("error", "text")}
	r := NewRouter(app)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/providers/test-provider/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminProviderTestRouteNotFound(t *testing.T) {
	reg := provider.NewRegistry()
	app := &App{Registry: reg, Logger: logger.New("error", "text")}
	r := NewRouter(app)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/providers/not-found-provider/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminProviderBalanceRouteExists(t *testing.T) {
	reg := provider.NewRegistry()
	if err := reg.Add(config.ProviderConfig{Name: "test-provider", Type: "openrouter", BaseURL: "https://openrouter.ai/api/v1"}, &dummyBalanceProvider{}); err != nil {
		t.Fatalf("failed to add provider: %v", err)
	}

	app := &App{Registry: reg, Logger: logger.New("error", "text")}
	r := NewRouter(app)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/providers/test-provider/balance", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAuthStatusRoute(t *testing.T) {
	app := &App{Logger: logger.New("error", "text")}
	r := NewRouter(app)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/auth/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAuthConnectRoute(t *testing.T) {
	app := &App{Logger: logger.New("error", "text")}
	r := NewRouter(app)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/auth/connect", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request (no body), got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAuthDisconnectRoute(t *testing.T) {
	app := &App{Logger: logger.New("error", "text")}
	r := NewRouter(app)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/auth/disconnect", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request (no body), got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAuthStatusRequiresAuthWhenKeysConfigured(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "admin", Key: "valid-admin-key-12345"},
		},
		Settings: config.DefaultSettings(),
	}
	authenticator := auth.NewAuthenticator(cfg)
	app := &App{
		Logger: logger.New("error", "text"),
		Auth:   authenticator,
		Config: cfg,
	}
	r := NewRouter(app)

	// Request without auth header should get 401
	req := httptest.NewRequest(http.MethodGet, "/admin/api/auth/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without auth header, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAuthConnectRequiresAuthWhenKeysConfigured(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "admin", Key: "valid-admin-key-12345"},
		},
		Settings: config.DefaultSettings(),
	}
	authenticator := auth.NewAuthenticator(cfg)
	app := &App{
		Logger: logger.New("error", "text"),
		Auth:   authenticator,
		Config: cfg,
	}
	r := NewRouter(app)

	// Request without auth header should get 401
	req := httptest.NewRequest(http.MethodPost, "/admin/api/auth/connect", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without auth header, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminProviderTestRouteRequiresAuthWhenKeysConfigured(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "admin", Key: "valid-admin-key-12345"},
		},
		Settings: config.DefaultSettings(),
	}
	authenticator := auth.NewAuthenticator(cfg)
	reg := provider.NewRegistry()
	_ = reg.Add(config.ProviderConfig{Name: "test-provider", Type: "openai", BaseURL: "https://api.openai.com/v1"}, &dummyProvider{})

	app := &App{
		Logger:   logger.New("error", "text"),
		Auth:     authenticator,
		Config:   cfg,
		Registry: reg,
	}
	r := NewRouter(app)

	// Request without auth header should get 401
	req := httptest.NewRequest(http.MethodGet, "/admin/api/providers/test-provider/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without auth header, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminConfigYAMLMasksAPIKeys(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "admin", Key: "valid-admin-key-12345"},
		},
		Settings: config.DefaultSettings(),
	}
	authenticator := auth.NewAuthenticator(cfg)
	reg := provider.NewRegistry()
	_ = reg.Add(config.ProviderConfig{Name: "test-provider", Type: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "test-api-key-example-987654321"}, &dummyProvider{})

	app := &App{
		Logger:   logger.New("error", "text"),
		Auth:     authenticator,
		Config:   cfg,
		Registry: reg,
	}
	r := NewRouter(app)

	// Make authenticated request
	req := httptest.NewRequest(http.MethodGet, "/admin/api/config/yaml", nil)
	req.Header.Set("Authorization", "Bearer valid-admin-key-12345")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if body == "" {
		t.Fatal("expected non-empty YAML body")
	}

	// Full API key must NOT appear in plaintext in the YAML output
	if strings.Contains(body, "test-api-key-example-987654321") {
		t.Error("API key 'test-api-key-example-987654321' exposed in full plaintext in YAML output")
	}
}

func TestAdminMeRouteWorksWithoutAuth(t *testing.T) {
	cfg := &config.Config{
		APIKeys: []config.APIKeyConfig{
			{Name: "admin", Key: "valid-admin-key-12345"},
		},
		Settings: config.DefaultSettings(),
	}
	authenticator := auth.NewAuthenticator(cfg)
	app := &App{
		Logger: logger.New("error", "text"),
		Auth:   authenticator,
		Config: cfg,
	}
	r := NewRouter(app)

	// /admin/api/me must work without auth (SPA uses it to detect auth state)
	req := httptest.NewRequest(http.MethodGet, "/admin/api/me", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /admin/api/me without auth, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAuthStatusWithUpstreamAuthReturnsOk(t *testing.T) {
	cfg := &config.Config{
		Settings: config.DefaultSettings(),
		Providers: []config.ProviderConfig{
			{
				Name:    "chatgpt-plus",
				Type:    "chatgpt-plus",
				BaseURL: "https://chatgpt.com",
				Auth: &config.AuthSettings{
					Mode: "session_cookie",
				},
			},
		},
	}

	dir := t.TempDir()
	upstreamAuth, err := upstream.InitFromProviders(cfg.Providers, filepath.Join(dir, "creds.enc"), "test-encryption-key-32bytes!")
	if err != nil {
		t.Fatalf("InitFromProviders() error = %v", err)
	}
	if upstreamAuth == nil {
		t.Fatal("InitFromProviders() returned nil - expected manager for auth-configured provider")
	}

	reg := provider.NewRegistry()
	_ = reg.Add(config.ProviderConfig{
		Name:    "chatgpt-plus",
		Type:    "chatgpt-plus",
		BaseURL: "https://chatgpt.com",
		Auth:    &config.AuthSettings{Mode: "session_cookie"},
	}, &dummyProvider{})

	app := &App{
		Logger:       logger.New("error", "text"),
		Registry:     reg,
		UpstreamAuth: upstreamAuth,
		Config:       cfg,
	}
	r := NewRouter(app)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/auth/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %q", resp["status"])
	}
	providers, ok := resp["providers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'providers' map in response")
	}
	status, ok := providers["chatgpt-plus"].(string)
	if !ok || status != "missing" {
		t.Fatalf("expected chatgpt-plus status 'missing', got %q", status)
	}
}

func TestAdminAuthStatusWithoutUpstreamAuthReturnsNotConfigured(t *testing.T) {
	app := &App{
		Logger: logger.New("error", "text"),
	}
	r := NewRouter(app)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/auth/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp["status"] != "not_configured" {
		t.Fatalf("expected status 'not_configured', got %q", resp["status"])
	}
}
