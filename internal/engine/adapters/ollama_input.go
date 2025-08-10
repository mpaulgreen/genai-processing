package adapters

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/errors"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// OllamaInputAdapter implements the InputAdapter interface for Ollama's API format
// which uses a single "prompt" field instead of OpenAI's "messages" array.
type OllamaInputAdapter struct {
	APIKey      string
	ModelName   string
	MaxTokens   int
	Temperature float64

	// SystemPrompt is the optional system prompt to prepend
	SystemPrompt string

	// examples are few-shot examples to include in prompt formatting
	examples []types.Example

	// formatter formats prompts using configurable templates when provided
	formatter interfaces.PromptFormatter
}

// NewOllamaInputAdapter creates a new OllamaInputAdapter with sensible defaults
func NewOllamaInputAdapter(apiKey string) *OllamaInputAdapter {
	return &OllamaInputAdapter{
		APIKey:      apiKey,
		ModelName:   "llama3.1:8b",
		MaxTokens:   4000,
		Temperature: 0.1,
	}
}

// AdaptRequest converts an internal request to Ollama's API request format
func (o *OllamaInputAdapter) AdaptRequest(req *types.InternalRequest) (*types.ModelRequest, error) {
	if req == nil {
		return nil, errors.NewInputAdapterError(
			"internal request cannot be nil",
			errors.ComponentInputAdapter,
			"ollama",
			"ollama_input_adapter",
			false,
		)
	}

	formattedPrompt, err := o.FormatPrompt(req.ProcessingRequest.Query, o.examples)
	if err != nil {
		return nil, errors.NewInputAdapterError(
			fmt.Sprintf("failed to format prompt: %v", err),
			errors.ComponentInputAdapter,
			"ollama",
			"ollama_input_adapter",
			true,
		).WithDetails("query", req.ProcessingRequest.Query)
	}

	// Create Ollama-specific request structure
	ollamaReq := types.OllamaRequest{
		Model:  o.ModelName,
		Prompt: formattedPrompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": o.Temperature,
			"num_predict": o.MaxTokens,
		},
		Format: "json", // Request JSON output for structured responses
	}

	// Convert to generic ModelRequest format that the provider can handle
	modelRequest := &types.ModelRequest{
		Model:      o.ModelName,
		Messages:   []interface{}{ollamaReq}, // Store Ollama request as a message
		Parameters: o.GetAPIParameters(),
	}

	return modelRequest, nil
}

// FormatPrompt creates a prompt suitable for Ollama models
func (o *OllamaInputAdapter) FormatPrompt(prompt string, examples []types.Example) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.NewInputAdapterError(
			"prompt cannot be empty",
			errors.ComponentInputAdapter,
			"ollama",
			"ollama_input_adapter",
			false,
		)
	}

	// Prefer formatter when available
	if o.formatter != nil {
		return o.formatter.FormatComplete(o.SystemPrompt, examples, prompt)
	}

	var builder strings.Builder

	// Include system prompt if available
	if strings.TrimSpace(o.SystemPrompt) != "" {
		builder.WriteString(o.SystemPrompt)
		builder.WriteString("\n\n")
	}

	// Include examples if available
	if len(examples) > 0 {
		builder.WriteString("Examples:\n\n")
		for i, ex := range examples {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("Input: %s\n", ex.Input))
			builder.WriteString(fmt.Sprintf("Output: %s\n", ex.Output))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("Convert this query to JSON: ")
	builder.WriteString(prompt)
	builder.WriteString("\n\nPlease respond with valid JSON only.")

	return builder.String(), nil
}

// GetAPIParameters returns Ollama-specific API parameters
func (o *OllamaInputAdapter) GetAPIParameters() map[string]interface{} {
	return map[string]interface{}{
		"api_key":      o.APIKey,
		"endpoint":     "",
		"method":       "POST",
		"content_type": "application/json",
		"headers": map[string]string{
			"Content-Type": "application/json",
		},
		"model_name":  o.ModelName,
		"max_tokens":  o.MaxTokens,
		"temperature": o.Temperature,
		"provider":    "ollama",
		"created_at":  time.Now().UTC(),
		"system":      o.SystemPrompt,
		"format":      "json",
	}
}

// ValidateRequest validates Ollama request parameters
func (o *OllamaInputAdapter) ValidateRequest(req *types.ModelRequest) error {
	if req == nil {
		return errors.NewInputAdapterError(
			"model request cannot be nil",
			errors.ComponentInputAdapter,
			"ollama",
			"ollama_input_adapter",
			false,
		)
	}

	if req.Model == "" {
		return errors.NewInputAdapterError(
			"model name is required",
			errors.ComponentInputAdapter,
			"ollama",
			"ollama_input_adapter",
			false,
		)
	}

	if len(req.Messages) == 0 {
		return errors.NewInputAdapterError(
			"at least one message is required",
			errors.ComponentInputAdapter,
			"ollama",
			"ollama_input_adapter",
			false,
		)
	}

	if params, ok := req.Parameters["max_tokens"]; ok {
		if maxTokens, ok := params.(int); ok {
			if maxTokens <= 0 || maxTokens > 8192 {
				return errors.NewInputAdapterError(
					fmt.Sprintf("max_tokens must be between 1 and 8192, got %d", maxTokens),
					errors.ComponentInputAdapter,
					"ollama",
					"ollama_input_adapter",
					false,
				)
			}
		}
	}

	if params, ok := req.Parameters["temperature"]; ok {
		if temp, ok := params.(float64); ok {
			if temp < 0.0 || temp > 2.0 {
				return errors.NewInputAdapterError(
					fmt.Sprintf("temperature must be between 0.0 and 2.0, got %f", temp),
					errors.ComponentInputAdapter,
					"ollama",
					"ollama_input_adapter",
					false,
				)
			}
		}
	}

	return nil
}

// Setters for configuration
func (o *OllamaInputAdapter) SetModelName(modelName string) { o.ModelName = modelName }

func (o *OllamaInputAdapter) SetMaxTokens(maxTokens int) error {
	if maxTokens <= 0 || maxTokens > 8192 {
		return errors.NewInputAdapterError(
			fmt.Sprintf("max_tokens must be between 1 and 8192, got %d", maxTokens),
			errors.ComponentInputAdapter,
			"ollama",
			"ollama_input_adapter",
			false,
		)
	}
	o.MaxTokens = maxTokens
	return nil
}

func (o *OllamaInputAdapter) SetTemperature(temperature float64) error {
	if temperature < 0.0 || temperature > 2.0 {
		return errors.NewInputAdapterError(
			fmt.Sprintf("temperature must be between 0.0 and 2.0, got %f", temperature),
			errors.ComponentInputAdapter,
			"ollama",
			"ollama_input_adapter",
			false,
		)
	}
	o.Temperature = temperature
	return nil
}

// Ensure OllamaInputAdapter implements the InputAdapter interface
var _ interfaces.InputAdapter = (*OllamaInputAdapter)(nil)

// SetExamples sets few-shot examples to include in formatting
func (o *OllamaInputAdapter) SetExamples(examples []types.Example) { o.examples = examples }

// SetFormatter sets a custom prompt formatter
func (o *OllamaInputAdapter) SetFormatter(formatter interfaces.PromptFormatter) {
	o.formatter = formatter
}

// SetSystemPrompt sets a custom system prompt
func (o *OllamaInputAdapter) SetSystemPrompt(prompt string) { o.SystemPrompt = prompt }
