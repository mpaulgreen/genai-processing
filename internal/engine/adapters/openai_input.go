package adapters

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/errors"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// OpenAIInputAdapter implements the InputAdapter interface for OpenAI models.
// It handles OpenAI-specific input formatting including system/user message format,
// chat completions API structure, and API parameters.
type OpenAIInputAdapter struct {
	// APIKey is the authentication key for OpenAI API
	APIKey string

	// ModelName is the specific OpenAI model to use
	ModelName string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int

	// Temperature controls the randomness of responses (0.0 to 2.0)
	Temperature float64

	// SystemPrompt is the system prompt for OpenShift audit queries
	SystemPrompt string

	// examples are few-shot examples to include in prompt formatting
	examples []types.Example

	// formatter formats prompts using configurable templates when provided
	formatter interfaces.PromptFormatter
}

// OpenAIMessage represents a single message in OpenAI's chat format
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequest represents the request payload for OpenAI Chat Completions API
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

// NewOpenAIInputAdapter creates a new OpenAIInputAdapter with default configuration
func NewOpenAIInputAdapter(apiKey string) *OpenAIInputAdapter {
	return &OpenAIInputAdapter{
		APIKey:      apiKey,
		ModelName:   "gpt-4",
		MaxTokens:   4000,
		Temperature: 0.1,
		// Empty by default; processor wiring should set this from prompts.yaml.
		// Fallbacks are handled by getSystemPromptWithFallback().
		SystemPrompt: "",
	}
}

// AdaptRequest converts an internal request to OpenAI-specific format.
// This method handles the transformation of generic request structures into
// the specific format required by OpenAI API, including system/user message
// formatting and chat completions structure.
func (o *OpenAIInputAdapter) AdaptRequest(req *types.InternalRequest) (*types.ModelRequest, error) {
	if req == nil {
		return nil, errors.NewInputAdapterError(
			"internal request cannot be nil",
			errors.ComponentInputAdapter,
			"openai",
			"openai_input_adapter",
			false,
		)
	}

	// Format the prompt with system/user message format and examples
	formattedPrompt, err := o.FormatPrompt(req.ProcessingRequest.Query, o.examples)
	if err != nil {
		return nil, errors.NewInputAdapterError(
			fmt.Sprintf("failed to format prompt: %v", err),
			errors.ComponentInputAdapter,
			"openai",
			"openai_input_adapter",
			true,
		).WithDetails("query", req.ProcessingRequest.Query)
	}

	// Create OpenAI message structure with system and user messages
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: o.getSystemPromptWithFallback(),
		},
		{
			Role:    "user",
			Content: formattedPrompt,
		},
	}

	// Create OpenAI request payload
	openAIRequest := OpenAIRequest{
		Model:       o.ModelName,
		Messages:    messages,
		MaxTokens:   o.MaxTokens,
		Temperature: o.Temperature,
	}

	// Convert to generic ModelRequest
	modelRequest := &types.ModelRequest{
		Model:      o.ModelName,
		Messages:   []interface{}{openAIRequest},
		Parameters: o.GetAPIParameters(),
	}

	return modelRequest, nil
}

// FormatPrompt formats a prompt string with examples using system/user message format.
// This method handles OpenAI-specific prompt formatting, including system prompt
// integration, few-shot example formatting, and user message structure.
func (o *OpenAIInputAdapter) FormatPrompt(prompt string, examples []types.Example) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.NewInputAdapterError(
			"prompt cannot be empty",
			errors.ComponentInputAdapter,
			"openai",
			"openai_input_adapter",
			false,
		)
	}

	// Prefer formatter when available
	if o.formatter != nil {
		return o.formatter.FormatComplete(o.SystemPrompt, examples, prompt)
	}

	// Build system/user message format
	var builder strings.Builder

	// Add examples to the user message if provided
	if len(examples) > 0 {
		builder.WriteString("Examples:\n\n")
		for i, example := range examples {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("Input: %s\n", example.Input))
			builder.WriteString(fmt.Sprintf("Output: %s\n", example.Output))
		}
		builder.WriteString("\n")
	}

	// Add the main query
	builder.WriteString(fmt.Sprintf("Convert this query to JSON: %s", prompt))

	return builder.String(), nil
}

// GetAPIParameters returns OpenAI-specific API parameters and configuration.
// This method provides the necessary parameters for API communication,
// including authentication headers, endpoint URLs, and provider-specific
// configuration required for successful API calls.
func (o *OpenAIInputAdapter) GetAPIParameters() map[string]interface{} {
	return map[string]interface{}{
		"api_key":      o.APIKey,
		"endpoint":     "https://api.openai.com/v1/chat/completions",
		"method":       "POST",
		"content_type": "application/json",
		"headers": map[string]string{
			"Authorization": "Bearer " + o.APIKey,
			"Content-Type":  "application/json",
		},
		"model_name":  o.ModelName,
		"max_tokens":  o.MaxTokens,
		"temperature": o.Temperature,
		"provider":    "openai",
		"created_at":  time.Now().UTC(),
		"system":      o.SystemPrompt,
	}
}

// ValidateRequest validates the OpenAI-specific request format.
// This method ensures the request meets OpenAI's requirements and constraints.
func (o *OpenAIInputAdapter) ValidateRequest(req *types.ModelRequest) error {
	if req == nil {
		return errors.NewInputAdapterError(
			"model request cannot be nil",
			errors.ComponentInputAdapter,
			"openai",
			"openai_input_adapter",
			false,
		)
	}

	if req.Model == "" {
		return errors.NewInputAdapterError(
			"model name is required",
			errors.ComponentInputAdapter,
			"openai",
			"openai_input_adapter",
			false,
		)
	}

	if len(req.Messages) == 0 {
		return errors.NewInputAdapterError(
			"at least one message is required",
			errors.ComponentInputAdapter,
			"openai",
			"openai_input_adapter",
			false,
		)
	}

	// Validate OpenAI-specific parameters
	if params, ok := req.Parameters["max_tokens"]; ok {
		if maxTokens, ok := params.(int); ok {
			if maxTokens <= 0 || maxTokens > 4096 {
				return errors.NewInputAdapterError(
					fmt.Sprintf("max_tokens must be between 1 and 4096, got %d", maxTokens),
					errors.ComponentInputAdapter,
					"openai",
					"openai_input_adapter",
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
					"openai",
					"openai_input_adapter",
					false,
				)
			}
		}
	}

	return nil
}

// SetModelName sets the OpenAI model name to use
func (o *OpenAIInputAdapter) SetModelName(modelName string) {
	o.ModelName = modelName
}

// SetMaxTokens sets the maximum number of tokens to generate
func (o *OpenAIInputAdapter) SetMaxTokens(maxTokens int) error {
	if maxTokens <= 0 || maxTokens > 4096 {
		return errors.NewInputAdapterError(
			fmt.Sprintf("max_tokens must be between 1 and 4096, got %d", maxTokens),
			errors.ComponentInputAdapter,
			"openai",
			"openai_input_adapter",
			false,
		)
	}
	o.MaxTokens = maxTokens
	return nil
}

// SetTemperature sets the temperature parameter
func (o *OpenAIInputAdapter) SetTemperature(temperature float64) error {
	if temperature < 0.0 || temperature > 2.0 {
		return errors.NewInputAdapterError(
			fmt.Sprintf("temperature must be between 0.0 and 2.0, got %f", temperature),
			errors.ComponentInputAdapter,
			"openai",
			"openai_input_adapter",
			false,
		)
	}
	o.Temperature = temperature
	return nil
}

// SetSystemPrompt sets a custom system prompt
func (o *OpenAIInputAdapter) SetSystemPrompt(prompt string) {
	o.SystemPrompt = prompt
}

// SetExamples sets few-shot examples to include in formatting
func (o *OpenAIInputAdapter) SetExamples(examples []types.Example) {
	o.examples = examples
}

// SetFormatter sets a custom prompt formatter
func (o *OpenAIInputAdapter) SetFormatter(formatter interfaces.PromptFormatter) {
	o.formatter = formatter
}

// getSystemPromptWithFallback returns configured system prompt or a minimal fallback
func (o *OpenAIInputAdapter) getSystemPromptWithFallback() string {
	if strings.TrimSpace(o.SystemPrompt) != "" {
		return o.SystemPrompt
	}
	return "You are an OpenShift audit query specialist. Convert natural language queries into structured JSON parameters for audit log analysis.\n\nAlways respond with valid JSON only. Do not include any markdown formatting, explanations, or additional text outside the JSON structure."
}

// Ensure OpenAIInputAdapter implements the InputAdapter interface
var _ interfaces.InputAdapter = (*OpenAIInputAdapter)(nil)
