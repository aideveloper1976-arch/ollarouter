package chatgpt

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// mockCredentialResolver implements CredentialResolver (legacy bearer) for tests.
type mockCredentialResolver struct {
	tokens map[string]string
}

func (m *mockCredentialResolver) GetCredential(provider string) (string, error) {
	if m == nil || m.tokens == nil {
		return "", errors.New("no credentials")
	}
	token, ok := m.tokens[provider]
	if !ok {
		return "", errors.New("credential not found")
	}
	return token, nil
}

// mockTokenResolver implements both CredentialResolver and TokenResolver.
type mockTokenResolver struct {
	token     string
	accountID string
	err       error
}

func (m *mockTokenResolver) GetCredential(provider string) (string, error) {
	return m.token, m.err
}

func (m *mockTokenResolver) UpstreamToken(provider string) (string, string, error) {
	return m.token, m.accountID, m.err
}

func TestNewAdapter(t *testing.T) {
	cfg := types.ProviderConfig{Name: "test-chatgpt", Type: "chatgpt-plus"}
	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	if adapter.Name() != "test-chatgpt" {
		t.Errorf("Name() = %q, want test-chatgpt", adapter.Name())
	}
	if adapter.Type() != "chatgpt-plus" {
		t.Errorf("Type() = %q, want chatgpt-plus", adapter.Type())
	}
}

func TestResponsesURLNormalization(t *testing.T) {
	cases := map[string]string{
		"":                                "https://chatgpt.com/backend-api/codex/responses",
		"https://chatgpt.com":             "https://chatgpt.com/backend-api/codex/responses",
		"https://chatgpt.com/backend-api": "https://chatgpt.com/backend-api/codex/responses",
		"https://example.com/backend-api/codex/responses": "https://example.com/backend-api/codex/responses",
	}
	for in, want := range cases {
		a, _ := NewAdapter(types.ProviderConfig{Name: "c", Type: "chatgpt-plus", BaseURL: in})
		if got := a.(*Adapter).responsesURL; got != want {
			t.Errorf("base %q -> responsesURL %q, want %q", in, got, want)
		}
	}
}

func TestListModelsDefaults(t *testing.T) {
	a, _ := NewAdapter(types.ProviderConfig{Name: "test-chatgpt", Type: "chatgpt-plus"})
	models, err := a.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected default models, got none")
	}
	if models[0].Name != "test-chatgpt/gpt-5.5" {
		t.Errorf("first model = %q, want test-chatgpt/gpt-5.5", models[0].Name)
	}
}

func TestCheckHealthOfflineWithoutAuth(t *testing.T) {
	a, _ := NewAdapter(types.ProviderConfig{Name: "test-chatgpt", Type: "chatgpt-plus"})
	if h := a.CheckHealth(context.Background()); h.Status != "offline" {
		t.Errorf("expected offline without auth, got %s", h.Status)
	}
}

func TestCheckHealthOnlineWithToken(t *testing.T) {
	resolver := &mockTokenResolver{token: "tok", accountID: "acc"}
	a, _ := NewAdapterWithResolver(types.ProviderConfig{Name: "test-chatgpt", Type: "chatgpt-plus"}, resolver)
	if h := a.CheckHealth(context.Background()); h.Status != "online" {
		t.Errorf("expected online with token, got %s (%s)", h.Status, h.ErrorMessage)
	}
}

func TestResolveTokenPrefersTokenResolver(t *testing.T) {
	resolver := &mockTokenResolver{token: "tok-123", accountID: "acct-9"}
	a, _ := NewAdapterWithResolver(types.ProviderConfig{Name: "test-chatgpt", Type: "chatgpt-plus"}, resolver)
	adapter := a.(*Adapter)
	tok, acc, err := adapter.resolveToken()
	if err != nil {
		t.Fatalf("resolveToken() error = %v", err)
	}
	if tok != "tok-123" || acc != "acct-9" {
		t.Errorf("got (%q,%q), want (tok-123, acct-9)", tok, acc)
	}
}

func TestSetHeaders(t *testing.T) {
	a, _ := NewAdapter(types.ProviderConfig{Name: "c", Type: "chatgpt-plus"})
	adapter := a.(*Adapter)
	req, _ := http.NewRequest("POST", "https://chatgpt.com/x", nil)
	adapter.setHeaders(req, "the-token", "the-account")

	if got := req.Header.Get("Authorization"); got != "Bearer the-token" {
		t.Errorf("Authorization = %q", got)
	}
	if got := req.Header.Get("chatgpt-account-id"); got != "the-account" {
		t.Errorf("chatgpt-account-id = %q", got)
	}
	if got := req.Header.Get("OpenAI-Beta"); got != "responses=experimental" {
		t.Errorf("OpenAI-Beta = %q", got)
	}
}

func TestBuildRequestMapsMessagesAndTools(t *testing.T) {
	a, _ := NewAdapter(types.ProviderConfig{Name: "c", Type: "chatgpt-plus"})
	adapter := a.(*Adapter)

	req := &types.ChatRequest{
		Model: "gpt-5",
		Messages: []types.Message{
			{Role: "system", Content: "be brief"},
			{Role: "user", Content: "hi"},
			{Role: "assistant", ToolCalls: []types.ToolCall{{ID: "call_1", Type: "function", Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{Name: "calc", Arguments: "{\"x\":1}"}}}},
			{Role: "tool", ToolCallID: "call_1", Content: "2"},
		},
		Tools: []types.ToolDefinition{{Type: "function", Function: types.ToolFunction{Name: "calc", Description: "calc"}}},
	}

	r := adapter.buildRequest(req)
	if r.Instructions != "be brief" {
		t.Errorf("instructions = %q", r.Instructions)
	}
	if len(r.Input) != 3 {
		t.Fatalf("expected 3 input items (user, function_call, function_call_output), got %d", len(r.Input))
	}
	if r.Input[0].Type != "message" || r.Input[0].Role != "user" {
		t.Errorf("input[0] = %+v", r.Input[0])
	}
	if r.Input[1].Type != "function_call" || r.Input[1].CallID != "call_1" || r.Input[1].Name != "calc" {
		t.Errorf("input[1] = %+v", r.Input[1])
	}
	if r.Input[2].Type != "function_call_output" || r.Input[2].CallID != "call_1" || r.Input[2].Output != "2" {
		t.Errorf("input[2] = %+v", r.Input[2])
	}
	if len(r.Tools) != 1 || r.Tools[0].Type != "function" || r.Tools[0].Name != "calc" {
		t.Errorf("tools = %+v", r.Tools)
	}
	if r.ToolChoice != "auto" {
		t.Errorf("tool_choice = %q, want auto", r.ToolChoice)
	}
}

func TestParseStreamContentAndToolCalls(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"Hel"}`,
		`data: {"type":"response.output_text.delta","delta":"lo"}`,
		`data: {"type":"response.output_item.added","item":{"type":"function_call","id":"fc_1","call_id":"call_abc","name":"run_shell","arguments":""}}`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"{\"command\":"}`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"\"ls\"}"}`,
		`data: {"type":"response.output_item.done","item":{"type":"function_call","id":"fc_1","call_id":"call_abc","name":"run_shell","arguments":"{\"command\":\"ls\"}"}}`,
		`data: {"type":"response.completed","response":{"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}`,
	}, "\n")

	a, _ := NewAdapter(types.ProviderConfig{Name: "c", Type: "chatgpt-plus"})
	adapter := a.(*Adapter)

	var content strings.Builder
	var calls []types.ToolCall
	var usage *types.TokenUsage
	err := adapter.parseStream(strings.NewReader(stream), func(c *types.StreamChunk) error {
		content.WriteString(c.Content)
		calls = append(calls, c.ToolCalls...)
		if c.TokenUsage != nil {
			usage = c.TokenUsage
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parseStream error = %v", err)
	}
	if content.String() != "Hello" {
		t.Errorf("content = %q, want Hello", content.String())
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].ID != "call_abc" || calls[0].Function.Name != "run_shell" || calls[0].Function.Arguments != `{"command":"ls"}` {
		t.Errorf("tool call = %+v", calls[0])
	}
	if usage == nil || usage.TotalTokens != 15 {
		t.Errorf("usage = %+v", usage)
	}
}
