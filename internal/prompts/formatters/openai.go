package formatters

import (
	"fmt"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// Ensure OpenAIFormatter implements interfaces.PromptFormatter
var _ interfaces.PromptFormatter = (*OpenAIFormatter)(nil)

type OpenAIFormatter struct {
	template      string
	systemMessage string
	userMessage   string
}

func NewOpenAIFormatter(template, systemMessage, userMessage string) *OpenAIFormatter {
	return &OpenAIFormatter{template: template, systemMessage: systemMessage, userMessage: userMessage}
}

func (f *OpenAIFormatter) FormatSystemPrompt(systemPrompt string) (string, error) {
	if strings.TrimSpace(f.systemMessage) == "" {
		return systemPrompt, nil
	}
	out := strings.ReplaceAll(f.systemMessage, "{system_prompt}", systemPrompt)
	out = strings.ReplaceAll(out, "{examples}", "")
	return out, nil
}

func (f *OpenAIFormatter) FormatExamples(examples []types.Example) (string, error) {
	if len(examples) == 0 {
		return "", nil
	}
	var b strings.Builder
	for i, ex := range examples {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("Input: %s\n", ex.Input))
		b.WriteString(fmt.Sprintf("Output: %s\n", ex.Output))
	}
	return b.String(), nil
}

func (f *OpenAIFormatter) FormatComplete(systemPrompt string, examples []types.Example, query string) (string, error) {
	// If no template fields, fallback to simple rendering
	if strings.TrimSpace(f.template) == "" && strings.TrimSpace(f.userMessage) == "" {
		var b strings.Builder
		b.WriteString(systemPrompt)
		if exStr, _ := f.FormatExamples(examples); exStr != "" {
			b.WriteString("\n\nExamples:\n")
			b.WriteString(exStr)
		}
		b.WriteString("\n\nConvert this query to JSON: ")
		b.WriteString(query)
		return b.String(), nil
	}

	exRendered, _ := f.FormatExamples(examples)
	// Prefer userMessage when provided
	if strings.TrimSpace(f.userMessage) != "" {
		out := strings.ReplaceAll(f.userMessage, "{query}", query)
		out = strings.ReplaceAll(out, "{system_prompt}", systemPrompt)
		out = strings.ReplaceAll(out, "{examples}", exRendered)
		return out, nil
	}
	// Else use full template
	out := f.template
	out = strings.ReplaceAll(out, "{system_prompt}", systemPrompt)
	out = strings.ReplaceAll(out, "{examples}", exRendered)
	out = strings.ReplaceAll(out, "{query}", query)
	return out, nil
}
