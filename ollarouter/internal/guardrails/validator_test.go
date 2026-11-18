package guardrails

import (
	"testing"
)

func TestValidate_ValidToolCallsPass(t *testing.T) {
	v := &Validator{}
	result := v.Validate("", []ToolCallCandidate{
		{Name: "get_weather", Arguments: `{"location":"London"}`, Raw: `{"name":"get_weather","arguments":{"location":"London"}}`},
	})
	if !result.Valid {
		t.Errorf("Validate() = %+v, want Valid=true", result)
	}
	if len(result.ToolCalls) != 1 {
		t.Errorf("len(ToolCalls) = %d, want 1", len(result.ToolCalls))
	}
}

func TestValidate_InvalidJSONArgumentsFails(t *testing.T) {
	v := &Validator{}
	result := v.Validate("", []ToolCallCandidate{
		{Name: "get_weather", Arguments: `{invalid json}`, Raw: `{"name":"get_weather","arguments":"{invalid json}"}`},
	})
	if result.Valid {
		t.Errorf("Validate() = %+v, want Valid=false for invalid JSON", result)
	}
}

func TestValidate_EmptyNameFails(t *testing.T) {
	v := &Validator{}
	result := v.Validate("", []ToolCallCandidate{
		{Name: "", Arguments: `{}`, Raw: `{"name":"","arguments":{}}`},
	})
	if result.Valid {
		t.Errorf("Validate() = Valid=true, want false for empty name")
	}
}

func TestValidate_MixedContentPassesWithPolicy(t *testing.T) {
	v := &Validator{}
	result := v.Validate("Here is the answer", []ToolCallCandidate{
		{Name: "search", Arguments: `{"query":"test"}`, Raw: `<invoke name="search"><parameter name="query">test</parameter></invoke>`},
	})
	if !result.Valid {
		t.Errorf("Validate() = %+v, want Valid=true for mixed content with valid tool calls", result)
	}
	if result.Content != "Here is the answer" {
		t.Errorf("Content = %q, want %q", result.Content, "Here is the answer")
	}
}

func TestValidate_EmptyToolCallsPasses(t *testing.T) {
	v := &Validator{}
	result := v.Validate("Just text", nil)
	if !result.Valid {
		t.Errorf("Validate() with no tools = %+v, want Valid=true", result)
	}
}

func TestValidate_EmptyToolCallsSlicePasses(t *testing.T) {
	v := &Validator{}
	result := v.Validate("Just text", []ToolCallCandidate{})
	if !result.Valid {
		t.Errorf("Validate() with empty tools = %+v, want Valid=true", result)
	}
}
