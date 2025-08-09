package formatters

import (
	"fmt"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// Ensure GenericFormatter implements interfaces.PromptFormatter
var _ interfaces.PromptFormatter = (*GenericFormatter)(nil)

type GenericFormatter struct {
	template string
}

func NewGenericFormatter(template string) *GenericFormatter {
	return &GenericFormatter{template: template}
}

func (f *GenericFormatter) FormatSystemPrompt(systemPrompt string) (string, error) {
	return systemPrompt, nil
}

func (f *GenericFormatter) FormatExamples(examples []types.Example) (string, error) {
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

func (f *GenericFormatter) FormatComplete(systemPrompt string, examples []types.Example, query string) (string, error) {
	if strings.TrimSpace(f.template) == "" {
		var b strings.Builder
		if strings.TrimSpace(systemPrompt) != "" {
			b.WriteString(systemPrompt)
			b.WriteString("\n\n")
		}
		if exStr, _ := f.FormatExamples(examples); exStr != "" {
			b.WriteString("Examples:\n")
			b.WriteString(exStr)
			b.WriteString("\n\n")
		}
		b.WriteString("Query: ")
		b.WriteString(query)
		b.WriteString("\n\nJSON Response:")
		return b.String(), nil
	}
	exRendered, _ := f.FormatExamples(examples)
	out := f.template
	out = strings.ReplaceAll(out, "{system_prompt}", systemPrompt)
	out = strings.ReplaceAll(out, "{examples}", exRendered)
	out = strings.ReplaceAll(out, "{query}", query)
	return out, nil
}
