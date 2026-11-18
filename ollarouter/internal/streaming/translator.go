package streaming

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// StreamTranslator translates between provider SSE and Ollama NDJSON.
type StreamTranslator interface {
	// ParseSSELine parses a single SSE line and returns a stream chunk.
	ParseSSELine(line string) (*types.StreamChunk, error)

	// ParseSSEEvent parses named SSE events (for Anthropic-style events).
	ParseSSEEvent(event string, data string) (*types.StreamChunk, error)

	// IsDone returns true if this event signals the end of the stream.
	IsDone(data string) bool
}

// NDJSONWriter writes Ollama-format NDJSON stream chunks.
type NDJSONWriter struct {
	writer    io.Writer
	flusher   httpFlusher
	model     string
	createdAt string
}

type httpFlusher interface {
	Flush()
}

// NewNDJSONWriter creates a new NDJSON writer.
func NewNDJSONWriter(w io.Writer, flusher httpFlusher, model string) *NDJSONWriter {
	return &NDJSONWriter{
		writer:    w,
		flusher:   flusher,
		model:     model,
		createdAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// WriteChunk writes a single stream chunk as NDJSON.
func (w *NDJSONWriter) WriteChunk(chunk *types.StreamChunk) error {
	olChunk := ollamaNDJSONChunk{
		Model:     w.model,
		CreatedAt: w.createdAt,
		Done:      chunk.Done,
	}

	if !chunk.Done {
		olChunk.Message = &struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			Role:    "assistant",
			Content: chunk.Content,
		}
	}

	if chunk.Done && chunk.Metrics != nil {
		olChunk.TotalDuration = chunk.Metrics.TotalDuration.Nanoseconds()
		olChunk.LoadDuration = chunk.Metrics.LoadDuration.Nanoseconds()
		olChunk.PromptEvalCount = chunk.Metrics.PromptEvalCount
		olChunk.EvalCount = chunk.Metrics.EvalCount
		olChunk.EvalDuration = chunk.Metrics.EvalDuration.Nanoseconds()
	}

	if chunk.TokenUsage != nil {
		olChunk.PromptEvalCount = chunk.TokenUsage.PromptTokens
		olChunk.EvalCount = chunk.TokenUsage.CompletionTokens
	}

	data, err := json.Marshal(olChunk)
	if err != nil {
		return fmt.Errorf("marshaling chunk: %w", err)
	}

	if _, err := w.writer.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing chunk: %w", err)
	}

	if w.flusher != nil {
		w.flusher.Flush()
	}

	return nil
}

// WriteFinalChunk writes the final done:true chunk.
func (w *NDJSONWriter) WriteFinalChunk() error {
	return w.WriteChunk(&types.StreamChunk{Done: true})
}

// WriteError writes an error as NDJSON.
func (w *NDJSONWriter) WriteError(errMsg string) error {
	olChunk := ollamaNDJSONChunk{
		Model:     w.model,
		CreatedAt: w.createdAt,
		Done:      true,
		Error:     errMsg,
	}
	data, _ := json.Marshal(olChunk)
	if _, err := w.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	if w.flusher != nil {
		w.flusher.Flush()
	}
	return nil
}

// ollamaNDJSONChunk is the Ollama-compatible NDJSON streaming format.
type ollamaNDJSONChunk struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   *struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message,omitempty"`
	Done            bool   `json:"done"`
	TotalDuration   int64  `json:"total_duration,omitempty"`
	LoadDuration    int64  `json:"load_duration,omitempty"`
	PromptEvalCount int    `json:"prompt_eval_count,omitempty"`
	EvalCount       int    `json:"eval_count,omitempty"`
	EvalDuration    int64  `json:"eval_duration,omitempty"`
	Error           string `json:"error,omitempty"`
}
