package formatters

import (
	"fmt"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// Ensure ClaudeFormatter implements interfaces.PromptFormatter
var _ interfaces.PromptFormatter = (*ClaudeFormatter)(nil)

type ClaudeFormatter struct {
	template string
}

func NewClaudeFormatter(template string) *ClaudeFormatter {
	return &ClaudeFormatter{template: template}
}

func (f *ClaudeFormatter) FormatSystemPrompt(systemPrompt string) (string, error) {
	// Identity for Claude; template applies in FormatComplete
	return systemPrompt, nil
}

func (f *ClaudeFormatter) FormatExamples(examples []types.Example) (string, error) {
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

func (f *ClaudeFormatter) FormatComplete(systemPrompt string, examples []types.Example, query string) (string, error) {
	// If no template, fallback to simple XML-style layout
	if strings.TrimSpace(f.template) == "" {
		var b strings.Builder
		b.WriteString("<instructions>\n")
		b.WriteString(systemPrompt)
		b.WriteString("\n</instructions>\n\n")
		if exStr, _ := f.FormatExamples(examples); exStr != "" {
			b.WriteString("<examples>\n")
			b.WriteString(exStr)
			b.WriteString("</examples>\n\n")
		}
		b.WriteString("<query>\n")
		b.WriteString(query)
		b.WriteString("\n</query>\n\nJSON Response:")
		return b.String(), nil
	}

	exRendered, _ := f.FormatExamples(examples)
	out := f.template
	out = strings.ReplaceAll(out, "{system_prompt}", systemPrompt)
	out = strings.ReplaceAll(out, "{examples}", exRendered)
	out = strings.ReplaceAll(out, "{query}", query)
	return out, nil
}
