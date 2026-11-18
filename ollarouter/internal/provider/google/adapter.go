package google

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
	httpClient *http.Client
	models     []types.ModelInfo
}

func NewAdapter(cfg types.ProviderConfig) (types.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	a := &Adapter{
		name:    cfg.Name,
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
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
func (a *Adapter) Type() string { return "google" }

func (a *Adapter) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	geminiReq := a.buildRequest(req)
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", a.baseURL, req.Model, a.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return a.translateResponse(&geminiResp, req.Model), nil
}

func (a *Adapter) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	geminiReq := a.buildRequest(req)
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", a.baseURL, req.Model, a.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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

	callSeq := 0 // monotonic id source for streamed function calls
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var geminiResp geminiResponse
		if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
			continue
		}

		if len(geminiResp.Candidates) > 0 {
			var text strings.Builder
			var calls []types.ToolCall
			for _, p := range geminiResp.Candidates[0].Content.Parts {
				if p.Text != "" {
					text.WriteString(p.Text)
				}
				if p.FunctionCall != nil {
					calls = append(calls, functionCallToToolCall(p.FunctionCall, callSeq))
					callSeq++
				}
			}
			if text.Len() > 0 {
				if err := handler(&types.StreamChunk{Content: text.String(), Done: false}); err != nil {
					return err
				}
			}
			if len(calls) > 0 {
				if err := handler(&types.StreamChunk{ToolCalls: calls, Done: false}); err != nil {
					return err
				}
			}
		}

		if geminiResp.UsageMetadata != nil {
			// Usage metadata usually comes in the last chunk
			usage := &types.TokenUsage{
				PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
				CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
			}
			handler(&types.StreamChunk{Done: true, TokenUsage: usage})
			return nil
		}
	}

	handler(&types.StreamChunk{Done: true})
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
	return a.discoverModels(ctx)
}

func (a *Adapter) discoverModels(ctx context.Context) ([]types.ModelInfo, error) {
	url := fmt.Sprintf("%s/models?key=%s", a.baseURL, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("google models API returned HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	models := make([]types.ModelInfo, 0, len(payload.Models))
	for _, m := range payload.Models {
		// Model name is like "models/gemini-1.5-pro"
		name := strings.TrimPrefix(m.Name, "models/")
		models = append(models, types.ModelInfo{
			Name:         a.name + "/" + name,
			Size:         0,
			ProviderName: a.name,
		})
	}
	a.models = models
	return models, nil
}

func (a *Adapter) CheckHealth(ctx context.Context) types.ProviderHealth {
	// Simple health check by listing models (requires key)
	url := fmt.Sprintf("%s/models?key=%s", a.baseURL, a.apiKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return types.ProviderHealth{Status: "offline", LastCheck: time.Now(), ErrorMessage: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return types.ProviderHealth{Status: "online", LastCheck: time.Now()}
	}
	return types.ProviderHealth{Status: "degraded", LastCheck: time.Now(), ErrorMessage: fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

// -- Internal Helpers ------------------------------------------------------

func (a *Adapter) buildRequest(req *types.ChatRequest) geminiRequest {
	// Gemini's functionResponse keys results by function name, but OpenAI-style
	// tool result messages only carry a tool_call_id. Build a lookup from the
	// assistant tool_calls so results can be matched back to a function name.
	idToName := map[string]string{}
	for _, m := range req.Messages {
		for _, tc := range m.ToolCalls {
			if tc.ID != "" {
				idToName[tc.ID] = tc.Function.Name
			}
		}
	}

	var systemParts []geminiPart
	contents := make([]geminiContent, 0, len(req.Messages))

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			if m.Content != "" {
				systemParts = append(systemParts, geminiPart{Text: m.Content})
			}
			continue
		case "tool":
			contents = append(contents, geminiContent{
				Role: "user",
				Parts: []geminiPart{{
					FunctionResponse: &geminiFunctionResponse{
						Name:     idToName[m.ToolCallID],
						Response: wrapToolResult(m.Content),
					},
				}},
			})
			continue
		}

		role := m.Role
		if role == "assistant" {
			role = "model"
		}

		var parts []geminiPart
		if m.Content != "" {
			parts = append(parts, geminiPart{Text: m.Content})
		}
		for _, tc := range m.ToolCalls {
			parts = append(parts, geminiPart{
				FunctionCall: &geminiFunctionCall{
					Name: tc.Function.Name,
					Args: parseArgs(tc.Function.Arguments),
				},
			})
		}
		if len(parts) == 0 {
			parts = append(parts, geminiPart{Text: ""})
		}
		contents = append(contents, geminiContent{Role: role, Parts: parts})
	}

	r := geminiRequest{Contents: contents}

	if len(systemParts) > 0 {
		r.SystemInstruction = &geminiContent{Parts: systemParts}
	}

	if len(req.Tools) > 0 {
		decls := make([]geminiFunctionDecl, 0, len(req.Tools))
		for _, t := range req.Tools {
			decls = append(decls, geminiFunctionDecl{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  sanitizeSchema(t.Function.Parameters),
			})
		}
		r.Tools = []geminiTool{{FunctionDeclarations: decls}}
	}

	if req.Options != nil {
		r.GenerationConfig = &geminiGenerationConfig{
			Temperature:     req.Options.Temperature,
			MaxOutputTokens: req.Options.MaxTokens,
		}
	}

	return r
}

func (a *Adapter) translateResponse(gr *geminiResponse, model string) *types.ChatResponse {
	cr := &types.ChatResponse{
		Model:   model,
		Message: types.Message{Role: "assistant"},
	}

	if len(gr.Candidates) > 0 {
		var text strings.Builder
		var calls []types.ToolCall
		seq := 0
		for _, p := range gr.Candidates[0].Content.Parts {
			if p.Text != "" {
				text.WriteString(p.Text)
			}
			if p.FunctionCall != nil {
				calls = append(calls, functionCallToToolCall(p.FunctionCall, seq))
				seq++
			}
		}
		cr.Message.Content = text.String()
		cr.Message.ToolCalls = calls
	}

	if gr.UsageMetadata != nil {
		cr.TokenUsage = &types.TokenUsage{
			PromptTokens:     gr.UsageMetadata.PromptTokenCount,
			CompletionTokens: gr.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      gr.UsageMetadata.TotalTokenCount,
		}
	}

	return cr
}

// functionCallToToolCall converts a Gemini functionCall into an internal
// OpenAI-style tool call, synthesizing an id (Gemini does not provide one).
func functionCallToToolCall(fc *geminiFunctionCall, seq int) types.ToolCall {
	argsBytes, _ := json.Marshal(fc.Args)
	if len(argsBytes) == 0 || string(argsBytes) == "null" {
		argsBytes = []byte("{}")
	}
	tc := types.ToolCall{ID: "call_" + strconv.Itoa(seq), Type: "function"}
	tc.Function.Name = fc.Name
	tc.Function.Arguments = string(argsBytes)
	return tc
}

// parseArgs converts an OpenAI-style JSON arguments string into a Gemini args object.
func parseArgs(s string) map[string]interface{} {
	if strings.TrimSpace(s) == "" {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if json.Unmarshal([]byte(s), &m) == nil {
		return m
	}
	return map[string]interface{}{}
}

// wrapToolResult coerces a tool-result string into the object Gemini expects.
func wrapToolResult(content string) map[string]interface{} {
	var m map[string]interface{}
	if json.Unmarshal([]byte(content), &m) == nil {
		return m
	}
	return map[string]interface{}{"result": content}
}

// sanitizeSchema removes JSON-Schema keywords that the Gemini function-calling
// API rejects (e.g. "$schema", "additionalProperties"). It walks the schema
// recursively and returns a cleaned copy.
func sanitizeSchema(in interface{}) interface{} {
	switch v := in.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			switch k {
			case "$schema", "additionalProperties", "$id", "$ref", "definitions", "$defs":
				continue
			}
			out[k] = sanitizeSchema(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = sanitizeSchema(item)
		}
		return out
	default:
		return in
	}
}

func (a *Adapter) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("google api error (HTTP %d): %s", resp.StatusCode, string(body))
}

// -- Gemini API Types ------------------------------------------------------

type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	Tools             []geminiTool            `json:"tools,omitempty"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response,omitempty"`
}

type geminiGenerationConfig struct {
	Temperature     float32 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content       geminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
	SafetyRatings []interface{} `json:"safetyRatings"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}
