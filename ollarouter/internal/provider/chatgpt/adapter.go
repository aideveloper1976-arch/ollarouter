package chatgpt

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

// CredentialResolver resolves a bearer credential for a provider.
type CredentialResolver interface {
	GetCredential(provider string) (string, error)
}

// TokenResolver resolves a (refreshed) access token plus the ChatGPT account id.
// The upstream auth manager implements this; the adapter prefers it over the
// plain CredentialResolver because the Codex backend needs the account id.
type TokenResolver interface {
	UpstreamToken(provider string) (accessToken, accountID string, err error)
}

// Default Codex Responses backend. Overridable via provider base_url.
const defaultResponsesURL = "https://chatgpt.com/backend-api/codex/responses"

type Adapter struct {
	name               string
	responsesURL       string
	sessionCookie      string
	userAgent          string
	httpClient         *http.Client
	models             []types.ModelInfo
	credentialResolver CredentialResolver
}

func NewAdapter(cfg types.ProviderConfig) (types.Provider, error) {
	return newAdapter(cfg, nil)
}

// NewAdapterWithResolver creates a ChatGPT adapter backed by a credential
// resolver (OAuth "Sign in with ChatGPT" tokens, with legacy bearer fallback).
func NewAdapterWithResolver(cfg types.ProviderConfig, resolver CredentialResolver) (types.Provider, error) {
	return newAdapter(cfg, resolver)
}

func newAdapter(cfg types.ProviderConfig, resolver CredentialResolver) (types.Provider, error) {
	// base_url, if set, points at the Codex responses endpoint host. We accept
	// either a full endpoint or a host root and normalize to the responses path.
	responsesURL := defaultResponsesURL
	if cfg.BaseURL != "" {
		b := strings.TrimRight(cfg.BaseURL, "/")
		if strings.Contains(b, "/responses") {
			responsesURL = b
		} else if strings.Contains(b, "/backend-api") {
			responsesURL = b + "/codex/responses"
		} else {
			responsesURL = b + "/backend-api/codex/responses"
		}
	}

	ua := cfg.UserAgent
	if ua == "" {
		// Advertise a recent Codex CLI version; the backend hides models from
		// clients older than each model's minimal_client_version.
		ua = "codex_cli_rs/0.140.0 (OllaRouter)"
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	a := &Adapter{
		name:          cfg.Name,
		responsesURL:  responsesURL,
		sessionCookie: cfg.SessionCookie,
		userAgent:     ua,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		credentialResolver: resolver,
	}

	for _, m := range cfg.Models {
		a.models = append(a.models, types.ModelInfo{
			Name:         cfg.Name + "/" + m.Name,
			ProviderName: cfg.Name,
		})
	}

	return a, nil
}

func (a *Adapter) Name() string { return a.name }
func (a *Adapter) Type() string { return "chatgpt-plus" }

// resolveToken returns a valid access token and account id, preferring the
// refreshing TokenResolver and falling back to a plain credential.
func (a *Adapter) resolveToken() (token, accountID string, err error) {
	if a.credentialResolver != nil {
		if tr, ok := a.credentialResolver.(TokenResolver); ok {
			if t, acc, e := tr.UpstreamToken(a.name); e == nil && t != "" {
				return t, acc, nil
			} else if e != nil {
				err = e
			}
		}
		if t, e := a.credentialResolver.GetCredential(a.name); e == nil && t != "" {
			return t, "", nil
		} else if e != nil {
			err = e
		}
	}
	if a.sessionCookie != "" {
		return "", "", fmt.Errorf("%w: a ChatGPT session cookie cannot authenticate the Codex backend; sign in with OAuth", types.ErrProviderAuthFailed)
	}
	if err == nil {
		err = fmt.Errorf("%w: not signed in to ChatGPT (run the OAuth connect flow)", types.ErrProviderAuthFailed)
	}
	return "", "", err
}

func (a *Adapter) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	// The Codex backend is streaming-only; buffer the stream into one response.
	var content strings.Builder
	var reasoning strings.Builder
	var toolCalls []types.ToolCall
	var usage *types.TokenUsage

	err := a.ChatStream(ctx, req, func(chunk *types.StreamChunk) error {
		content.WriteString(chunk.Content)
		reasoning.WriteString(chunk.ReasoningContent)
		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}
		if chunk.TokenUsage != nil {
			usage = chunk.TokenUsage
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.ChatResponse{
		Model: req.Model,
		Message: types.Message{
			Role:             "assistant",
			Content:          content.String(),
			ReasoningContent: reasoning.String(),
			ToolCalls:        toolCalls,
		},
		TokenUsage: usage,
	}, nil
}

func (a *Adapter) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	token, accountID, err := a.resolveToken()
	if err != nil {
		return err
	}

	reqBody := a.buildRequest(req)
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.responsesURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(httpReq, token, accountID)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", types.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return a.handleError(resp)
	}

	return a.parseStream(resp.Body, handler)
}

// parseStream reads the Codex Responses SSE stream and emits internal chunks.
func (a *Adapter) parseStream(r io.Reader, handler types.StreamHandler) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	acc := newFnAccumulator()
	var usage *types.TokenUsage
	completed := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var ev responsesEvent
		if json.Unmarshal([]byte(data), &ev) != nil {
			continue
		}

		switch ev.Type {
		case "response.output_text.delta":
			if ev.Delta != "" {
				if err := handler(&types.StreamChunk{Content: ev.Delta, Done: false}); err != nil {
					return err
				}
			}
		case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
			if ev.Delta != "" {
				if err := handler(&types.StreamChunk{ReasoningContent: ev.Delta, Done: false}); err != nil {
					return err
				}
			}
		case "response.output_item.added":
			if ev.Item != nil && ev.Item.Type == "function_call" {
				e := acc.get(ev.Item.itemKey())
				e.merge(ev.Item)
			}
		case "response.function_call_arguments.delta":
			if ev.ItemID != "" && ev.Delta != "" {
				acc.get(ev.ItemID).args.WriteString(ev.Delta)
			}
		case "response.output_item.done":
			if ev.Item != nil && ev.Item.Type == "function_call" {
				e := acc.get(ev.Item.itemKey())
				e.mergeFinal(ev.Item)
			}
		case "response.completed":
			if ev.Response != nil && ev.Response.Usage != nil {
				usage = ev.Response.Usage.toTokenUsage()
			}
			acc.flush(handler)
			handler(&types.StreamChunk{Done: true, TokenUsage: usage})
			completed = true
			return nil
		case "response.failed", "error":
			msg := "ChatGPT Codex API error"
			if ev.Response != nil && ev.Response.Error != nil && ev.Response.Error.Message != "" {
				msg = ev.Response.Error.Message
			}
			return fmt.Errorf("%w: %s", types.ErrProviderUnavailable, msg)
		}
	}

	if !completed {
		acc.flush(handler)
		handler(&types.StreamChunk{Done: true, TokenUsage: usage})
	}
	return scanner.Err()
}

func (a *Adapter) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	chatReq := &types.ChatRequest{
		Model:    req.Model,
		Messages: []types.Message{{Role: "user", Content: req.Prompt}},
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
		Model:    req.Model,
		Messages: []types.Message{{Role: "user", Content: req.Prompt}},
		Stream:   true,
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
	// The set of Codex models available to a ChatGPT account is small and tied to
	// the plan; gpt-5.5 is current. Override via the provider's `models:` config.
	defaults := []string{"gpt-5.5"}
	out := make([]types.ModelInfo, len(defaults))
	for i, slug := range defaults {
		out[i] = types.ModelInfo{Name: a.name + "/" + slug, ProviderName: a.name}
	}
	return out, nil
}

func (a *Adapter) CheckHealth(ctx context.Context) types.ProviderHealth {
	if _, _, err := a.resolveToken(); err != nil {
		return types.ProviderHealth{
			Status:       "offline",
			LastCheck:    time.Now(),
			ErrorMessage: "not signed in (OAuth connect required)",
		}
	}
	return types.ProviderHealth{Status: "online", LastCheck: time.Now()}
}

func (a *Adapter) setHeaders(req *http.Request, token, accountID string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if accountID != "" {
		req.Header.Set("chatgpt-account-id", accountID)
	}
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("originator", "codex_cli_rs")
	req.Header.Set("User-Agent", a.userAgent)
}

func (a *Adapter) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case 401, 403:
		return fmt.Errorf("%w: ChatGPT auth rejected (token expired or account lacks Codex access): %s", types.ErrProviderAuthFailed, truncate(string(body), 300))
	case 429:
		return fmt.Errorf("%w: ChatGPT usage limit reached", types.ErrProviderRateLimited)
	default:
		return fmt.Errorf("chatgpt codex error (HTTP %d): %s", resp.StatusCode, truncate(string(body), 500))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// -- Request building ------------------------------------------------------

func (a *Adapter) buildRequest(req *types.ChatRequest) responsesRequest {
	r := responsesRequest{
		Model:  req.Model,
		Store:  false,
		Stream: true,
	}

	var instructions strings.Builder
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			if instructions.Len() > 0 {
				instructions.WriteString("\n\n")
			}
			instructions.WriteString(m.Content)
		case "tool":
			r.Input = append(r.Input, responsesInput{
				Type:   "function_call_output",
				CallID: m.ToolCallID,
				Output: m.Content,
			})
		case "assistant":
			if m.Content != "" {
				r.Input = append(r.Input, responsesInput{
					Type:    "message",
					Role:    "assistant",
					Content: []responsesContent{{Type: "output_text", Text: m.Content}},
				})
			}
			for _, tc := range m.ToolCalls {
				r.Input = append(r.Input, responsesInput{
					Type:      "function_call",
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
					CallID:    tc.ID,
				})
			}
		default: // user and anything else
			r.Input = append(r.Input, responsesInput{
				Type:    "message",
				Role:    "user",
				Content: []responsesContent{{Type: "input_text", Text: m.Content}},
			})
		}
	}
	r.Instructions = instructions.String()
	// The Codex backend rejects requests without instructions ("Instructions are
	// required"). Supply a neutral default when no system message was provided.
	if strings.TrimSpace(r.Instructions) == "" {
		r.Instructions = "You are a helpful assistant."
	}

	for _, t := range req.Tools {
		r.Tools = append(r.Tools, responsesTool{
			Type:        "function",
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  t.Function.Parameters,
		})
	}
	if len(r.Tools) > 0 {
		r.ToolChoice = "auto"
		r.ParallelToolCalls = true
	}

	return r
}

// -- Function-call accumulation -------------------------------------------

type fnEntry struct {
	callID string
	name   string
	args   strings.Builder
}

func (e *fnEntry) merge(item *responsesEventItem) {
	if item.CallID != "" {
		e.callID = item.CallID
	}
	if item.Name != "" {
		e.name = item.Name
	}
	if item.Arguments != "" {
		e.args.WriteString(item.Arguments)
	}
}

// mergeFinal applies the authoritative final item: complete arguments replace
// any streamed fragments.
func (e *fnEntry) mergeFinal(item *responsesEventItem) {
	if item.CallID != "" {
		e.callID = item.CallID
	}
	if item.Name != "" {
		e.name = item.Name
	}
	if item.Arguments != "" {
		e.args.Reset()
		e.args.WriteString(item.Arguments)
	}
}

type fnAccumulator struct {
	byKey map[string]*fnEntry
	keys  []string
}

func newFnAccumulator() *fnAccumulator {
	return &fnAccumulator{byKey: map[string]*fnEntry{}}
}

func (acc *fnAccumulator) get(key string) *fnEntry {
	if e, ok := acc.byKey[key]; ok {
		return e
	}
	e := &fnEntry{}
	acc.byKey[key] = e
	acc.keys = append(acc.keys, key)
	return e
}

func (acc *fnAccumulator) flush(handler types.StreamHandler) {
	if len(acc.keys) == 0 {
		return
	}
	calls := make([]types.ToolCall, 0, len(acc.keys))
	for i, k := range acc.keys {
		e := acc.byKey[k]
		id := e.callID
		if id == "" {
			id = "call_" + strconv.Itoa(i)
		}
		tc := types.ToolCall{ID: id, Type: "function"}
		tc.Function.Name = e.name
		tc.Function.Arguments = e.args.String()
		if tc.Function.Arguments == "" {
			tc.Function.Arguments = "{}"
		}
		calls = append(calls, tc)
	}
	acc.byKey = map[string]*fnEntry{}
	acc.keys = nil
	handler(&types.StreamChunk{ToolCalls: calls, Done: false})
}

// -- Codex Responses API types --------------------------------------------

type responsesRequest struct {
	Model             string           `json:"model"`
	Instructions      string           `json:"instructions,omitempty"`
	Input             []responsesInput `json:"input"`
	Tools             []responsesTool  `json:"tools,omitempty"`
	ToolChoice        string           `json:"tool_choice,omitempty"`
	ParallelToolCalls bool             `json:"parallel_tool_calls,omitempty"`
	Store             bool             `json:"store"`
	Stream            bool             `json:"stream"`
}

type responsesInput struct {
	Type      string             `json:"type"`
	Role      string             `json:"role,omitempty"`
	Content   []responsesContent `json:"content,omitempty"`
	Name      string             `json:"name,omitempty"`
	Arguments string             `json:"arguments,omitempty"`
	CallID    string             `json:"call_id,omitempty"`
	Output    string             `json:"output,omitempty"`
}

type responsesContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesTool struct {
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type responsesEvent struct {
	Type     string                  `json:"type"`
	Delta    string                  `json:"delta,omitempty"`
	ItemID   string                  `json:"item_id,omitempty"`
	Item     *responsesEventItem     `json:"item,omitempty"`
	Response *responsesEventResponse `json:"response,omitempty"`
}

type responsesEventItem struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// itemKey returns a stable key for accumulating an item's fragments.
func (it *responsesEventItem) itemKey() string {
	if it.ID != "" {
		return it.ID
	}
	return it.CallID
}

type responsesEventResponse struct {
	Usage *responsesUsage `json:"usage,omitempty"`
	Error *responsesError `json:"error,omitempty"`
}

type responsesError struct {
	Message string `json:"message"`
}

type responsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (u *responsesUsage) toTokenUsage() *types.TokenUsage {
	total := u.TotalTokens
	if total == 0 {
		total = u.InputTokens + u.OutputTokens
	}
	return &types.TokenUsage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      total,
	}
}
