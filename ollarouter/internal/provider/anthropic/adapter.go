package anthropic

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
	apiKey     string
	version    string
	httpClient *http.Client
	models     []types.ModelInfo
}

func NewAdapter(cfg types.ProviderConfig) (types.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	version := cfg.AnthropicVersion
	if version == "" {
		version = "2023-06-01"
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	a := &Adapter{
		name:    cfg.Name,
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		version: version,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	for _, m := range cfg.Models {
		a.models = append(a.models, types.ModelInfo{
			Name:         cfg.Name + "/" + m.Name,
			Size:         0,
			ProviderName: cfg.Name,
		})
	}

	return a, nil
}

func (a *Adapter) Name() string { return a.name }
func (a *Adapter) Type() string { return "anthropic" }

func (a *Adapter) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	anthReq := a.buildRequest(req)
	body, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, a.handleError(resp)
	}

	var anthResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return a.translateResponse(&anthResp), nil
}

func (a *Adapter) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	anthReq := a.buildRequest(req)
	body, err := json.Marshal(anthReq)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

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

	currentContent := ""
	inputTokens := 0
	doneSent := false
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "message_start":
				if event.Message != nil && event.Message.Usage != nil {
					inputTokens = event.Message.Usage.InputTokens
				}
			case "content_block_start":
				if event.ContentBlock != nil && event.ContentBlock.Text != "" {
					currentContent += event.ContentBlock.Text
					if err := handler(&types.StreamChunk{
						Content: event.ContentBlock.Text,
						Done:    false,
					}); err != nil {
						return err
					}
				}
			case "content_block_delta":
				if event.Delta != nil && event.Delta.Text != "" {
					currentContent += event.Delta.Text
					if err := handler(&types.StreamChunk{
						Content: event.Delta.Text,
						Done:    false,
					}); err != nil {
						return err
					}
				}
			case "message_delta":
				usage := &types.TokenUsage{}
				usage.PromptTokens = inputTokens
				if event.Usage != nil {
					usage.CompletionTokens = event.Usage.OutputTokens
				}
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				doneSent = true
				if err := handler(&types.StreamChunk{
					Done:       true,
					TokenUsage: usage,
				}); err != nil {
					return err
				}
				return nil
			}
		}
	}

	if !doneSent {
		handler(&types.StreamChunk{Done: true})
	}

	return scanner.Err()
}

func (a *Adapter) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	chatReq := &types.ChatRequest{
		Model: req.Model,
		Messages: []types.Message{
			{Role: "user", Content: req.Prompt},
		},
		Options: req.Options,
	}
	if req.System != "" {
		chatReq.Messages = append([]types.Message{{Role: "system", Content: req.System}}, chatReq.Messages...)
	}

	chatResp, err := a.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	return &types.GenerateResponse{
		Model:      chatResp.Model,
		Response:   chatResp.Message.Content,
		TokenUsage: chatResp.TokenUsage,
	}, nil
}

func (a *Adapter) GenerateStream(ctx context.Context, req *types.GenerateRequest, handler types.StreamHandler) error {
	chatReq := &types.ChatRequest{
		Model: req.Model,
		Messages: []types.Message{
			{Role: "user", Content: req.Prompt},
		},
		Options: req.Options,
		Stream:  true,
	}
	if req.System != "" {
		chatReq.Messages = append([]types.Message{{Role: "system", Content: req.System}}, chatReq.Messages...)
	}
	return a.ChatStream(ctx, chatReq, handler)
}

func (a *Adapter) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	if len(a.models) > 0 {
		return a.models, nil
	}

	models, err := a.discoverModels(ctx)
	if err != nil {
		return nil, err
	}
	a.models = models
	return models, nil
}

func (a *Adapter) discoverModels(ctx context.Context) ([]types.ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("anthropic models API returned HTTP %d", resp.StatusCode)
	}

	var raw struct {
		Models json.RawMessage `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var modelEntries []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw.Models, &modelEntries); err == nil {
		models := make([]types.ModelInfo, len(modelEntries))
		for i, m := range modelEntries {
			models[i] = types.ModelInfo{
				Name:         a.name + "/" + m.ID,
				Size:         0,
				ProviderName: a.name,
			}
		}
		return models, nil
	}

	var modelNames []string
	if err := json.Unmarshal(raw.Models, &modelNames); err == nil {
		models := make([]types.ModelInfo, len(modelNames))
		for i, name := range modelNames {
			models[i] = types.ModelInfo{
				Name:         a.name + "/" + name,
				Size:         0,
				ProviderName: a.name,
			}
		}
		return models, nil
	}

	return nil, fmt.Errorf("unexpected anthropic models response")
}

func (a *Adapter) CheckHealth(ctx context.Context) types.ProviderHealth {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+"/models", nil)
	if err != nil {
		return types.ProviderHealth{Status: "offline", LastCheck: time.Now(), ErrorMessage: err.Error()}
	}
	a.setHeaders(req)

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

func (a *Adapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", a.version)
}

func (a *Adapter) buildRequest(req *types.ChatRequest) anthropicRequest {
	// Extract system message
	var system string
	messages := make([]anthropicMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		messages = append(messages, anthropicMessage{Role: m.Role, Content: m.Content})
	}

	anthReq := anthropicRequest{
		Model:     req.Model,
		Messages:  messages,
		MaxTokens: 2048,
		Stream:    req.Stream,
	}

	if system != "" {
		anthReq.System = system
	}
	if req.Options != nil {
		if req.Options.MaxTokens > 0 {
			anthReq.MaxTokens = req.Options.MaxTokens
		}
		if req.Options.Temperature > 0 {
			anthReq.Temperature = req.Options.Temperature
		}
		if req.Options.TopP > 0 {
			anthReq.TopP = req.Options.TopP
		}
		anthReq.StopSequences = req.Options.Stop
	}

	return anthReq
}

func (a *Adapter) translateResponse(resp *anthropicResponse) *types.ChatResponse {
	cr := &types.ChatResponse{
		Model: resp.Model,
		Message: types.Message{
			Role:    "assistant",
			Content: "",
		},
	}
	for _, block := range resp.Content {
		if block.Type == "text" {
			cr.Message.Content += block.Text
		}
	}
	if resp.Usage != nil {
		cr.TokenUsage = &types.TokenUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		}
	}
	return cr
}

func (a *Adapter) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp anthropicErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		switch resp.StatusCode {
		case 401:
			return fmt.Errorf("%w: %s", types.ErrProviderAuthFailed, errResp.Error.Message)
		case 429:
			return fmt.Errorf("%w: %s", types.ErrProviderRateLimited, errResp.Error.Message)
		default:
			return fmt.Errorf("anthropic error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
	}
	return fmt.Errorf("anthropic error (HTTP %d): %s", resp.StatusCode, string(body))
}

// -- Anthropic API types ----------------------------------------------------

type anthropicRequest struct {
	Model         string             `json:"model"`
	Messages      []anthropicMessage `json:"messages"`
	System        string             `json:"system,omitempty"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   float32            `json:"temperature,omitempty"`
	TopP          float32            `json:"top_p,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Role    string             `json:"role"`
	Model   string             `json:"model"`
	Content []anthropicContent `json:"content"`
	Usage   *anthropicUsage    `json:"usage,omitempty"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicStreamEvent struct {
	Type         string                 `json:"type"`
	Message      *anthropicResponse     `json:"message,omitempty"`
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
	Delta        *anthropicDelta        `json:"delta,omitempty"`
	Usage        *anthropicUsage        `json:"usage,omitempty"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicDelta struct {
	Type         string `json:"type"`
	Text         string `json:"text"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
}

type anthropicErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
