package streaming

import (
	"io"
	"net/http"
)

// OllamaNDJSONWriter writes Ollama-format NDJSON to an http.ResponseWriter.
type OllamaNDJSONWriter struct {
	*NDJSONWriter
	httpWriter http.ResponseWriter
}

// NewOllamaWriter creates an NDJSON writer for HTTP responses.
func NewOllamaWriter(w http.ResponseWriter, model string) *OllamaNDJSONWriter {
	flusher, _ := w.(interface{ Flush() })

	w.Header().Set("Content-Type", "application/x-ndjson")
	if flusher != nil {
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
	}

	return &OllamaNDJSONWriter{
		NDJSONWriter: NewNDJSONWriter(w, flusher, model),
		httpWriter:   w,
	}
}

// NewOllamaWriterWithWriter creates an NDJSON writer for a generic io.Writer.
func NewOllamaWriterWithWriter(w io.Writer, model string) *NDJSONWriter {
	flusher, _ := w.(httpFlusher)
	return NewNDJSONWriter(w, flusher, model)
}
