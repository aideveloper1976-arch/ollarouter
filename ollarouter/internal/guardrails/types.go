package guardrails

import "errors"

// Sentinel errors for tool call validation.
var (
	ErrInvalidToolCall = errors.New("invalid tool call output")
)

// Config holds guardrails configuration for a provider.
type Config struct {
	Enabled            bool   // Master switch for guardrails
	ToolCallValidation bool   // Validate tool call structure and arguments
	RescueMalformed    bool   // Attempt to rescue malformed tool calls
	MixedOutputPolicy  string // "tool_call_first" (default) or "text_first"
	MaxRetries         int    // Max retries for invalid tool outputs (default 3)
}

// ValidationResult holds the result of tool call validation.
type ValidationResult struct {
	Valid     bool
	ToolCalls []ToolCallCandidate
	Content   string
	Error     string
}

// ToolCallCandidate holds a parsed tool call candidate from rescue parsing.
type ToolCallCandidate struct {
	Name      string // Function/tool name
	Arguments string // JSON-encoded arguments
	Raw       string // Original raw text
}

// DefaultConfig returns default guardrails config for a provider type.
// Strong providers (openai, anthropic, google, chatgpt-plus) are disabled by default.
// Local/weak providers (ollama, openai-compatible) have guardrails enabled by default.
func DefaultConfig(providerType string) Config {
	cfg := Config{
		Enabled:            false,
		ToolCallValidation: true,
		RescueMalformed:    true,
		MixedOutputPolicy:  "tool_call_first",
		MaxRetries:         3,
	}
	if IsLocalOrWeakProvider(providerType) {
		cfg.Enabled = true
	}
	return cfg
}

// IsLocalOrWeakProvider returns true if the provider type is considered local or weak
// and should have guardrails enabled by default.
func IsLocalOrWeakProvider(providerType string) bool {
	switch providerType {
	case "ollama", "openai-compatible":
		return true
	default:
		return false
	}
}
