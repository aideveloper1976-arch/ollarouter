package guardrails

import (
	"testing"
)

func TestRescue_XMLInvokeBlock(t *testing.T) {
	input := `<invoke name="get_weather"><parameter name="location">London</parameter></invoke>`
	candidates := RescueToolCalls(input)
	if len(candidates) == 0 {
		t.Fatal("RescueToolCalls() returned 0 candidates for XML invoke block")
	}
	if candidates[0].Name != "get_weather" {
		t.Errorf("Name = %q, want get_weather", candidates[0].Name)
	}
}

func TestRescue_XMLInvokeWithText(t *testing.T) {
	input := `Let me check the weather for you.

<invoke name="get_weather">
<parameter name="location">Paris</parameter>
</invoke>`
	candidates := RescueToolCalls(input)
	if len(candidates) == 0 {
		t.Fatal("RescueToolCalls() returned 0 candidates for XML invoke with text")
	}
	if candidates[0].Name != "get_weather" {
		t.Errorf("Name = %q, want get_weather", candidates[0].Name)
	}
}

func TestRescue_FencedJSONBlock(t *testing.T) {
	input := "```json\n{\"name\": \"search\", \"arguments\": {\"query\": \"test\"}}\n```"
	candidates := RescueToolCalls(input)
	if len(candidates) == 0 {
		t.Fatal("RescueToolCalls() returned 0 candidates for fenced JSON")
	}
	if candidates[0].Name != "search" {
		t.Errorf("Name = %q, want search", candidates[0].Name)
	}
}

func TestRescue_RawJSONObject(t *testing.T) {
	input := `{"name": "calculate", "arguments": {"a": 1, "b": 2}}`
	candidates := RescueToolCalls(input)
	if len(candidates) == 0 {
		t.Fatal("RescueToolCalls() returned 0 candidates for raw JSON")
	}
	if candidates[0].Name != "calculate" {
		t.Errorf("Name = %q, want calculate", candidates[0].Name)
	}
}

func TestRescue_NoMatchReturnsEmpty(t *testing.T) {
	input := "Just a regular response without any tool calls."
	candidates := RescueToolCalls(input)
	if len(candidates) != 0 {
		t.Errorf("RescueToolCalls() = %v, want empty for no match", candidates)
	}
}

func TestRescue_MultipleXMLInvokes(t *testing.T) {
	input := `<invoke name="search"><parameter name="q">hello</parameter></invoke>
<invoke name="calculate"><parameter name="x">1</parameter><parameter name="y">2</parameter></invoke>`
	candidates := RescueToolCalls(input)
	if len(candidates) != 2 {
		t.Fatalf("RescueToolCalls() = %d candidates, want 2", len(candidates))
	}
	if candidates[0].Name != "search" {
		t.Errorf("candidates[0].Name = %q, want search", candidates[0].Name)
	}
	if candidates[1].Name != "calculate" {
		t.Errorf("candidates[1].Name = %q, want calculate", candidates[1].Name)
	}
}

func TestRescue_ExtractContentWithoutToolCalls(t *testing.T) {
	input := "Here is some text\n\n<invoke name=\"test\"><parameter name=\"x\">1</parameter></invoke>\n\nAnd more text"
	content := ExtractContent(input)
	// The invoke block and its surrounding whitespace are removed.
	// Text before and after are joined cleanly.
	if content != "Here is some textAnd more text" {
		t.Errorf("ExtractContent() = %q, want %q", content, "Here is some textAnd more text")
	}
}

func TestRescue_ExtractContentNoTools(t *testing.T) {
	input := "Just regular text"
	content := ExtractContent(input)
	if content != "Just regular text" {
		t.Errorf("ExtractContent() = %q, want 'Just regular text'", content)
	}
}
