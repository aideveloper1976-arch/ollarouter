package streaming

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// ChatGPTTranslator translates ChatGPT Plus SSE events to Ollama NDJSON chunks.
type ChatGPTTranslator struct {
	accumulatedContent string
}

func NewChatGPTTranslator() *ChatGPTTranslator {
	return &ChatGPTTranslator{}
}

func (t *ChatGPTTranslator) ParseSSELine(line string) (*types.StreamChunk, error) {
	if !strings.HasPrefix(line, "data: ") {
		return nil, nil
	}

	data := strings.TrimPrefix(line, "data: ")

	if t.IsDone(data) {
		return &types.StreamChunk{Done: true}, nil
	}

	var event struct {
		Type    string `json:"type"`
		Delta   string `json:"delta,omitempty"`
		Message *struct {
			ID      string `json:"id"`
			Content struct {
				Parts []string `json:"parts"`
			} `json:"content"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage,omitempty"`
		} `json:"message,omitempty"`
	}

	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, err
	}

	switch event.Type {
	case "response.text.delta":
		t.accumulatedContent += event.Delta
		return &types.StreamChunk{
			Content: event.Delta,
			Done:    false,
		}, nil

	case "response.text.done":
		// Text is finished, but the message may still be completing
		return nil, nil

	case "message.complete":
		chunk := &types.StreamChunk{Done: true}
		if event.Message != nil && event.Message.Usage != nil {
			chunk.TokenUsage = &types.TokenUsage{
				PromptTokens:     event.Message.Usage.PromptTokens,
				CompletionTokens: event.Message.Usage.CompletionTokens,
				TotalTokens:      event.Message.Usage.PromptTokens + event.Message.Usage.CompletionTokens,
			}
		}
		return chunk, nil

	case "error":
		return &types.StreamChunk{
			Done:  true,
			Error: fmt.Errorf("ChatGPT API error"),
		}, nil
	}

	return nil, nil
}

func (t *ChatGPTTranslator) ParseSSEEvent(event string, data string) (*types.StreamChunk, error) {
	return nil, nil
}

func (t *ChatGPTTranslator) IsDone(data string) bool {
	return strings.Contains(data, `"type":"message.complete"`)
}

// GetAccumulatedContent returns all content accumulated during streaming.
func (t *ChatGPTTranslator) GetAccumulatedContent() string {
	return t.accumulatedContent
}
