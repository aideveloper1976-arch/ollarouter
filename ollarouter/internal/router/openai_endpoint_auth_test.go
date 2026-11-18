package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/auth"
	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/logger"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
)

func newOpenAIEndpointTestRouter(t *testing.T, public bool) http.Handler {
	t.Helper()
	cfg := &config.Config{
		APIKeys:  []config.APIKeyConfig{{Name: "client", Key: "valid-client-key-12345"}},
		Settings: config.DefaultSettings(),
	}
	app := &App{
		Logger:   logger.New("error", "text"),
		Registry: provider.NewRegistry(),
		Auth:     auth.NewAuthenticator(cfg),
		Config:   cfg,
		Settings: RouterSettings{OpenAIEndpointPublic: public},
	}
	return NewRouter(app)
}

func postChatCompletions(h http.Handler, path, authHeader string) int {
	body := `{"model":"none/none","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code
}

func TestOpenAIEndpointRequiresAuthWhenKeysConfigured(t *testing.T) {
	h := newOpenAIEndpointTestRouter(t, false)
	for _, path := range []string{"/v1/chat/completions", "/chat/completions"} {
		if code := postChatCompletions(h, path, ""); code != http.StatusUnauthorized {
			t.Errorf("%s without key: expected 401, got %d", path, code)
		}
		if code := postChatCompletions(h, path, "Bearer wrong-key"); code != http.StatusUnauthorized {
			t.Errorf("%s with wrong key: expected 401, got %d", path, code)
		}
		if code := postChatCompletions(h, path, "Bearer valid-client-key-12345"); code == http.StatusUnauthorized {
			t.Errorf("%s with valid key: got 401", path)
		}
	}
}

func TestOpenAIEndpointPublicOptOut(t *testing.T) {
	h := newOpenAIEndpointTestRouter(t, true)
	if code := postChatCompletions(h, "/v1/chat/completions", ""); code == http.StatusUnauthorized {
		t.Fatalf("openai_endpoint_public=true should skip auth, got 401")
	}
}
