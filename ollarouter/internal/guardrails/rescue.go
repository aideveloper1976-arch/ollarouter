package guardrails

import (
	"encoding/json"
	"strings"
)

// RescueToolCalls attempts to extract tool calls from malformed model output.
// Supports: XML-like <invoke> blocks, fenced JSON blocks, and raw JSON objects.
func RescueToolCalls(raw string) []ToolCallCandidate {
	var candidates []ToolCallCandidate

	candidates = append(candidates, rescueXMLInvokes(raw)...)
	candidates = append(candidates, rescueFencedJSON(raw)...)
	candidates = append(candidates, rescueRawJSON(raw)...)

	return candidates
}

// ExtractContent removes tool call syntax from the text, returning only the natural language content.
func ExtractContent(raw string) string {
	result := raw

	// Remove XML invoke blocks
	result = removeXMLInvokes(result)

	// Remove fenced JSON blocks
	result = removeFencedJSON(result)

	// Remove raw JSON tool call objects
	result = removeRawJSONToolCalls(result)

	return strings.TrimSpace(result)
}

// rescueXMLInvokes parses <invoke name="..."> blocks from the text.
func rescueXMLInvokes(raw string) []ToolCallCandidate {
	var candidates []ToolCallCandidate
	remaining := raw

	for {
		start := strings.Index(remaining, "<invoke")
		if start < 0 {
			break
		}

		end := strings.Index(remaining[start:], "</invoke>")
		if end < 0 {
			break
		}
		end = start + end + len("</invoke>")

		block := remaining[start:end]
		remaining = remaining[end:]

		name := extractInvokeName(block)
		if name == "" {
			continue
		}

		params := extractParameters(block)

		candidates = append(candidates, ToolCallCandidate{
			Name:      name,
			Arguments: params,
			Raw:       block,
		})
	}

	return candidates
}

// extractInvokeName extracts the name attribute from an <invoke> tag.
func extractInvokeName(block string) string {
	// Match <invoke name="...">
	idx := strings.Index(block, `name="`)
	if idx < 0 {
		return ""
	}
	start := idx + len(`name="`)
	end := strings.Index(block[start:], `"`)
	if end < 0 {
		return ""
	}
	return block[start : start+end]
}

// extractParameters extracts parameters from an XML invoke block as JSON.
func extractParameters(block string) string {
	params := make(map[string]string)
	remaining := block

	for {
		start := strings.Index(remaining, "<parameter")
		if start < 0 {
			break
		}

		end := strings.Index(remaining[start:], "</parameter>")
		if end < 0 {
			break
		}
		end = start + end + len("</parameter>")

		paramBlock := remaining[start:end]
		remaining = remaining[end:]

		name := extractParameterName(paramBlock)
		if name == "" {
			continue
		}

		value := extractParameterValue(paramBlock)
		params[name] = value
	}

	if len(params) == 0 {
		return "{}"
	}

	data, _ := json.Marshal(params)
	return string(data)
}

// extractParameterName extracts the name attribute from a <parameter> tag.
func extractParameterName(block string) string {
	idx := strings.Index(block, `name="`)
	if idx < 0 {
		return ""
	}
	start := idx + len(`name="`)
	end := strings.Index(block[start:], `"`)
	if end < 0 {
		return ""
	}
	return block[start : start+end]
}

// extractParameterValue extracts the text content of a <parameter> block.
func extractParameterValue(block string) string {
	start := strings.Index(block, ">")
	if start < 0 {
		return ""
	}
	start++ // skip past >
	closeIdx := strings.LastIndex(block, "</parameter>")
	if closeIdx < 0 {
		return ""
	}
	if start >= closeIdx {
		return ""
	}
	return strings.TrimSpace(block[start:closeIdx])
}

// rescueFencedJSON parses ```json ... ``` blocks.
func rescueFencedJSON(raw string) []ToolCallCandidate {
	var candidates []ToolCallCandidate

	start := strings.Index(raw, "```json")
	if start < 0 {
		return nil
	}
	start += len("```json")

	end := strings.Index(raw[start:], "```")
	if end < 0 {
		return nil
	}

	jsonBlock := strings.TrimSpace(raw[start : start+end])

	// Try parsing as a single object with name + arguments
	var single struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(jsonBlock), &single); err == nil && single.Name != "" {
		args := string(single.Arguments)
		candidates = append(candidates, ToolCallCandidate{
			Name:      single.Name,
			Arguments: args,
			Raw:       jsonBlock,
		})
		return candidates
	}

	// Try parsing as an array
	var arr []struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(jsonBlock), &arr); err == nil {
		for _, item := range arr {
			if item.Name != "" {
				candidates = append(candidates, ToolCallCandidate{
					Name:      item.Name,
					Arguments: string(item.Arguments),
					Raw:       jsonBlock,
				})
			}
		}
	}

	return candidates
}

// rescueRawJSON attempts to parse raw JSON objects containing name+arguments fields.
func rescueRawJSON(raw string) []ToolCallCandidate {
	var candidates []ToolCallCandidate

	// Try to find a JSON object in the text
	braceStart := strings.Index(raw, "{")
	if braceStart < 0 {
		return nil
	}

	// Try parsing the entire text as JSON
	var single struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(raw), &single); err == nil && single.Name != "" {
		candidates = append(candidates, ToolCallCandidate{
			Name:      single.Name,
			Arguments: string(single.Arguments),
			Raw:       raw,
		})
	}

	return candidates
}

// removeXMLInvokes removes <invoke>...</invoke> blocks from text,
// including surrounding whitespace and blank lines.
func removeXMLInvokes(raw string) string {
	result := raw
	for {
		start := strings.Index(result, "<invoke")
		if start < 0 {
			break
		}
		end := strings.Index(result[start:], "</invoke>")
		if end < 0 {
			break
		}
		end = start + end + len("</invoke>")

		// Extend removal to include preceding newline/whitespace
		preStart := start
		for preStart > 0 && (result[preStart-1] == ' ' || result[preStart-1] == '\t' || result[preStart-1] == '\n' || result[preStart-1] == '\r') {
			preStart--
		}
		// Include trailing newline
		postEnd := end
		for postEnd < len(result) && (result[postEnd] == ' ' || result[postEnd] == '\t' || result[postEnd] == '\n' || result[postEnd] == '\r') {
			postEnd++
		}

		result = result[:preStart] + result[postEnd:]
	}
	return strings.TrimSpace(result)
}

// removeFencedJSON removes ```json ... ``` blocks from text.
func removeFencedJSON(raw string) string {
	result := raw
	for {
		start := strings.Index(result, "```json")
		if start < 0 {
			break
		}
		end := strings.Index(result[start+len("```json"):], "```")
		if end < 0 {
			break
		}
		end = start + len("```json") + end + len("```")
		result = result[:start] + result[end:]
	}
	return result
}

// removeRawJSONToolCalls removes top-level JSON objects with name+arguments from text.
func removeRawJSONToolCalls(raw string) string {
	lines := strings.Split(raw, "\n")
	var kept []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check if this line looks like a JSON tool call
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			var check struct {
				Name string `json:"name"`
			}
			if json.Unmarshal([]byte(trimmed), &check) == nil && check.Name != "" {
				continue // skip tool call JSON lines
			}
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
