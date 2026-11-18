package config

import (
	"os"
	"testing"
)

func TestParseMinimalConfig(t *testing.T) {
	yaml := `
settings:
  port: 11435
providers:
  - name: test-openai
    type: openai
    api_key: "${TEST_KEY}"
    models:
      - name: gpt-4o
`
	os.Setenv("TEST_KEY", "sk-test123")
	defer os.Unsetenv("TEST_KEY")

	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Settings.Port != 11435 {
		t.Errorf("Port = %d, want 11435", cfg.Settings.Port)
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("len(Providers) = %d, want 1", len(cfg.Providers))
	}
	if cfg.Providers[0].APIKey != "sk-test123" {
		t.Errorf("APIKey = %q, want sk-test123", cfg.Providers[0].APIKey)
	}
}

func TestParseEnvInterpolation(t *testing.T) {
	os.Setenv("MY_KEY", "my-secret-value")
	defer os.Unsetenv("MY_KEY")

	yaml := `
providers:
  - name: test
    type: openai
    api_key: "${MY_KEY}"
    models:
      - name: gpt-4
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Providers[0].APIKey != "my-secret-value" {
		t.Errorf("APIKey = %q, want my-secret-value", cfg.Providers[0].APIKey)
	}
}

func TestParseDefaultValue(t *testing.T) {
	yaml := `
providers:
  - name: test
    type: openai
    api_key: "${MISSING_KEY:-default-key}"
    models:
      - name: gpt-4
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Providers[0].APIKey != "default-key" {
		t.Errorf("APIKey = %q, want default-key", cfg.Providers[0].APIKey)
	}
}

func TestParseAuthSettings(t *testing.T) {
	yaml := `
providers:
  - name: chatgpt-auth
    type: chatgpt-plus
    base_url: "https://chatgpt.com"
    session_cookie: "legacy-cookie"
    auth:
      mode: oauth
      client_id: "test-client-id"
      redirect_uri: "http://localhost:11437/callback"
      callback_port: 11437
      keychain_service: "ollarouter-test"
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	p := cfg.Providers[0]
	if p.SessionCookie != "legacy-cookie" {
		t.Errorf("SessionCookie = %q, want legacy-cookie", p.SessionCookie)
	}
	if p.Auth == nil {
		t.Fatal("Auth settings should not be nil")
	}
	if p.Auth.Mode != "oauth" {
		t.Errorf("Auth.Mode = %q, want oauth", p.Auth.Mode)
	}
	if p.Auth.ClientID != "test-client-id" {
		t.Errorf("Auth.ClientID = %q, want test-client-id", p.Auth.ClientID)
	}
	if p.Auth.RedirectURI != "http://localhost:11437/callback" {
		t.Errorf("Auth.RedirectURI = %q, want http://localhost:11437/callback", p.Auth.RedirectURI)
	}
	if p.Auth.CallbackPort != 11437 {
		t.Errorf("Auth.CallbackPort = %d, want 11437", p.Auth.CallbackPort)
	}
	if p.Auth.KeychainService != "ollarouter-test" {
		t.Errorf("Auth.KeychainService = %q, want ollarouter-test", p.Auth.KeychainService)
	}
}

func TestParseNoAuthSettings(t *testing.T) {
	yaml := `
providers:
  - name: test-openai
    type: openai
    api_key: "sk-test"
    models:
      - name: gpt-4o
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Providers[0].Auth != nil {
		t.Errorf("Auth should be nil when not specified, got %+v", cfg.Providers[0].Auth)
	}
}

func TestParseDuplicateProviders(t *testing.T) {
	yaml := `
providers:
  - name: dup
    type: openai
    models:
      - name: gpt-4
  - name: dup
    type: anthropic
    models:
      - name: claude
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("Expected error for duplicate providers, got nil")
	}
}

func TestParseDefaults(t *testing.T) {
	yaml := `providers: []`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Settings.Port != 11435 {
		t.Errorf("Default port = %d, want 11435", cfg.Settings.Port)
	}
	if cfg.Settings.LogLevel != "info" {
		t.Errorf("Default log level = %s, want info", cfg.Settings.LogLevel)
	}
}

func TestParseVirtualModels(t *testing.T) {
	yaml := `
virtual_models:
  - name: fast-chat
    fallback_chain:
      - openai/gpt-4o-mini
      - anthropic/claude-sonnet
    timeout: 30
providers:
  - name: test
    type: openai
    models:
      - name: gpt-4o-mini
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(cfg.VirtualModels) != 1 {
		t.Fatalf("len(VirtualModels) = %d, want 1", len(cfg.VirtualModels))
	}
	if cfg.VirtualModels[0].Name != "fast-chat" {
		t.Errorf("VirtualModel name = %q, want fast-chat", cfg.VirtualModels[0].Name)
	}
	if len(cfg.VirtualModels[0].FallbackChain) != 2 {
		t.Errorf("FallbackChain length = %d, want 2", len(cfg.VirtualModels[0].FallbackChain))
	}
}
