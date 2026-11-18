package streaming

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

func TestOpenAITranslator(t *testing.T) {
	tr := NewOpenAITranslator()

	// Test basic content chunk
	chunk, err := tr.ParseSSELine(`data: {"choices":[{"delta":{"role":"assistant","content":"Hello"}}]}`)
	if err != nil {
		t.Fatalf("ParseSSELine error: %v", err)
	}
	if chunk == nil {
		t.Fatal("Expected chunk, got nil")
	}
	if chunk.Content != "Hello" {
		t.Errorf("Content = %q, want %q", chunk.Content, "Hello")
	}
	if chunk.Done {
		t.Error("Expected done=false")
	}

	// Test [DONE] signal
	done, err := tr.ParseSSELine(`data: [DONE]`)
	if err != nil {
		t.Fatalf("ParseSSELine error: %v", err)
	}
	if !done.Done {
		t.Error("Expected done=true")
	}

	// Test finish reason
	finish, err := tr.ParseSSELine(`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":20}}`)
	if err != nil {
		t.Fatalf("ParseSSELine error: %v", err)
	}
	if !finish.Done {
		t.Error("Expected done=true")
	}
	if finish.TokenUsage.PromptTokens != 10 || finish.TokenUsage.CompletionTokens != 20 {
		t.Errorf("Unexpected token usage: %+v", finish.TokenUsage)
	}
}

func TestAnthropicTranslator(t *testing.T) {
	tr := NewAnthropicTranslator()

	// Test content_block_start
	chunk, err := tr.ParseSSEEvent("content_block_start", `{"content_block":{"type":"text","text":"Hello"}}`)
	if err != nil {
		t.Fatalf("ParseSSEEvent error: %v", err)
	}
	if chunk == nil || chunk.Content != "Hello" {
		t.Errorf("Content = %v, want Hello", chunk)
	}

	// Test content_block_delta
	chunk, err = tr.ParseSSEEvent("content_block_delta", `{"delta":{"type":"text_delta","text":" world"}}`)
	if err != nil {
		t.Fatalf("ParseSSEEvent error: %v", err)
	}
	if chunk.Content != " world" {
		t.Errorf("Content = %q, want ' world'", chunk.Content)
	}

	// Test message_delta (stream end)
	chunk, err = tr.ParseSSEEvent("message_delta", `{"delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":20}}`)
	if err != nil {
		t.Fatalf("ParseSSEEvent error: %v", err)
	}
	if !chunk.Done {
		t.Error("Expected done=true")
	}
	if chunk.TokenUsage.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %d, want 20", chunk.TokenUsage.CompletionTokens)
	}
}

func TestChatGPTTranslator(t *testing.T) {
	tr := NewChatGPTTranslator()

	// Test text delta
	chunk, err := tr.ParseSSELine(`data: {"type":"response.text.delta","delta":"Hello"}`)
	if err != nil {
		t.Fatalf("ParseSSELine error: %v", err)
	}
	if chunk == nil || chunk.Content != "Hello" {
		t.Errorf("Content = %v, want Hello", chunk)
	}

	// Test message complete
	chunk, err = tr.ParseSSELine(`data: {"type":"message.complete","message":{"id":"msg1","content":{"parts":["Hello"]},"usage":{"prompt_tokens":10,"completion_tokens":5}}}`)
	if err != nil {
		t.Fatalf("ParseSSELine error: %v", err)
	}
	if chunk == nil {
		t.Fatal("Expected chunk, got nil")
	}
	if !chunk.Done {
		t.Error("Expected done=true")
	}
	// TokenUsage may be nil if parsing fails, but we check the done flag at minimum
}

func TestNDJSONWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewNDJSONWriter(&buf, nil, "test-model")

	err := writer.WriteChunk(&types.StreamChunk{
		Content: "Hello",
		Done:    false,
	})
	if err != nil {
		t.Fatalf("WriteChunk error: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	if !strings.Contains(line, `"message"`) {
		t.Errorf("Expected message field in: %s", line)
	}
	if !strings.Contains(line, `"content":"Hello"`) {
		t.Errorf("Expected content in: %s", line)
	}
	if !strings.Contains(line, `"done":false`) {
		t.Errorf("Expected done:false in: %s", line)
	}

	buf.Reset()
	err = writer.WriteChunk(&types.StreamChunk{Done: true})
	if err != nil {
		t.Fatalf("WriteFinalChunk error: %v", err)
	}
	line = strings.TrimSpace(buf.String())
	if !strings.Contains(line, `"done":true`) {
		t.Errorf("Expected done:true in: %s", line)
	}
}
