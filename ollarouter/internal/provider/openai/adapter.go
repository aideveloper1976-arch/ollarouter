package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

type Adapter struct {
	name       string
	baseURL    string
	apiKey     string
	orgID      string
	httpClient *http.Client
	models     []types.ModelInfo
}

func NewAdapter(cfg types.ProviderConfig) (types.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	a := &Adapter{
		name:    cfg.Name,
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		orgID:   cfg.OrganizationID,
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

func (a *Adapter) Type() string { return "openai" }

func (a *Adapter) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	openAIReq := a.buildChatRequest(req)
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
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

	var openAIResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return a.translateChatResponse(&openAIResp), nil
}

func (a *Adapter) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	openAIReq := a.buildChatRequest(req)
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
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

	// Streamed tool calls arrive as fragments keyed by index; accumulate them
	// and emit complete calls once the upstream signals finish.
	acc := newToolCallAccumulator()

	doneSent := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			acc.flush(handler)
			doneSent = true
			handler(&types.StreamChunk{Done: true})
			return nil
		}

		var streamEvent openAIStreamEvent
		if err := json.Unmarshal([]byte(data), &streamEvent); err != nil {
			continue
		}

		if len(streamEvent.Choices) > 0 {
			d := streamEvent.Choices[0].Delta
			if len(d.ToolCalls) > 0 {
				acc.add(d.ToolCalls)
			} else if d.Content != "" {
				if err := handler(&types.StreamChunk{Content: d.Content, Done: false}); err != nil {
					return err
				}
			}
		}

		if len(streamEvent.Choices) > 0 && streamEvent.Choices[0].FinishReason != "" {
			acc.flush(handler)
			usage := streamEvent.Usage
			finalChunk := &types.StreamChunk{Done: true}
			if usage != nil {
				finalChunk.TokenUsage = &types.TokenUsage{
					PromptTokens:     usage.PromptTokens,
					CompletionTokens: usage.CompletionTokens,
					TotalTokens:      usage.PromptTokens + usage.CompletionTokens,
				}
			}
			doneSent = true
			handler(finalChunk)
			return nil
		}
	}

	if !doneSent {
		acc.flush(handler)
		handler(&types.StreamChunk{Done: true})
	}

	return scanner.Err()
}

func (a *Adapter) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	// Translate generate to chat format
	chatReq := &types.ChatRequest{
		Model: req.Model,
		Messages: []types.Message{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.Prompt},
		},
		Options: req.Options,
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
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.Prompt},
		},
		Options: req.Options,
		Stream:  true,
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
		return nil, fmt.Errorf("openai models API returned HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	models := make([]types.ModelInfo, len(payload.Data))
	for i, m := range payload.Data {
		models[i] = types.ModelInfo{
			Name:         a.name + "/" + m.ID,
			Size:         0,
			ProviderName: a.name,
		}
	}
	return models, nil
}

func (a *Adapter) CheckHealth(ctx context.Context) types.ProviderHealth {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+"/models", nil)
	if err != nil {
		return types.ProviderHealth{
			Status:       "offline",
			LastCheck:    time.Now(),
			ErrorMessage: err.Error(),
		}
	}
	a.setHeaders(req)

	resp, err := a.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return types.ProviderHealth{
			Status:       "offline",
			LastCheck:    time.Now(),
			LatencyMS:    latency,
			ErrorMessage: err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return types.ProviderHealth{
			Status:    "online",
			LastCheck: time.Now(),
			LatencyMS: latency,
		}
	}

	return types.ProviderHealth{
		Status:       "degraded",
		LastCheck:    time.Now(),
		LatencyMS:    latency,
		ErrorMessage: fmt.Sprintf("HTTP %d", resp.StatusCode),
	}
}

func (a *Adapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	if a.orgID != "" {
		req.Header.Set("OpenAI-Organization", a.orgID)
	}
}

func (a *Adapter) buildChatRequest(req *types.ChatRequest) openAIChatRequest {
	messages := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msg := openAIMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			otc := openAIToolCall{ID: tc.ID, Type: tc.Type}
			if otc.Type == "" {
				otc.Type = "function"
			}
			otc.Function.Name = tc.Function.Name
			otc.Function.Arguments = tc.Function.Arguments
			msg.ToolCalls = append(msg.ToolCalls, otc)
		}
		if len(m.Images) > 0 {
			content := []openAIContentPart{
				{Type: "text", Text: m.Content},
			}
			for _, img := range m.Images {
				content = append(content, openAIContentPart{
					Type: "image_url",
					ImageURL: &openAIImageURL{
						URL: "data:image/png;base64," + img,
					},
				})
			}
			msg.Content = ""
			msg.ContentParts = content
		}
		messages[i] = msg
	}

	openAIReq := openAIChatRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	for _, t := range req.Tools {
		ot := openAITool{
			Type: t.Type,
			Function: openAIToolFunc{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
		if ot.Type == "" {
			ot.Type = "function"
		}
		openAIReq.Tools = append(openAIReq.Tools, ot)
	}

	if req.Stream {
		openAIReq.StreamOptions = &openAIStreamOptions{IncludeUsage: true}
	}

	if req.Options != nil {
		openAIReq.Temperature = req.Options.Temperature
		openAIReq.TopP = req.Options.TopP
		openAIReq.MaxTokens = req.Options.MaxTokens
		openAIReq.Stop = req.Options.Stop
		openAIReq.FrequencyPenalty = req.Options.FrequencyPenalty
		openAIReq.PresencePenalty = req.Options.PresencePenalty
		if req.Options.Seed != 0 {
			openAIReq.Seed = &req.Options.Seed
		}
	}

	return openAIReq
}

func (a *Adapter) translateChatResponse(resp *openAIChatResponse) *types.ChatResponse {
	cr := &types.ChatResponse{
		Model: resp.Model,
	}
	if len(resp.Choices) > 0 {
		m := resp.Choices[0].Message
		cr.Message = types.Message{
			Role:    m.Role,
			Content: m.Content,
		}
		if cr.Message.Role == "" {
			cr.Message.Role = "assistant"
		}
		for _, tc := range m.ToolCalls {
			itc := types.ToolCall{ID: tc.ID, Type: tc.Type}
			if itc.Type == "" {
				itc.Type = "function"
			}
			itc.Function.Name = tc.Function.Name
			itc.Function.Arguments = tc.Function.Arguments
			cr.Message.ToolCalls = append(cr.Message.ToolCalls, itc)
		}
	}
	if resp.Usage != nil {
		cr.TokenUsage = &types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.PromptTokens + resp.Usage.CompletionTokens,
		}
	}
	return cr
}

func (a *Adapter) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp openAIErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		switch resp.StatusCode {
		case 401:
			return fmt.Errorf("%w: %s", types.ErrProviderAuthFailed, errResp.Error.Message)
		case 429:
			return fmt.Errorf("%w: %s", types.ErrProviderRateLimited, errResp.Error.Message)
		case 404:
			return fmt.Errorf("%w: %s", types.ErrProviderModelNotFound, errResp.Error.Message)
		default:
			return fmt.Errorf("openai error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
	}

	switch resp.StatusCode {
	case 401:
		return fmt.Errorf("%w (HTTP %d)", types.ErrProviderAuthFailed, resp.StatusCode)
	case 429:
		return fmt.Errorf("%w (HTTP %d)", types.ErrProviderRateLimited, resp.StatusCode)
	case 404:
		return fmt.Errorf("%w (HTTP %d)", types.ErrProviderModelNotFound, resp.StatusCode)
	default:
		return fmt.Errorf("openai error (HTTP %d): %s", resp.StatusCode, string(body))
	}
}

// -- OpenAI API types -------------------------------------------------------

type openAIChatRequest struct {
	Model            string               `json:"model"`
	Messages         []openAIMessage      `json:"messages"`
	Tools            []openAITool         `json:"tools,omitempty"`
	Temperature      float32              `json:"temperature,omitempty"`
	TopP             float32              `json:"top_p,omitempty"`
	MaxTokens        int                  `json:"max_tokens,omitempty"`
	Stop             []string             `json:"stop,omitempty"`
	FrequencyPenalty float32              `json:"frequency_penalty,omitempty"`
	PresencePenalty  float32              `json:"presence_penalty,omitempty"`
	Seed             *int64               `json:"seed,omitempty"`
	Stream           bool                 `json:"stream"`
	StreamOptions    *openAIStreamOptions `json:"stream_options,omitempty"`
}

type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIMessage struct {
	Role         string              `json:"-"`
	Content      string              `json:"-"`
	ContentParts []openAIContentPart `json:"-"`
	ToolCalls    []openAIToolCall    `json:"-"`
	ToolCallID   string              `json:"-"`
}

// MarshalJSON implements custom JSON marshaling for openAIMessage.
// When ContentParts is non-empty, content is serialized as an array;
// otherwise it is serialized as a plain string. Tool calls and tool_call_id
// are included when present.
func (m openAIMessage) MarshalJSON() ([]byte, error) {
	out := struct {
		Role       string           `json:"role"`
		Content    interface{}      `json:"content,omitempty"`
		ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
		ToolCallID string           `json:"tool_call_id,omitempty"`
	}{
		Role:       m.Role,
		ToolCalls:  m.ToolCalls,
		ToolCallID: m.ToolCallID,
	}
	if len(m.ContentParts) > 0 {
		out.Content = m.ContentParts
	} else if m.Content != "" {
		out.Content = m.Content
	}
	return json.Marshal(out)
}

// UnmarshalJSON implements custom JSON unmarshaling for openAIMessage.
// Supports both string and array forms of the content field, plus tool calls.
func (m *openAIMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role       string           `json:"role"`
		Content    json.RawMessage  `json:"content"`
		ToolCalls  []openAIToolCall `json:"tool_calls"`
		ToolCallID string           `json:"tool_call_id"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role
	m.ToolCalls = raw.ToolCalls
	m.ToolCallID = raw.ToolCallID

	// Content can be string, array, or null (null for tool-call-only messages).
	if len(raw.Content) > 0 && string(raw.Content) != "null" {
		if raw.Content[0] == '[' {
			return json.Unmarshal(raw.Content, &m.ContentParts)
		}
		return json.Unmarshal(raw.Content, &m.Content)
	}
	return nil
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIToolFunc `json:"function"`
}

type openAIToolFunc struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIStreamToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// toolCallAccumulator reassembles streamed tool-call fragments by index.
type toolCallAccumulator struct {
	byIndex map[int]*accEntry
	order   []int
}

type accEntry struct {
	id   string
	typ  string
	name string
	args strings.Builder
}

func newToolCallAccumulator() *toolCallAccumulator {
	return &toolCallAccumulator{byIndex: map[int]*accEntry{}}
}

func (acc *toolCallAccumulator) add(deltas []openAIStreamToolCall) {
	for _, d := range deltas {
		e, ok := acc.byIndex[d.Index]
		if !ok {
			e = &accEntry{}
			acc.byIndex[d.Index] = e
			acc.order = append(acc.order, d.Index)
		}
		if d.ID != "" {
			e.id = d.ID
		}
		if d.Type != "" {
			e.typ = d.Type
		}
		if d.Function.Name != "" {
			e.name = d.Function.Name
		}
		e.args.WriteString(d.Function.Arguments)
	}
}

// flush emits the accumulated tool calls (if any) as a single stream chunk.
func (acc *toolCallAccumulator) flush(handler types.StreamHandler) {
	if len(acc.order) == 0 {
		return
	}
	calls := make([]types.ToolCall, 0, len(acc.order))
	for _, idx := range acc.order {
		e := acc.byIndex[idx]
		tc := types.ToolCall{ID: e.id, Type: e.typ}
		if tc.Type == "" {
			tc.Type = "function"
		}
		if tc.ID == "" {
			tc.ID = "call_" + strconv.Itoa(idx)
		}
		tc.Function.Name = e.name
		tc.Function.Arguments = e.args.String()
		calls = append(calls, tc)
	}
	acc.byIndex = map[int]*accEntry{}
	acc.order = nil
	handler(&types.StreamChunk{ToolCalls: calls, Done: false})
}

type openAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIChatResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   *openAIUsage   `json:"usage,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIStreamEvent struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Model   string               `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage         `json:"usage,omitempty"`
}

type openAIStreamChoice struct {
	Index        int         `json:"index"`
	Delta        openAIDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type openAIDelta struct {
	Role      string                 `json:"role,omitempty"`
	Content   string                 `json:"content,omitempty"`
	ToolCalls []openAIStreamToolCall `json:"tool_calls,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
