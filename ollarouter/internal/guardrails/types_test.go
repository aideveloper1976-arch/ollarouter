package guardrails

import (
	"testing"
)

func TestDefaultConfig_Ollama(t *testing.T) {
	cfg := DefaultConfig("ollama")
	if !cfg.Enabled {
		t.Errorf("DefaultConfig(ollama).Enabled = false, want true (ollama is a local/weak provider)")
	}
}

func TestDefaultConfig_OpenAICompat(t *testing.T) {
	cfg := DefaultConfig("openai-compatible")
	if !cfg.Enabled {
		t.Errorf("DefaultConfig(openai-compatible).Enabled = false, want true (openai-compatible is a local/weak provider)")
	}
}

func TestDefaultConfig_UnknownTypeDisabled(t *testing.T) {
	cfg := DefaultConfig("openai")
	if cfg.Enabled {
		t.Errorf("DefaultConfig(openai).Enabled = true, want false for strong provider")
	}
}

func TestDefaultConfig_HasDefaultPolicy(t *testing.T) {
	cfg := DefaultConfig("ollama")
	if cfg.MixedOutputPolicy != "tool_call_first" {
		t.Errorf("DefaultConfig().MixedOutputPolicy = %q, want tool_call_first", cfg.MixedOutputPolicy)
	}
}

func TestDefaultConfig_DefaultMaxRetries(t *testing.T) {
	cfg := DefaultConfig("ollama")
	if cfg.MaxRetries != 3 {
		t.Errorf("DefaultConfig().MaxRetries = %d, want 3", cfg.MaxRetries)
	}
}

func TestDefaultConfig_DefaultToolCallValidation(t *testing.T) {
	cfg := DefaultConfig("ollama")
	if !cfg.ToolCallValidation {
		t.Errorf("DefaultConfig().ToolCallValidation = false, want true")
	}
}

func TestDefaultConfig_DefaultRescueMalformed(t *testing.T) {
	cfg := DefaultConfig("ollama")
	if !cfg.RescueMalformed {
		t.Errorf("DefaultConfig().RescueMalformed = false, want true")
	}
}

func TestIsLocalOrWeakProvider_Ollama(t *testing.T) {
	if !IsLocalOrWeakProvider("ollama") {
		t.Errorf("IsLocalOrWeakProvider(ollama) = false, want true")
	}
}

func TestIsLocalOrWeakProvider_OpenAICompat(t *testing.T) {
	if !IsLocalOrWeakProvider("openai-compatible") {
		t.Errorf("IsLocalOrWeakProvider(openai-compatible) = false, want true")
	}
}

func TestIsLocalOrWeakProvider_OpenAI(t *testing.T) {
	if IsLocalOrWeakProvider("openai") {
		t.Errorf("IsLocalOrWeakProvider(openai) = true, want false")
	}
}

func TestIsLocalOrWeakProvider_Anthropic(t *testing.T) {
	if IsLocalOrWeakProvider("anthropic") {
		t.Errorf("IsLocalOrWeakProvider(anthropic) = true, want false")
	}
}
