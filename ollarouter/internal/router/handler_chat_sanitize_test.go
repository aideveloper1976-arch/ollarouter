package router

import "testing"

func TestSanitizeAssistantContent_RemovesLeakageAndKeepsAnswer(t *testing.T) {
	raw := "Considering tool usage\nI’m thinking about how to proceed\nFinal answer here"
	got := sanitizeAssistantContent(raw)
	if got != "Final answer here" {
		t.Fatalf("expected final answer only, got %q", got)
	}
}

func TestSanitizeAssistantContent_RemovesDSMLTags(t *testing.T) {
	raw := "<｜｜DSML｜｜tool_calls>\n<｜｜DSML｜｜invoke name=\"run_terminal\">\nReal output"
	got := sanitizeAssistantContent(raw)
	if got != "Real output" {
		t.Fatalf("expected dsml removed, got %q", got)
	}
}

func TestSanitizeStreamingDelta_HoldsPartialThenFlushes(t *testing.T) {
	state := ""
	if got := sanitizeStreamingDelta(&state, "Considering tool", false); got != "" {
		t.Fatalf("expected no output for partial line, got %q", got)
	}
	if got := sanitizeStreamingDelta(&state, " usage\n", false); got != "" {
		t.Fatalf("expected leakage line dropped, got %q", got)
	}
	if got := sanitizeStreamingDelta(&state, "Real answer", false); got != "" {
		t.Fatalf("expected buffered line until flush, got %q", got)
	}
	if got := sanitizeStreamingDelta(&state, "", true); got != "Real answer" {
		t.Fatalf("expected flushed answer, got %q", got)
	}
}

func TestSanitizeAssistantContent_RemovesToolTagBlocks(t *testing.T) {
	raw := "How is UI testing done using the browser\n\n<read_file>\n<file>c:/repo/docs</file>\n</read_file>\n<semantic_search>\n<query>browser ui testing</query>\n</semantic_search>"
	got := sanitizeAssistantContent(raw)
	if got != "How is UI testing done using the browser" {
		t.Fatalf("expected tool tags removed, got %q", got)
	}
}

func TestSanitizeAssistantContent_RemovesSystemReminderBlock(t *testing.T) {
	raw := "<system-reminder>\nPlan mode ACTIVE\nYou are in read-only mode\n</system-reminder>\nFinal answer"
	got := sanitizeAssistantContent(raw)
	if got != "Final answer" {
		t.Fatalf("expected reminder block removed, got %q", got)
	}
}
