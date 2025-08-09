package interfaces

import (
	"genai-processing/pkg/types"
)

// PromptFormatter defines an interface for formatting prompts using
// configurable templates for different providers/models.
//
// Implementations should safely handle missing/empty templates by
// falling back to identity formatting.
type PromptFormatter interface {
	// FormatSystemPrompt formats the system prompt section.
	FormatSystemPrompt(systemPrompt string) (string, error)

	// FormatExamples formats the few-shot examples section.
	FormatExamples(examples []types.Example) (string, error)

	// FormatComplete composes the final prompt (typically the user message)
	// from system prompt, examples, and the current query.
	FormatComplete(systemPrompt string, examples []types.Example, query string) (string, error)
}
