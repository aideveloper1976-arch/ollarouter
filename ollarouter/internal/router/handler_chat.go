package router

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/guardrails"
	"github.com/aideveloper1976-arch/ollarouter/internal/streaming"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// FallbackVersion is the version reported by /api/version, /health and
// /metrics. Release builds override it with -ldflags -X.
var FallbackVersion = "0.22.0"

// ollamaChatRequest is the incoming Ollama chat format.
type ollamaChatRequest struct {
	Model    string                 `json:"model"`
	Messages []ollamaMsg            `json:"messages"`
	Stream   bool                   `json:"stream"`
	Tools    []ollamaToolDef        `json:"tools,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ollamaMsg struct {
	Role             string               `json:"role"`
	Content          interface{}          `json:"content"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
	Images           []string             `json:"images,omitempty"`
	ToolCalls        []ollamaToolCallItem `json:"tool_calls,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`
}

var leakagePrefixes = []string{
	"Considering tool usage",
	"I'm thinking about",
	"I’m thinking about",
	"I'll proceed with",
	"I will proceed with",
	"Now, I need to",
	"Let me think",
	"I should",
	"Let me explore",
	"Plan mode ACTIVE",
}

var blockedTaggedSections = []string{
	"system-reminder",
	"read_file",
	"semantic_search",
	"tool_calls",
	"invoke",
	"parameter",
}

func stripTaggedBlocks(raw string) string {
	out := raw
	for _, name := range blockedTaggedSections {
		openTag := "<" + name + ">"
		closeTag := "</" + name + ">"
		for {
			start := strings.Index(out, openTag)
			if start < 0 {
				break
			}
			end := strings.Index(out[start:], closeTag)
			if end < 0 {
				out = out[:start]
				break
			}
			end = start + end + len(closeTag)
			out = out[:start] + out[end:]
		}
	}
	return out
}

func isToolLikeTagLine(t string) bool {
	if !strings.HasPrefix(t, "<") || !strings.HasSuffix(t, ">") {
		return false
	}
	tag := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "<"), ">"))
	tag = strings.TrimPrefix(tag, "/")
	if tag == "" {
		return false
	}
	parts := strings.Fields(tag)
	name := strings.ToLower(parts[0])
	if strings.Contains(name, "-") || strings.Contains(name, "_") {
		return true
	}
	switch name {
	case "read_file", "semantic_search", "system-reminder", "tool_calls", "invoke", "parameter", "file", "query":
		return true
	default:
		return false
	}
}

func isLeakageLine(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return false
	}

	if strings.Contains(t, "<｜｜DSML｜｜") || strings.Contains(t, "</｜｜DSML｜｜") {
		return true
	}
	if strings.HasPrefix(t, "<tool_calls>") || strings.HasPrefix(t, "</tool_calls>") {
		return true
	}
	if strings.HasPrefix(t, "<invoke ") || strings.HasPrefix(t, "</invoke>") {
		return true
	}
	if strings.HasPrefix(t, "<parameter ") || strings.HasPrefix(t, "</parameter>") {
		return true
	}
	if isToolLikeTagLine(t) {
		return true
	}

	for _, p := range leakagePrefixes {
		if strings.HasPrefix(t, p) {
			return true
		}
	}

	return false
}

func sanitizeAssistantContent(raw string) string {
	if raw == "" {
		return raw
	}
	raw = stripTaggedBlocks(raw)
	lines := strings.Split(raw, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if isLeakageLine(line) {
			continue
		}
		kept = append(kept, line)
	}
	out := strings.TrimSpace(strings.Join(kept, "\n"))
	out = strings.ReplaceAll(out, "<｜｜DSML｜｜", "")
	return strings.TrimSpace(out)
}

func sanitizeStreamingDelta(state *string, in string, flush bool) string {
	*state += in
	if !flush && !strings.Contains(*state, "\n") {
		return ""
	}

	parts := strings.Split(*state, "\n")
	last := ""
	if !flush {
		last = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}

	kept := make([]string, 0, len(parts))
	for _, line := range parts {
		if isLeakageLine(line) {
			continue
		}
		kept = append(kept, line)
	}

	*state = last
	if len(kept) == 0 {
		return ""
	}
	return strings.Join(kept, "\n")
}

// contentToString converts OpenAI content (string or array of content parts) to a plain string.
func contentToString(content interface{}) string {
	if content == nil {
		return ""
	}
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, part := range v {
			if m, ok := part.(map[string]interface{}); ok {
				if t, ok := m["type"].(string); ok && t == "text" {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%v", content)
	}
}

type ollamaToolDef struct {
	Type     string            `json:"type"`
	Function ollamaToolFuncDef `json:"function"`
}

type ollamaToolFuncDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type ollamaToolCallItem struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function ollamaToolCallFunc `json:"function"`
}

type ollamaToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func handleChat(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body: "+err.Error())
			return
		}

		if req.Model == "" {
			writeError(w, 400, "model is required")
			return
		}

		if len(req.Messages) == 0 {
			writeError(w, 400, "messages is required")
			return
		}

		// Convert to internal format
		messages := make([]types.Message, len(req.Messages))
		for i, m := range req.Messages {
			msg := types.Message{
				Role:             m.Role,
				Content:          contentToString(m.Content),
				ReasoningContent: m.ReasoningContent,
				Images:           m.Images,
				ToolCallID:       m.ToolCallID,
			}
			if len(m.ToolCalls) > 0 {
				msg.ToolCalls = make([]types.ToolCall, len(m.ToolCalls))
				for j, tc := range m.ToolCalls {
					msg.ToolCalls[j] = types.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
			messages[i] = msg
		}

		tools := make([]types.ToolDefinition, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = types.ToolDefinition{
				Type: t.Type,
				Function: types.ToolFunction{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}

		internalReq := &types.ChatRequest{
			Model:    req.Model,
			Messages: messages,
			Tools:    tools,
			Stream:   req.Stream,
		}

		// Resolve model - check virtual models first, then real provider models
		ctx := r.Context()
		startTime := time.Now()

		// Check if this is a virtual model
		chain, isVirtual := app.Engine.ResolveVirtualModel(req.Model)
		if isVirtual == nil {
			// Virtual model found - use fallback executor
			app.Logger.Info("resolved virtual model", "model", req.Model, "chain", chain)

			if req.Stream {
				// Streaming with fallback
				w.Header().Set("Content-Type", "application/x-ndjson")
				flusher, ok := w.(http.Flusher)
				if !ok {
					writeError(w, 500, "streaming not supported")
					return
				}
				w.WriteHeader(http.StatusOK)
				flusher.Flush()

				writer := streaming.NewOllamaWriter(w, req.Model)

				var lastUsage *types.TokenUsage
				var finalModel string
				providerName, err := app.Fallback.ExecuteWithFallback(ctx, chain, func(ctx context.Context, pName, mName string) error {
					ps, ok := app.Registry.Get(pName)
					if !ok {
						return types.ErrProviderUnavailable
					}
					finalModel = mName
					internalReq.Model = mName
					return ps.Provider.ChatStream(ctx, internalReq, func(chunk *types.StreamChunk) error {
						if chunk.Done && chunk.TokenUsage != nil {
							lastUsage = chunk.TokenUsage
						}
						return writer.WriteChunk(chunk)
					})
				})

				latency := time.Since(startTime).Milliseconds()
				if err != nil {
					writer.WriteError(err.Error())
					app.recordUsage("", req.Model, 502, nil, latency)
				} else {
					app.Logger.Info("streaming finished (virtual)",
						"virtual_model", req.Model,
						"actual_model", finalModel,
						"provider", providerName,
						"tokens", lastUsage,
					)
					app.recordUsage(providerName, finalModel, 200, lastUsage, latency)
				}
				return
			}

			// Non-streaming with fallback
			var chatResp *types.ChatResponse
			providerName, err := app.Fallback.ExecuteWithFallback(ctx, chain, func(ctx context.Context, pName, mName string) error {
				ps, ok := app.Registry.Get(pName)
				if !ok {
					return types.ErrProviderUnavailable
				}
				internalReq.Model = mName
				var cerr error
				chatResp, cerr = ps.Provider.Chat(ctx, internalReq)
				return cerr
			})

			latency := time.Since(startTime).Milliseconds()
			if err != nil {
				writeError(w, 502, err.Error())
				app.recordUsage("", req.Model, 502, nil, latency)
				return
			}

			app.Logger.Info("model response",
				"model", chatResp.Model,
				"provider", providerName,
				"tokens", chatResp.TokenUsage,
				"content_snippet", truncateString(chatResp.Message.Content, 100),
			)

			app.recordUsage(providerName, chatResp.Model, 200, chatResp.TokenUsage, latency)

			respMsg := ollamaMsg{Role: chatResp.Message.Role, Content: sanitizeAssistantContent(chatResp.Message.Content), ReasoningContent: chatResp.Message.ReasoningContent}
			if len(chatResp.Message.ToolCalls) > 0 {
				respMsg.ToolCalls = make([]ollamaToolCallItem, len(chatResp.Message.ToolCalls))
				for j, tc := range chatResp.Message.ToolCalls {
					respMsg.ToolCalls[j] = ollamaToolCallItem{
						ID:   tc.ID,
						Type: tc.Type,
						Function: ollamaToolCallFunc{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
			ollamaResp := ollamaChatResponse{
				Model:     req.Model,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
				Message:   respMsg,
				Done:      true,
			}

			if chatResp.TokenUsage != nil {
				ollamaResp.PromptEvalCount = chatResp.TokenUsage.PromptTokens
				ollamaResp.EvalCount = chatResp.TokenUsage.CompletionTokens
			}

			writeJSON(w, http.StatusOK, ollamaResp)
			return
		}

		// Not a virtual model - resolve from provider registry
		var targetProvider types.Provider
		var targetModel string
		var targetProviderName string
		var targetBaseURL string

		for _, ps := range app.Registry.List() {
			for _, mi := range ps.ModelCache {
				shortName := shortModelName(mi.Name)
				if mi.Name == req.Model || shortName == req.Model {
					targetProvider = ps.Provider
					targetModel = shortName
					targetProviderName = ps.Config.Name
					targetBaseURL = ps.Config.BaseURL
					break
				}
			}
			if targetProvider != nil {
				break
			}
		}

		if targetProvider == nil {
			for _, ps := range app.Registry.List() {
				for _, mi := range ps.ModelCache {
					shortName := shortModelName(mi.Name)
					if shortName == req.Model {
						targetProvider = ps.Provider
						targetModel = shortName
						targetProviderName = ps.Config.Name
						targetBaseURL = ps.Config.BaseURL
						break
					}
				}
				if targetProvider != nil {
					break
				}
			}
		}

		if targetProvider == nil {
			writeError(w, 404, "model \""+req.Model+"\" not found")
			return
		}

		app.Logger.Info("provider chain",
			"model", req.Model,
			"provider", targetProviderName,
			"provider_type", targetProvider.Type(),
			"target_model", targetModel,
			"base_url", targetBaseURL,
		)

		internalReq.Model = targetModel

		// Streaming
		if req.Stream {
			w.Header().Set("Content-Type", "application/x-ndjson")
			flusher, ok := w.(http.Flusher)
			if !ok {
				writeError(w, 500, "streaming not supported")
				return
			}
			w.WriteHeader(http.StatusOK)
			flusher.Flush()

			writer := streaming.NewOllamaWriter(w, req.Model)

			var lastUsage *types.TokenUsage
			err := targetProvider.ChatStream(ctx, internalReq, func(chunk *types.StreamChunk) error {
				if chunk.Done && chunk.TokenUsage != nil {
					lastUsage = chunk.TokenUsage
				}
				return writer.WriteChunk(chunk)
			})

			latency := time.Since(startTime).Milliseconds()
			if err != nil {
				writer.WriteError(err.Error())
				app.recordUsage(targetProvider.Name(), targetModel, 502, nil, latency)
			} else {
				app.Logger.Info("streaming finished",
					"model", targetModel,
					"provider", targetProvider.Name(),
					"tokens", lastUsage,
				)
				app.recordUsage(targetProvider.Name(), targetModel, 200, lastUsage, latency)
			}
			return
		}

		// Non-streaming
		resp, err := targetProvider.Chat(ctx, internalReq)
		latency := time.Since(startTime).Milliseconds()
		if err != nil {
			writeError(w, 502, err.Error())
			app.recordUsage(targetProvider.Name(), targetModel, 502, nil, latency)
			return
		}

		app.Logger.Info("model response",
			"model", targetModel,
			"provider", targetProvider.Name(),
			"tokens", resp.TokenUsage,
			"content_snippet", truncateString(resp.Message.Content, 100),
		)

		app.recordUsage(targetProvider.Name(), targetModel, 200, resp.TokenUsage, latency)

		// Convert to Ollama format
		respMsg2 := ollamaMsg{Role: resp.Message.Role, Content: sanitizeAssistantContent(resp.Message.Content), ReasoningContent: resp.Message.ReasoningContent}
		if len(resp.Message.ToolCalls) > 0 {
			respMsg2.ToolCalls = make([]ollamaToolCallItem, len(resp.Message.ToolCalls))
			for j, tc := range resp.Message.ToolCalls {
				respMsg2.ToolCalls[j] = ollamaToolCallItem{
					ID:   tc.ID,
					Type: tc.Type,
					Function: ollamaToolCallFunc{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		ollamaResp := ollamaChatResponse{
			Model:     req.Model,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Message:   respMsg2,
			Done:      true,
		}

		if resp.TokenUsage != nil {
			ollamaResp.PromptEvalCount = resp.TokenUsage.PromptTokens
			ollamaResp.EvalCount = resp.TokenUsage.CompletionTokens
		}

		writeJSON(w, http.StatusOK, ollamaResp)
	}
}

type ollamaChatResponse struct {
	Model           string    `json:"model"`
	CreatedAt       string    `json:"created_at"`
	Message         ollamaMsg `json:"message"`
	Done            bool      `json:"done"`
	TotalDuration   int64     `json:"total_duration,omitempty"`
	PromptEvalCount int       `json:"prompt_eval_count,omitempty"`
	EvalCount       int       `json:"eval_count,omitempty"`
}

func handleGenerate(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model   string                 `json:"model"`
			Prompt  string                 `json:"prompt"`
			System  string                 `json:"system,omitempty"`
			Stream  bool                   `json:"stream"`
			Options map[string]interface{} `json:"options,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body")
			return
		}

		if req.Model == "" {
			writeError(w, 400, "model is required")
			return
		}

		if req.Prompt == "" {
			writeError(w, 400, "prompt is required")
			return
		}

		// Find provider for model (same logic as chat)
		var targetProvider types.Provider
		var targetModel string
		var targetProviderName string
		var targetBaseURL string

		for _, ps := range app.Registry.List() {
			for _, mi := range ps.ModelCache {
				shortName := shortModelName(mi.Name)
				if mi.Name == req.Model || shortName == req.Model {
					targetProvider = ps.Provider
					targetModel = shortName
					targetProviderName = ps.Config.Name
					targetBaseURL = ps.Config.BaseURL
					break
				}
			}
			if targetProvider != nil {
				break
			}
		}

		if targetProvider == nil {
			writeError(w, 404, "model \""+req.Model+"\" not found")
			return
		}

		app.Logger.Info("provider chain",
			"model", req.Model,
			"provider", targetProviderName,
			"provider_type", targetProvider.Type(),
			"target_model", targetModel,
			"base_url", targetBaseURL,
		)

		ctx := r.Context()
		startTime := time.Now()
		internalReq := &types.GenerateRequest{
			Model:  targetModel,
			Prompt: req.Prompt,
			System: req.System,
			Stream: req.Stream,
		}

		if req.Stream {
			w.Header().Set("Content-Type", "application/x-ndjson")
			flusher, ok := w.(http.Flusher)
			if !ok {
				writeError(w, 500, "streaming not supported")
				return
			}
			w.WriteHeader(http.StatusOK)
			flusher.Flush()

			writer := streaming.NewOllamaWriter(w, req.Model)

			var lastUsage *types.TokenUsage
			err := targetProvider.GenerateStream(ctx, internalReq, func(chunk *types.StreamChunk) error {
				if chunk.Done && chunk.TokenUsage != nil {
					lastUsage = chunk.TokenUsage
				}
				return writer.WriteChunk(chunk)
			})

			latency := time.Since(startTime).Milliseconds()
			if err != nil {
				writer.WriteError(err.Error())
				app.recordUsage(targetProvider.Name(), targetModel, 502, nil, latency)
			} else {
				app.Logger.Info("streaming finished",
					"model", targetModel,
					"provider", targetProvider.Name(),
					"tokens", lastUsage,
				)
				app.recordUsage(targetProvider.Name(), targetModel, 200, lastUsage, latency)
			}
			return
		}

		resp, err := targetProvider.Generate(ctx, internalReq)
		latency := time.Since(startTime).Milliseconds()
		if err != nil {
			writeError(w, 502, err.Error())
			app.recordUsage(targetProvider.Name(), targetModel, 502, nil, latency)
			return
		}

		app.Logger.Info("model response",
			"model", targetModel,
			"provider", targetProvider.Name(),
			"tokens", resp.TokenUsage,
			"content_snippet", truncateString(resp.Response, 100),
		)

		app.recordUsage(targetProvider.Name(), targetModel, 200, resp.TokenUsage, latency)

		ollamaResp := map[string]interface{}{
			"model":      req.Model,
			"response":   resp.Response,
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"done":       true,
		}

		writeJSON(w, http.StatusOK, ollamaResp)
	}
}

// ollamaTagDetails mirrors Ollama's exact field order in /api/tags details.
type ollamaTagDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// ollamaTagModel mirrors Ollama's exact field order in /api/tags model entries.
type ollamaTagModel struct {
	Name       string           `json:"name"`
	Model      string           `json:"model"`
	ModifiedAt string           `json:"modified_at"`
	Size       int64            `json:"size"`
	Digest     string           `json:"digest"`
	Details    ollamaTagDetails `json:"details"`
}

// ollamaTagsResponse is the top-level /api/tags response.
type ollamaTagsResponse struct {
	Models []ollamaTagModel `json:"models"`
}

func handleTags(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		models := app.Registry.GetAllModels()

		ollamaModels := make([]ollamaTagModel, len(models))
		for i, m := range models {
			modifiedAt := m.ModifiedAt
			if modifiedAt == "" {
				modifiedAt = time.Now().UTC().Format(time.RFC3339)
			}

			digest := m.Digest
			if digest == "" {
				digest = fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(m.Name)))
			}

			families := m.Details.Families
			if len(families) == 0 && m.Details.Family != "" {
				families = []string{m.Details.Family}
			}

			// Use provider/modelname format for clear provider identification
			fullName := m.Name

			ollamaModels[i] = ollamaTagModel{
				Name:       fullName,
				Model:      fullName,
				ModifiedAt: modifiedAt,
				Size:       m.Size,
				Digest:     digest,
				Details: ollamaTagDetails{
					ParentModel:       m.Details.ParentModel,
					Format:            m.Details.Format,
					Family:            m.Details.Family,
					Families:          families,
					ParameterSize:     m.Details.ParameterSize,
					QuantizationLevel: m.Details.QuantizationLevel,
				},
			}
		}

		writeJSON(w, http.StatusOK, ollamaTagsResponse{Models: ollamaModels})
	}
}

func shortModelName(model string) string {
	slashIdx := strings.Index(model, "/")
	if slashIdx >= 0 && slashIdx+1 < len(model) {
		return model[slashIdx+1:]
	}
	return model
}

func handlePS(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// OllaRouter doesn't "load" models in the same way Ollama does.
		// We return an empty list to satisfy the API.
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"models": []interface{}{},
		})
	}
}

func handleShow(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
			Name  string `json:"name"`
		}

		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, 400, "invalid request body")
				return
			}
		} else {
			req.Model = r.URL.Query().Get("model")
			if req.Model == "" {
				req.Model = r.URL.Query().Get("name")
			}
		}

		modelName := req.Model
		if modelName == "" {
			modelName = req.Name
		}

		// VS Code sends POST /api/show with empty body {} as a discovery ping
		if modelName == "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"license":    "MIT",
				"modelfile":  "",
				"parameters": "",
				"template":   "",
				"details": map[string]interface{}{
					"format":             "",
					"family":             "",
					"parameter_size":     "",
					"quantization_level": "",
				},
				"model_info": map[string]interface{}{
					"general.architecture":       "transformer",
					"transformer.context_length": 32768,
				},
				"capabilities": []string{},
			})
			return
		}

		for _, ps := range app.Registry.List() {
			for _, mi := range ps.ModelCache {
				if mi.Name == modelName || shortModelName(mi.Name) == modelName || mi.Name == ps.Config.Name+"/"+modelName {
					families := mi.Details.Families
					if len(families) == 0 && mi.Details.Family != "" {
						families = []string{mi.Details.Family}
					}

					arch := mi.Details.Family
					if arch == "" {
						arch = "transformer"
					}
					displayName := mi.Name
					if mi.Details.ParameterSize != "" {
						displayName = mi.Name + " (" + mi.Details.ParameterSize + ")"
					}

					contextLen := 32768
					if mi.Details.ParameterSize != "" {
						contextLen = 8192
					}

					capabilities := []string{"tools"}
					if mi.Details.Format == "gguf" {
						capabilities = []string{"tools"}
					}

					details := map[string]interface{}{
						"license":    "MIT",
						"modelfile":  "",
						"parameters": "",
						"template":   "",
						"details": map[string]interface{}{
							"parent_model":       mi.Details.ParentModel,
							"format":             mi.Details.Format,
							"family":             mi.Details.Family,
							"families":           families,
							"parameter_size":     mi.Details.ParameterSize,
							"quantization_level": mi.Details.QuantizationLevel,
						},
						"model_info": map[string]interface{}{
							"general.architecture":   arch,
							"general.basename":       displayName,
							arch + ".context_length": contextLen,
							arch + ".parameters":     mi.Details.ParameterSize,
						},
						"capabilities": capabilities,
						"provider":     ps.Config.Name,
						"modified_at":  mi.ModifiedAt,
						"size":         mi.Size,
					}
					writeJSON(w, http.StatusOK, details)
					return
				}
			}
		}

		writeError(w, 404, "model \""+modelName+"\" not found")
	}
}

// openAIChatRequest is the OpenAI chat completions request format.
type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Stream      bool            `json:"stream"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role             string           `json:"role"`
	Content          interface{}      `json:"content"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIToolCallFunc `json:"function"`
}

type openAIToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "chatcmpl-" + hex.EncodeToString(b)
}

func handleV1ChatCompletions(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body: "+err.Error())
			return
		}

		if req.Model == "" {
			writeError(w, 400, "model is required")
			return
		}

		if len(req.Messages) == 0 {
			writeError(w, 400, "messages is required")
			return
		}

		// Convert to internal format
		messages := make([]types.Message, len(req.Messages))
		for i, m := range req.Messages {
			msg := types.Message{
				Role:             m.Role,
				Content:          contentToString(m.Content),
				ReasoningContent: m.ReasoningContent,
				ToolCallID:       m.ToolCallID,
			}
			if len(m.ToolCalls) > 0 {
				msg.ToolCalls = make([]types.ToolCall, len(m.ToolCalls))
				for j, tc := range m.ToolCalls {
					msg.ToolCalls[j] = types.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
			messages[i] = msg
		}

		tools := make([]types.ToolDefinition, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = types.ToolDefinition{
				Type: t.Type,
				Function: types.ToolFunction{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}

		internalReq := &types.ChatRequest{
			Model:    req.Model,
			Messages: messages,
			Tools:    tools,
			Stream:   req.Stream,
		}

		ctx := r.Context()
		startTime := time.Now()

		// Resolve model (same logic as handleChat)
		var targetProvider types.Provider
		var targetModel string
		var targetProviderName string
		_, isVirtual := app.Engine.ResolveVirtualModel(req.Model)
		if isVirtual == nil {
			if req.Stream {
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				flusher, ok := w.(http.Flusher)
				if !ok {
					writeError(w, 500, "streaming not supported")
					return
				}
				w.WriteHeader(http.StatusOK)
				flusher.Flush()

				id := generateID()
				roleData, _ := json.Marshal(map[string]interface{}{
					"id":      id,
					"object":  "chat.completion.chunk",
					"created": time.Now().Unix(),
					"model":   req.Model,
					"choices": []map[string]interface{}{
						{"index": 0, "delta": map[string]string{"role": "assistant"}, "finish_reason": nil},
					},
				})
				fmt.Fprintf(w, "data: %s\n\n", roleData)

				finalData, _ := json.Marshal(map[string]interface{}{
					"id":      id,
					"object":  "chat.completion.chunk",
					"created": time.Now().Unix(),
					"model":   req.Model,
					"choices": []map[string]interface{}{
						{"index": 0, "delta": map[string]string{}, "finish_reason": "stop"},
					},
				})
				fmt.Fprintf(w, "data: %s\n\n", finalData)
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}

			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":      generateID(),
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   req.Model,
				"choices": []map[string]interface{}{
					{"index": 0, "message": map[string]string{"role": "assistant", "content": ""}, "finish_reason": "stop"},
				},
				"usage": map[string]interface{}{},
			})
			return
		}

		// Not virtual - resolve from registry
		for _, ps := range app.Registry.List() {
			for _, mi := range ps.ModelCache {
				shortName := shortModelName(mi.Name)
				if mi.Name == req.Model || shortName == req.Model {
					targetProvider = ps.Provider
					targetModel = shortName
					targetProviderName = ps.Config.Name
					break
				}
			}
			if targetProvider != nil {
				break
			}
		}

		if targetProvider == nil {
			writeError(w, 404, "model \""+req.Model+"\" not found")
			return
		}

		internalReq.Model = targetModel

		// Streaming
		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, ok := w.(http.Flusher)
			if !ok {
				writeError(w, 500, "streaming not supported")
				return
			}
			w.WriteHeader(http.StatusOK)
			flusher.Flush()

			id := generateID()
			hasRole := false
			hasToolCalls := false
			var finishReason string
			pendingText := ""

			err := targetProvider.ChatStream(ctx, internalReq, func(chunk *types.StreamChunk) error {
				delta := map[string]interface{}{}
				if !hasRole {
					delta["role"] = "assistant"
					hasRole = true
				}
				if len(chunk.ToolCalls) > 0 {
					hasToolCalls = true
					toolCalls := make([]map[string]interface{}, len(chunk.ToolCalls))
					for j, tc := range chunk.ToolCalls {
						toolCalls[j] = map[string]interface{}{
							"index": j,
							"id":    tc.ID,
							"type":  tc.Type,
							"function": map[string]string{
								"name":      tc.Function.Name,
								"arguments": tc.Function.Arguments,
							},
						}
					}
					delta["tool_calls"] = toolCalls
				} else if chunk.Content != "" {
					clean := sanitizeStreamingDelta(&pendingText, chunk.Content, false)
					if clean != "" {
						delta["content"] = clean
					}
				} else if chunk.ReasoningContent != "" {
					delta["reasoning_content"] = chunk.ReasoningContent
				}

				var choiceFinishReason interface{}
				if chunk.Done {
					flushText := sanitizeStreamingDelta(&pendingText, "", true)
					if flushText != "" {
						delta["content"] = flushText
					}
					if hasToolCalls {
						choiceFinishReason = "tool_calls"
					} else {
						choiceFinishReason = "stop"
					}
					finishReason, _ = choiceFinishReason.(string)

					usage := map[string]interface{}{}
					if chunk.TokenUsage != nil {
						usage = map[string]interface{}{
							"prompt_tokens":     chunk.TokenUsage.PromptTokens,
							"completion_tokens": chunk.TokenUsage.CompletionTokens,
							"total_tokens":      chunk.TokenUsage.TotalTokens,
						}
					}

					chunkData, _ := json.Marshal(map[string]interface{}{
						"id":      id,
						"object":  "chat.completion.chunk",
						"created": time.Now().Unix(),
						"model":   req.Model,
						"choices": []map[string]interface{}{
							{"index": 0, "delta": delta, "finish_reason": choiceFinishReason},
						},
						"usage": usage,
					})
					fmt.Fprintf(w, "data: %s\n\n", chunkData)
					fmt.Fprintf(w, "data: [DONE]\n\n")
					flusher.Flush()
					return nil
				}

				if len(delta) == 0 {
					return nil
				}

				chunkData, _ := json.Marshal(map[string]interface{}{
					"id":      id,
					"object":  "chat.completion.chunk",
					"created": time.Now().Unix(),
					"model":   req.Model,
					"choices": []map[string]interface{}{
						{"index": 0, "delta": delta, "finish_reason": nil},
					},
				})
				fmt.Fprintf(w, "data: %s\n\n", chunkData)
				flusher.Flush()
				return nil
			})

			if err != nil {
				if finishReason == "" {
					errData, _ := json.Marshal(map[string]interface{}{
						"id":      id,
						"object":  "chat.completion.chunk",
						"created": time.Now().Unix(),
						"model":   req.Model,
						"choices": []map[string]interface{}{
							{"index": 0, "delta": map[string]string{}, "finish_reason": "error"},
						},
					})
					fmt.Fprintf(w, "data: %s\n\n", errData)
					fmt.Fprintf(w, "data: [DONE]\n\n")
					flusher.Flush()
				}
				app.recordUsage(targetProvider.Name(), targetModel, 502, nil, time.Since(startTime).Milliseconds())
			} else {
				if finishReason == "" {
					// Safety net: provider returned nil but stream was not properly terminated
					finalData, _ := json.Marshal(map[string]interface{}{
						"id":      id,
						"object":  "chat.completion.chunk",
						"created": time.Now().Unix(),
						"model":   req.Model,
						"choices": []map[string]interface{}{
							{"index": 0, "delta": map[string]string{}, "finish_reason": "stop"},
						},
					})
					fmt.Fprintf(w, "data: %s\n\n", finalData)
					fmt.Fprintf(w, "data: [DONE]\n\n")
					flusher.Flush()
				}
				app.recordUsage(targetProvider.Name(), targetModel, 200, nil, time.Since(startTime).Milliseconds())
			}
			return
		}

		// Non-streaming
		resp, err := targetProvider.Chat(ctx, internalReq)
		latency := time.Since(startTime).Milliseconds()
		if err != nil {
			writeError(w, 502, err.Error())
			app.recordUsage(targetProvider.Name(), targetModel, 502, nil, latency)
			return
		}

		// Apply guardrails if enabled for this provider
		if ps, ok := app.Registry.Get(targetProviderName); ok && ps.Config.Guardrails != nil && ps.Config.Guardrails.Enabled {
			resp = applyGuardrails(ctx, ps.Config.Guardrails, resp, internalReq, targetProvider)
		}

		app.recordUsage(targetProvider.Name(), targetModel, 200, resp.TokenUsage, latency)

		usage := map[string]interface{}{}
		if resp.TokenUsage != nil {
			usage = map[string]interface{}{
				"prompt_tokens":     resp.TokenUsage.PromptTokens,
				"completion_tokens": resp.TokenUsage.CompletionTokens,
				"total_tokens":      resp.TokenUsage.TotalTokens,
			}
		}

		content := resp.Message.Content
		toolCalls := resp.Message.ToolCalls

		// Apply mixed output policy: tool_call_first drops text when tool calls present
		if ps, ok := app.Registry.Get(targetProviderName); ok && ps.Config.Guardrails != nil && ps.Config.Guardrails.Enabled {
			if ps.Config.Guardrails.MixedOutputPolicy == "tool_call_first" && len(toolCalls) > 0 {
				content = ""
			}
		}

		msg := map[string]interface{}{
			"role":    resp.Message.Role,
			"content": sanitizeAssistantContent(content),
		}
		if resp.Message.ReasoningContent != "" {
			msg["reasoning_content"] = resp.Message.ReasoningContent
		}
		if len(toolCalls) > 0 {
			tcMaps := make([]map[string]interface{}, len(toolCalls))
			for j, tc := range toolCalls {
				tcMaps[j] = map[string]interface{}{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]string{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
			}
			msg["tool_calls"] = tcMaps
		}

		finishReason := "stop"
		if len(toolCalls) > 0 {
			finishReason = "tool_calls"
		}

		openAIResp := map[string]interface{}{
			"id":      generateID(),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       msg,
					"finish_reason": finishReason,
				},
			},
			"usage": usage,
		}

		writeJSON(w, http.StatusOK, openAIResp)
	}
}

func handleVersion(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"version": FallbackVersion,
		})
	}
}

// applyGuardrails validates and optionally rescues tool calls from a provider response.
// It may retry the request if the tool call output is invalid and guardrails are configured.
func applyGuardrails(ctx context.Context, g *types.GuardrailsConfig, resp *types.ChatResponse, req *types.ChatRequest, provider types.Provider) *types.ChatResponse {
	if resp == nil {
		return resp
	}

	// No tool calls — nothing to validate
	if len(resp.Message.ToolCalls) == 0 && resp.Message.Content == "" {
		return resp
	}

	// Build tool call candidates from the response
	candidates := make([]guardrails.ToolCallCandidate, len(resp.Message.ToolCalls))
	for i, tc := range resp.Message.ToolCalls {
		candidates[i] = guardrails.ToolCallCandidate{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
			Raw:       tc.Function.Name + "(" + tc.Function.Arguments + ")",
		}
	}

	validator := &guardrails.Validator{}
	result := validator.Validate(resp.Message.Content, candidates)

	if result.Valid {
		// Valid tool calls — return as-is
		return resp
	}

	// Invalid tool calls — attempt rescue if enabled
	if g.RescueMalformed && len(candidates) > 0 {
		rescued := guardrails.RescueToolCalls(resp.Message.Content)
		if len(rescued) > 0 {
			content := guardrails.ExtractContent(resp.Message.Content)
			// Convert rescued candidates to proper ToolCall objects
			newToolCalls := make([]types.ToolCall, len(rescued))
			for i, rc := range rescued {
				newToolCalls[i] = types.ToolCall{
					ID:   "call_rescued_" + rc.Name,
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      rc.Name,
						Arguments: rc.Arguments,
					},
				}
			}

			// Validate rescued calls
			rescueResult := validator.Validate(content, rescued)
			if rescueResult.Valid {
				resp.Message.Content = content
				resp.Message.ToolCalls = newToolCalls
				return resp
			}
		}
	}

	// Rescue failed or not enabled — attempt retry if configured
	if g.MaxRetries > 0 {
		r, retryErr := guardrails.RetryLoop(ctx, g.MaxRetries, func(ctx context.Context) (string, error) {
			newResp, err := provider.Chat(ctx, req)
			if err != nil {
				return "", err
			}

			newCandidates := make([]guardrails.ToolCallCandidate, len(newResp.Message.ToolCalls))
			for i, tc := range newResp.Message.ToolCalls {
				newCandidates[i] = guardrails.ToolCallCandidate{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}

			r := validator.Validate(newResp.Message.Content, newCandidates)
			if !r.Valid {
				return "", guardrails.ErrInvalidToolCall
			}

			resp.Message = newResp.Message
			return "ok", nil
		})

		if retryErr == nil && r == "ok" {
			return resp
		}
	}

	// All retries exhausted — return original response with sanitized content
	resp.Message.Content = guardrails.ExtractContent(resp.Message.Content)
	resp.Message.ToolCalls = nil
	return resp
}

// truncateString helper for logging.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
