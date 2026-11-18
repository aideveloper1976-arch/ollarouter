package config

import "fmt"

// Config is the root configuration structure for OllaRouter.
type Config struct {
	Settings      Settings             `yaml:"settings"`
	APIKeys       []APIKeyConfig       `yaml:"api_keys"`
	Providers     []ProviderConfig     `yaml:"providers"`
	VirtualModels []VirtualModelConfig `yaml:"virtual_models"`
}

// Settings holds global OllaRouter settings.
type Settings struct {
	Port                       int    `yaml:"port" default:"11435"`
	AdminPort                  int    `yaml:"admin_port" default:"11436"`
	LogLevel                   string `yaml:"log_level" default:"info"`
	LogFormat                  string `yaml:"log_format" default:"json"`
	PrometheusEnabled          bool   `yaml:"prometheus_enabled" default:"true"`
	RateLimitingEnabled        bool   `yaml:"rate_limiting_enabled" default:"true"`
	DefaultMaxRetries          int    `yaml:"default_max_retries" default:"3"`
	DefaultTimeoutSec          int    `yaml:"default_timeout_seconds" default:"30"`
	UsagePersistence           string `yaml:"usage_persistence" default:"memory"`
	UsageFilePath              string `yaml:"usage_file_path" default:"/data/usage.json"`
	CircuitBreakerThreshold    int    `yaml:"circuit_breaker_threshold" default:"5"`
	CircuitBreakerCooldown     int    `yaml:"circuit_breaker_cooldown_seconds" default:"30"`
	UpstreamAuthCredentialPath string `yaml:"upstream_auth_credential_path" default:"/data/upstream-auth.enc"`
	UpstreamAuthEncryptionKey  string `yaml:"upstream_auth_encryption_key"`
	// OpenAIEndpointPublic leaves /v1/chat/completions open even when
	// api_keys are configured. Only for clients that cannot send a key.
	OpenAIEndpointPublic bool `yaml:"openai_endpoint_public,omitempty"`
}

// APIKeyConfig defines a client API key.
type APIKeyConfig struct {
	Name          string  `yaml:"name"`
	Key           string  `yaml:"key"`
	RateLimit     int     `yaml:"rate_limit"`
	MonthlyBudget float64 `yaml:"monthly_budget_usd"`
}

// ProviderConfig defines a single provider configuration.
type ProviderConfig struct {
	Name             string            `yaml:"name"`
	Type             string            `yaml:"type"`
	BaseURL          string            `yaml:"base_url"`
	APIKey           string            `yaml:"api_key"`
	OrganizationID   string            `yaml:"organization_id,omitempty"`
	AnthropicVersion string            `yaml:"anthropic_version,omitempty"`
	AutoDiscover     bool              `yaml:"auto_discover,omitempty"`
	PollingInterval  int               `yaml:"polling_interval_seconds,omitempty"`
	Models           []ModelConfig     `yaml:"models,omitempty"`
	Pricing          *PricingConfig    `yaml:"pricing,omitempty"`
	MaxRetries       int               `yaml:"max_retries,omitempty"`
	TimeoutSec       int               `yaml:"timeout_seconds,omitempty"`
	Headers          map[string]string `yaml:"headers,omitempty"`
	// Tool call guardrails for local/weak providers
	Guardrails *GuardrailsConfig `yaml:"guardrails,omitempty"`
	// ChatGPT Plus specific
	SessionCookie string        `yaml:"session_cookie,omitempty"`
	UserAgent     string        `yaml:"user_agent,omitempty"`
	Auth          *AuthSettings `yaml:"auth,omitempty"`
}

// GuardrailsConfig defines tool call validation and rescue settings for a provider.
type GuardrailsConfig struct {
	Enabled            bool   `yaml:"enabled"`
	ToolCallValidation bool   `yaml:"tool_call_validation"`
	RescueMalformed    bool   `yaml:"rescue_malformed_tool_calls"`
	MixedOutputPolicy  string `yaml:"mixed_output_policy"`
	MaxRetries         int    `yaml:"max_retries"`
}

// AuthSettings defines the upstream authentication configuration for a provider.
type AuthSettings struct {
	Mode            string `yaml:"mode"`                       // "oauth" or "session_cookie" or "none"
	ClientID        string `yaml:"client_id,omitempty"`        // OAuth client ID
	ClientSecret    string `yaml:"client_secret,omitempty"`    // OAuth client secret
	RedirectURI     string `yaml:"redirect_uri,omitempty"`     // OAuth redirect URI
	CallbackPort    int    `yaml:"callback_port,omitempty"`    // Local callback server port
	KeychainService string `yaml:"keychain_service,omitempty"` // OS keychain service name
	TokenEndpoint   string `yaml:"token_endpoint,omitempty"`   // Custom token endpoint
	AuthEndpoint    string `yaml:"auth_endpoint,omitempty"`    // Custom auth endpoint
	Scopes          string `yaml:"scopes,omitempty"`           // OAuth scopes
}

// ModelConfig defines a single model within a provider.
type ModelConfig struct {
	Name       string `yaml:"name"`
	MaxTokens  int    `yaml:"max_tokens,omitempty"`
	ContextLen int    `yaml:"context_length,omitempty"`
}

// PricingConfig defines per-token pricing.
type PricingConfig struct {
	InputPer1K  float64 `yaml:"input_per_1k"`
	OutputPer1K float64 `yaml:"output_per_1k"`
}

// VirtualModelConfig defines a virtual model with fallback chain.
type VirtualModelConfig struct {
	Name          string   `yaml:"name"`
	FallbackChain []string `yaml:"fallback_chain"`
	Timeout       int      `yaml:"timeout,omitempty"`
	MaxTokens     int      `yaml:"max_tokens,omitempty"`
}

// DefaultSettings returns the default settings.
func DefaultSettings() Settings {
	return Settings{
		Port:                       11435,
		AdminPort:                  11436,
		LogLevel:                   "info",
		LogFormat:                  "json",
		PrometheusEnabled:          true,
		RateLimitingEnabled:        true,
		DefaultMaxRetries:          3,
		DefaultTimeoutSec:          30,
		UsagePersistence:           "memory",
		UsageFilePath:              "/data/usage.json",
		CircuitBreakerThreshold:    5,
		CircuitBreakerCooldown:     30,
		UpstreamAuthCredentialPath: "/data/upstream-auth.enc",
		UpstreamAuthEncryptionKey:  "",
	}
}

// Validate checks the configuration for common errors.
func (c *Config) Validate() error {
	if c.Settings.Port <= 0 || c.Settings.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Settings.Port)
	}
	seen := make(map[string]bool)
	for _, p := range c.Providers {
		if seen[p.Name] {
			return fmt.Errorf("duplicate provider name: %s", p.Name)
		}
		seen[p.Name] = true
		if p.Name == "" {
			return fmt.Errorf("provider name is required")
		}
		if p.Type == "" {
			return fmt.Errorf("provider type is required for %s", p.Name)
		}
	}
	seenVM := make(map[string]bool)
	for _, v := range c.VirtualModels {
		if seenVM[v.Name] {
			return fmt.Errorf("duplicate virtual model name: %s", v.Name)
		}
		seenVM[v.Name] = true
	}
	return nil
}
