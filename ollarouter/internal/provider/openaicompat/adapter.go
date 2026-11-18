package openaicompat

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
	name         string
	providerType string
	baseURL      string
	apiKey       string
	headers      map[string]string
	httpClient   *http.Client
	models       []types.ModelInfo
}

func NewAdapter(cfg types.ProviderConfig) (types.Provider, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	providerType := cfg.Type
	if providerType == "" {
		providerType = "openai-compatible"
	}
	if providerType == "openrouter" && baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	if providerType == "deepseek" && baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	a := &Adapter{
		name:         cfg.Name,
		providerType: providerType,
		baseURL:      baseURL,
		apiKey:       cfg.APIKey,
		headers:      cfg.Headers,
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
func (a *Adapter) Type() string { return a.providerType }

func (a *Adapter) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	openAIReq := a.buildRequest(req)
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

	var openAIResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	cr := &types.ChatResponse{Model: openAIResp.Model}
	if len(openAIResp.Choices) > 0 {
		msg := openAIResp.Choices[0].Message
		cr.Message = types.Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
		}
		if cr.Message.Role == "" {
			cr.Message.Role = "assistant"
		}
		cr.Message.ToolCalls = toInternalToolCalls(msg.ToolCalls)
	}
	if openAIResp.Usage != nil {
		cr.TokenUsage = &types.TokenUsage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.PromptTokens + openAIResp.Usage.CompletionTokens,
		}
	}
	return cr, nil
}

func (a *Adapter) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	openAIReq := a.buildRequest(req)
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
		// Fall back to non-streaming
		bodyBytes, _ := io.ReadAll(resp.Body)
		var cResp chatResponse
		if json.Unmarshal(bodyBytes, &cResp) == nil && len(cResp.Choices) > 0 {
			msg := cResp.Choices[0].Message
			if msg.Content != "" {
				handler(&types.StreamChunk{Content: msg.Content, Done: false})
			}
			if tcs := toInternalToolCalls(msg.ToolCalls); len(tcs) > 0 {
				handler(&types.StreamChunk{ToolCalls: tcs, Done: false})
			}
			handler(&types.StreamChunk{Done: true})
			return nil
		}
		// Re-wrap the already-consumed body so handleError can report it.
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return a.handleError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// Streamed tool calls arrive as fragments keyed by index; accumulate them
	// and emit complete calls once the upstream signals finish.
	acc := newToolCallAccumulator()
	var finalUsage *types.TokenUsage

	doneSent := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			acc.flush(handler)
			doneSent = true
			handler(&types.StreamChunk{Done: true, TokenUsage: finalUsage})
			return nil
		}

		var event streamEvent
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		if event.Usage != nil {
			finalUsage = &types.TokenUsage{
				PromptTokens:     event.Usage.PromptTokens,
				CompletionTokens: event.Usage.CompletionTokens,
				TotalTokens:      event.Usage.PromptTokens + event.Usage.CompletionTokens,
			}
		}

		if len(event.Choices) > 0 {
			ch := event.Choices[0]
			if len(ch.Delta.ToolCalls) > 0 {
				acc.add(ch.Delta.ToolCalls)
			} else if ch.Delta.Content != "" {
				handler(&types.StreamChunk{Content: ch.Delta.Content, Done: false})
			} else if ch.Delta.ReasoningContent != "" {
				handler(&types.StreamChunk{ReasoningContent: ch.Delta.ReasoningContent, Done: false})
			}
			if ch.FinishReason != "" {
				acc.flush(handler)
				doneSent = true
				handler(&types.StreamChunk{Done: true, TokenUsage: finalUsage})
				return nil
			}
		}
	}

	if !doneSent {
		acc.flush(handler)
		handler(&types.StreamChunk{Done: true, TokenUsage: finalUsage})
	}

	return scanner.Err()
}

func (a *Adapter) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	chatReq := &types.ChatRequest{
		Model:    req.Model,
		Messages: []types.Message{{Role: "user", Content: req.Prompt}},
		Options:  req.Options,
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
		Model:    req.Model,
		Messages: []types.Message{{Role: "user", Content: req.Prompt}},
		Options:  req.Options,
		Stream:   true,
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
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+a.modelDiscoveryPath(), nil)
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
		return nil, fmt.Errorf("%s models API returned HTTP %d", a.providerType, resp.StatusCode)
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
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+a.modelDiscoveryPath(), nil)
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
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
	for k, v := range a.headers {
		req.Header.Set(k, v)
	}
}

func (a *Adapter) GetBalance(ctx context.Context) (*types.ProviderBalance, error) {
	switch a.providerType {
	case "openrouter":
		return a.getOpenRouterBalance(ctx)
	case "deepseek":
		return a.getDeepSeekBalance(ctx)
	default:
		return nil, fmt.Errorf("balance API is not supported for provider type %s", a.providerType)
	}
}

func (a *Adapter) getOpenRouterBalance(ctx context.Context) (*types.ProviderBalance, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+"/credits", nil)
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, a.handleError(resp)
	}

	var payload struct {
		Data struct {
			TotalCredits float64 `json:"total_credits"`
			TotalUsage   float64 `json:"total_usage"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return &types.ProviderBalance{
		Currency:         "USD",
		TotalCredits:     payload.Data.TotalCredits,
		TotalUsage:       payload.Data.TotalUsage,
		RemainingCredits: payload.Data.TotalCredits - payload.Data.TotalUsage,
		Source:           "openrouter",
	}, nil
}

// getDeepSeekBalance calls DeepSeek's GET /user/balance endpoint.
// DeepSeek reports the current available balance rather than a
// cumulative credits/usage pair, so total_credits/total_usage are
// derived from granted+topped-up balance vs. what's left, purely to
// fit the shared ProviderBalance shape the UI already renders.
func (a *Adapter) getDeepSeekBalance(ctx context.Context) (*types.ProviderBalance, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+"/user/balance", nil)
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, a.handleError(resp)
	}

	var payload struct {
		IsAvailable  bool `json:"is_available"`
		BalanceInfos []struct {
			Currency        string `json:"currency"`
			TotalBalance    string `json:"total_balance"`
			GrantedBalance  string `json:"granted_balance"`
			ToppedUpBalance string `json:"topped_up_balance"`
		} `json:"balance_infos"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.BalanceInfos) == 0 {
		return nil, fmt.Errorf("deepseek balance response contained no balance_infos")
	}

	info := payload.BalanceInfos[0]
	totalBalance, _ := strconv.ParseFloat(info.TotalBalance, 64)
	grantedBalance, _ := strconv.ParseFloat(info.GrantedBalance, 64)
	toppedUpBalance, _ := strconv.ParseFloat(info.ToppedUpBalance, 64)
	totalCredits := grantedBalance + toppedUpBalance

	return &types.ProviderBalance{
		Currency:         info.Currency,
		TotalCredits:     totalCredits,
		TotalUsage:       totalCredits - totalBalance,
		RemainingCredits: totalBalance,
		Source:           "deepseek",
	}, nil
}

func (a *Adapter) buildRequest(req *types.ChatRequest) chatRequest {
	messages := make([]message, len(req.Messages))
	for i, m := range req.Messages {
		msg := message{
			Role:             m.Role,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
			ToolCallID:       m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			ntc := toolCall{ID: tc.ID, Type: tc.Type}
			if ntc.Type == "" {
				ntc.Type = "function"
			}
			ntc.Function.Name = tc.Function.Name
			ntc.Function.Arguments = tc.Function.Arguments
			msg.ToolCalls = append(msg.ToolCalls, ntc)
		}
		messages[i] = msg
	}
	r := chatRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}
	for _, t := range req.Tools {
		td := toolDefinition{
			Type: t.Type,
			Function: toolFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
		if td.Type == "" {
			td.Type = "function"
		}
		r.Tools = append(r.Tools, td)
	}
	if req.Options != nil {
		r.Temperature = req.Options.Temperature
		r.MaxTokens = req.Options.MaxTokens
	}
	if req.Stream {
		r.StreamOptions = &streamOptions{IncludeUsage: true}
	}
	return r
}

func (a *Adapter) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case 401:
		return fmt.Errorf("%w (HTTP %d)", types.ErrProviderAuthFailed, resp.StatusCode)
	case 429:
		return fmt.Errorf("%w (HTTP %d)", types.ErrProviderRateLimited, resp.StatusCode)
	default:
		return fmt.Errorf("%s error (HTTP %d): %s", a.providerType, resp.StatusCode, string(body))
	}
}

func (a *Adapter) modelDiscoveryPath() string {
	if a.providerType == "openrouter" {
		return "/models/user"
	}
	return "/models"
}

// -- Tool call helpers ----------------------------------------------------

// toInternalToolCalls converts wire-format tool calls to the internal type.
func toInternalToolCalls(in []toolCall) []types.ToolCall {
	if len(in) == 0 {
		return nil
	}
	out := make([]types.ToolCall, len(in))
	for i, tc := range in {
		out[i] = types.ToolCall{ID: tc.ID, Type: tc.Type}
		if out[i].Type == "" {
			out[i].Type = "function"
		}
		out[i].Function.Name = tc.Function.Name
		out[i].Function.Arguments = tc.Function.Arguments
	}
	return out
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

func (acc *toolCallAccumulator) add(deltas []streamToolCall) {
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

// -- Types ----------------------------------------------------------------

type chatRequest struct {
	Model         string           `json:"model"`
	Messages      []message        `json:"messages"`
	Tools         []toolDefinition `json:"tools,omitempty"`
	Temperature   float32          `json:"temperature,omitempty"`
	MaxTokens     int              `json:"max_tokens,omitempty"`
	Stream        bool             `json:"stream"`
	StreamOptions *streamOptions   `json:"stream_options,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type toolDefinition struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []toolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Model   string       `json:"model"`
	Choices []choice     `json:"choices"`
	Usage   *openAIUsage `json:"usage,omitempty"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type streamEvent struct {
	Choices []streamChoice `json:"choices"`
	Usage   *openAIUsage   `json:"usage,omitempty"`
}

type streamChoice struct {
	Delta        streamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

type streamDelta struct {
	Role             string           `json:"role,omitempty"`
	Content          string           `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ToolCalls        []streamToolCall `json:"tool_calls,omitempty"`
}

type streamToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
