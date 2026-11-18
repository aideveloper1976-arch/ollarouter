package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

type Adapter struct {
	name       string
	baseURL    string
	httpClient *http.Client
	models     []types.ModelInfo
}

func NewAdapter(cfg types.ProviderConfig) (types.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://ollama:11434"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	a := &Adapter{
		name:    cfg.Name,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	// Try to discover models from running Ollama instance
	if cfg.AutoDiscover {
		models, err := a.discoverModels()
		if err == nil {
			a.models = models
		}
	}

	// Also add configured models
	for _, m := range cfg.Models {
		a.models = append(a.models, types.ModelInfo{
			Name:         cfg.Name + "/" + m.Name,
			ModifiedAt:   time.Now().UTC().Format(time.RFC3339),
			Size:         0,
			ProviderName: cfg.Name,
		})
	}

	return a, nil
}

func (a *Adapter) Name() string { return a.name }
func (a *Adapter) Type() string { return "ollama" }

func (a *Adapter) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	ollamaReq := ollamaChatRequest{
		Model:    req.Model,
		Messages: make([]ollamaMessage, len(req.Messages)),
		Stream:   false,
		Tools:    make([]ollamaTool, len(req.Tools)),
	}

	for i, t := range req.Tools {
		ollamaReq.Tools[i] = ollamaTool{
			Type: t.Type,
			Function: ollamaFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
	}
	if len(req.Tools) == 0 {
		ollamaReq.Tools = nil
	}

	for i, m := range req.Messages {
		msg := ollamaMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]ollamaToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				msg.ToolCalls[j] = ollamaToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: ollamaToolCallFunc{
						Name:      tc.Function.Name,
						Arguments: json.RawMessage(tc.Function.Arguments),
					},
				}
			}
		}
		ollamaReq.Messages[i] = msg
	}

	body, _ := json.Marshal(ollamaReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, a.handleError(resp)
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	respMsg := types.Message{
		Role:    ollamaResp.Message.Role,
		Content: ollamaResp.Message.Content,
	}
	if len(ollamaResp.Message.ToolCalls) > 0 {
		respMsg.ToolCalls = make([]types.ToolCall, len(ollamaResp.Message.ToolCalls))
		for j, tc := range ollamaResp.Message.ToolCalls {
			respMsg.ToolCalls[j] = types.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: string(tc.Function.Arguments),
				},
			}
		}
	}

	cr := &types.ChatResponse{
		Model:   req.Model,
		Message: respMsg,
	}

	return cr, nil
}

func (a *Adapter) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	ollamaReq := ollamaChatRequest{
		Model:    req.Model,
		Messages: make([]ollamaMessage, len(req.Messages)),
		Stream:   true,
		Tools:    make([]ollamaTool, len(req.Tools)),
	}

	for i, t := range req.Tools {
		ollamaReq.Tools[i] = ollamaTool{
			Type: t.Type,
			Function: ollamaFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
	}
	if len(req.Tools) == 0 {
		ollamaReq.Tools = nil
	}

	for i, m := range req.Messages {
		msg := ollamaMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]ollamaToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				msg.ToolCalls[j] = ollamaToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: ollamaToolCallFunc{
						Name:      tc.Function.Name,
						Arguments: json.RawMessage(tc.Function.Arguments),
					},
				}
			}
		}
		ollamaReq.Messages[i] = msg
	}

	body, _ := json.Marshal(ollamaReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", types.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return a.handleError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	doneSent := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk ollamaStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		sc := &types.StreamChunk{
			Content: chunk.Message.Content,
			Done:    chunk.Done,
		}

		if len(chunk.Message.ToolCalls) > 0 {
			sc.ToolCalls = make([]types.ToolCall, len(chunk.Message.ToolCalls))
			for j, tc := range chunk.Message.ToolCalls {
				sc.ToolCalls[j] = types.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Function.Name,
						Arguments: argsToString(tc.Function.Arguments),
					},
				}
			}
		}

		if chunk.Done {
			sc.Metrics = &types.Metrics{
				TotalDuration:   time.Duration(chunk.TotalDuration) * time.Nanosecond,
				PromptEvalCount: chunk.PromptEvalCount,
				EvalCount:       chunk.EvalCount,
				EvalDuration:    time.Duration(chunk.EvalDuration) * time.Nanosecond,
			}
		}

		if err := handler(sc); err != nil {
			return err
		}

		if chunk.Done {
			doneSent = true
			return nil
		}
	}

	if !doneSent {
		handler(&types.StreamChunk{Done: true})
	}

	return scanner.Err()
}

func (a *Adapter) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	ollamaReq := ollamaGenerateRequest{
		Model:  req.Model,
		Prompt: req.Prompt,
		System: req.System,
		Stream: false,
	}
	if req.Options != nil {
		ollamaReq.Options = &ollamaOptions{
			Temperature: req.Options.Temperature,
			TopP:        req.Options.TopP,
		}
	}

	body, _ := json.Marshal(ollamaReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, a.handleError(resp)
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &types.GenerateResponse{
		Model:    req.Model,
		Response: ollamaResp.Response,
	}, nil
}

func (a *Adapter) GenerateStream(ctx context.Context, req *types.GenerateRequest, handler types.StreamHandler) error {
	ollamaReq := ollamaGenerateRequest{
		Model:  req.Model,
		Prompt: req.Prompt,
		System: req.System,
		Stream: true,
	}

	body, _ := json.Marshal(ollamaReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", types.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return a.handleError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	doneSent := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk ollamaGenStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		sc := &types.StreamChunk{
			Content: chunk.Response,
			Done:    chunk.Done,
		}

		if err := handler(sc); err != nil {
			return err
		}

		if chunk.Done {
			doneSent = true
			return nil
		}
	}

	if !doneSent {
		handler(&types.StreamChunk{Done: true})
	}

	return scanner.Err()
}

func (a *Adapter) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	if len(a.models) > 0 {
		return a.models, nil
	}
	models, err := a.discoverModels()
	if err != nil {
		return nil, err
	}
	a.models = models
	return models, nil
}

func (a *Adapter) CheckHealth(ctx context.Context) types.ProviderHealth {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+"/api/tags", nil)
	if err != nil {
		return types.ProviderHealth{Status: "offline", LastCheck: time.Now(), ErrorMessage: err.Error()}
	}

	resp, err := a.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return types.ProviderHealth{Status: "offline", LastCheck: time.Now(), LatencyMS: latency, ErrorMessage: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return types.ProviderHealth{Status: "online", LastCheck: time.Now(), LatencyMS: latency}
	}
	return types.ProviderHealth{Status: "degraded", LastCheck: time.Now(), LatencyMS: latency, ErrorMessage: fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

func (a *Adapter) discoverModels() ([]types.ModelInfo, error) {
	resp, err := a.httpClient.Get(a.baseURL + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tagsResp ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, err
	}

	models := make([]types.ModelInfo, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = types.ModelInfo{
			Name:         a.name + "/" + m.Name,
			ModifiedAt:   m.ModifiedAt,
			Size:         m.Size,
			Digest:       m.Digest,
			ProviderName: a.name,
			Details: types.ModelDetails{
				Format:            m.Details.Format,
				Family:            m.Details.Family,
				ParameterSize:     m.Details.ParameterSize,
				QuantizationLevel: m.Details.QuantizationLevel,
			},
		}
	}

	return models, nil
}

func (a *Adapter) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("ollama error (HTTP %d): %s", resp.StatusCode, string(body))
}

// -- Ollama API types -----------------------------------------------------

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type ollamaTool struct {
	Type     string         `json:"type"`
	Function ollamaFunction `json:"function"`
}

type ollamaFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type ollamaToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function ollamaToolCallFunc `json:"function"`
}

type ollamaToolCallFunc struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// argsToString converts a json.RawMessage arguments field to a JSON string.
// Ollama may return arguments as either a JSON string or a JSON object.
func argsToString(r json.RawMessage) string {
	if len(r) == 0 {
		return ""
	}
	if r[0] == '"' {
		var s string
		json.Unmarshal(r, &s)
		return s
	}
	return string(r)
}

type ollamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

type ollamaStreamChunk struct {
	Model           string        `json:"model"`
	CreatedAt       string        `json:"created_at"`
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	TotalDuration   int64         `json:"total_duration,omitempty"`
	LoadDuration    int64         `json:"load_duration,omitempty"`
	PromptEvalCount int           `json:"prompt_eval_count,omitempty"`
	EvalCount       int           `json:"eval_count,omitempty"`
	EvalDuration    int64         `json:"eval_duration,omitempty"`
}

type ollamaGenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	System  string         `json:"system,omitempty"`
	Stream  bool           `json:"stream"`
	Options *ollamaOptions `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	CreatedAt string `json:"created_at"`
	Done      bool   `json:"done"`
}

type ollamaGenStreamChunk struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type ollamaOptions struct {
	Temperature float32 `json:"temperature,omitempty"`
	TopP        float32 `json:"top_p,omitempty"`
}

type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name       string            `json:"name"`
	ModifiedAt string            `json:"modified_at"`
	Size       int64             `json:"size"`
	Digest     string            `json:"digest"`
	Details    ollamaModelDetail `json:"details"`
}

type ollamaModelDetail struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families,omitempty"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}
