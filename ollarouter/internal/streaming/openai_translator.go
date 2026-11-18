package streaming

import (
	"encoding/json"
	"strings"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// OpenAITranslator translates OpenAI SSE events to Ollama NDJSON chunks.
type OpenAITranslator struct{}

func NewOpenAITranslator() *OpenAITranslator {
	return &OpenAITranslator{}
}

func (t *OpenAITranslator) ParseSSELine(line string) (*types.StreamChunk, error) {
	if !strings.HasPrefix(line, "data: ") {
		return nil, nil
	}

	data := strings.TrimPrefix(line, "data: ")

	if t.IsDone(data) {
		return &types.StreamChunk{Done: true}, nil
	}

	var event struct {
		Choices []struct {
			Delta struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage,omitempty"`
	}

	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, err
	}

	chunk := &types.StreamChunk{Done: false}

	if len(event.Choices) > 0 {
		chunk.Content = event.Choices[0].Delta.Content
		if event.Choices[0].FinishReason != "" {
			chunk.Done = true
		}
	}

	if event.Usage != nil {
		chunk.TokenUsage = &types.TokenUsage{
			PromptTokens:     event.Usage.PromptTokens,
			CompletionTokens: event.Usage.CompletionTokens,
			TotalTokens:      event.Usage.PromptTokens + event.Usage.CompletionTokens,
		}
	}

	return chunk, nil
}

func (t *OpenAITranslator) ParseSSEEvent(event string, data string) (*types.StreamChunk, error) {
	return nil, nil
}

func (t *OpenAITranslator) IsDone(data string) bool {
	return data == "[DONE]"
}
