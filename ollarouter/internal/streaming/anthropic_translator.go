package streaming

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// AnthropicTranslator translates Anthropic SSE events to Ollama NDJSON chunks.
type AnthropicTranslator struct {
	currentContent string
	finished       bool
}

func NewAnthropicTranslator() *AnthropicTranslator {
	return &AnthropicTranslator{}
}

func (t *AnthropicTranslator) ParseSSELine(line string) (*types.StreamChunk, error) {
	return nil, nil
}

func (t *AnthropicTranslator) ParseSSEEvent(event string, data string) (*types.StreamChunk, error) {
	switch event {
	case "content_block_start":
		var block struct {
			ContentBlock *struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content_block"`
		}
		if err := json.Unmarshal([]byte(data), &block); err != nil {
			return nil, err
		}
		if block.ContentBlock != nil && block.ContentBlock.Text != "" {
			t.currentContent += block.ContentBlock.Text
			return &types.StreamChunk{
				Content: block.ContentBlock.Text,
				Done:    false,
			}, nil
		}

	case "content_block_delta":
		var delta struct {
			Delta *struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &delta); err != nil {
			return nil, err
		}
		if delta.Delta != nil && delta.Delta.Text != "" {
			t.currentContent += delta.Delta.Text
			return &types.StreamChunk{
				Content: delta.Delta.Text,
				Done:    false,
			}, nil
		}

	case "message_delta":
		var msgDelta struct {
			Usage *struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Delta *struct {
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &msgDelta); err != nil {
			return nil, err
		}
		usage := &types.TokenUsage{}
		if msgDelta.Usage != nil {
			usage.CompletionTokens = msgDelta.Usage.OutputTokens
			usage.TotalTokens = usage.CompletionTokens
		}
		t.finished = true
		return &types.StreamChunk{
			Done:       true,
			TokenUsage: usage,
		}, nil

	case "error":
		var errResp struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &errResp); err == nil {
			return &types.StreamChunk{
				Done:  true,
				Error: fmt.Errorf("%s", errResp.Error.Message),
			}, nil
		}
	}

	return nil, nil
}

func (t *AnthropicTranslator) IsDone(data string) bool {
	return strings.Contains(data, `"type":"message_delta"`)
}

// Reset clears the accumulated state.
func (t *AnthropicTranslator) Reset() {
	t.currentContent = ""
	t.finished = false
}
