package guardrails

import (
	"encoding/json"
	"strings"
)

// Validator validates tool call output from LLMs.
type Validator struct{}

// Validate checks tool call candidates for validity.
// Returns a ValidationResult indicating overall validity.
func (v *Validator) Validate(content string, candidates []ToolCallCandidate) ValidationResult {
	if len(candidates) == 0 {
		return ValidationResult{
			Valid:   true,
			Content: content,
		}
	}

	validCalls := make([]ToolCallCandidate, 0, len(candidates))
	var errors []string

	for _, c := range candidates {
		if c.Name == "" {
			errors = append(errors, "tool call with empty name")
			continue
		}

		// Validate arguments is valid JSON
		if c.Arguments != "" && !json.Valid([]byte(c.Arguments)) {
			errors = append(errors, "tool call "+c.Name+" has invalid JSON arguments: "+c.Arguments)
			continue
		}

		validCalls = append(validCalls, c)
	}

	if len(errors) > 0 {
		return ValidationResult{
			Valid:     false,
			ToolCalls: validCalls,
			Content:   content,
			Error:     strings.Join(errors, "; "),
		}
	}

	return ValidationResult{
		Valid:     true,
		ToolCalls: validCalls,
		Content:   content,
	}
}
