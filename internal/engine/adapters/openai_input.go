package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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

	// Use formatter from prompts.yaml if available, otherwise fall back to direct construction
	var systemContent, userContent string

	if o.formatter != nil {
		// Use the configured formatter from prompts.yaml
		// The formatter will handle the system_message and user_message templates
		formattedPrompt, err := o.formatter.FormatComplete(o.SystemPrompt, o.examples, req.ProcessingRequest.Query)
		if err != nil {
			return nil, errors.NewInputAdapterError(
				fmt.Sprintf("failed to format prompt: %v", err),
				errors.ComponentInputAdapter,
				"openai",
				"openai_input_adapter",
				true,
			).WithDetails("query", req.ProcessingRequest.Query)
		}

		// For OpenAI, we need to split into system and user messages
		// The formatter returns the complete formatted prompt, so we'll use it as user message
		// and build a minimal system message
		systemContent = o.getSystemPromptWithFallback()
		userContent = formattedPrompt
	} else {
		// Fallback to direct construction (for backward compatibility)
		systemContent = o.buildSystemMessage(o.examples)
		userContent = fmt.Sprintf("Query: %s\n\nJSON:", req.ProcessingRequest.Query)
	}

	// Log prompts if enabled - for temporary debugging
	if os.Getenv("LOG_PROMPTS") == "true" || os.Getenv("LOG_PROMPTS") == "1" {
		log.Printf("[PromptDebug][openai] System prompt:\n%s", systemContent)
		log.Printf("[PromptDebug][openai] User prompt:\n%s", userContent)
	}

	// Convert to generic ModelRequest with role/content messages expected by provider
	modelRequest := &types.ModelRequest{
		Model: o.ModelName,
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": systemContent,
			},
			map[string]interface{}{
				"role":    "user",
				"content": userContent,
			},
		},
		Parameters: o.GetAPIParameters(),
	}

	// Log complete raw message if enabled
	if os.Getenv("LOG_PROMPTS") == "true" || os.Getenv("LOG_PROMPTS") == "1" {
		// Create a clean version of the request for logging (without sensitive data)
		logRequest := map[string]interface{}{
			"model": o.ModelName,
			"messages": []map[string]interface{}{
				{
					"role":    "system",
					"content": systemContent,
				},
				{
					"role":    "user",
					"content": userContent,
				},
			},
			"parameters": map[string]interface{}{
				"max_tokens":  o.MaxTokens,
				"temperature": o.Temperature,
				"provider":    "openai",
			},
		}

		// Convert to JSON for pretty logging
		if jsonData, err := json.MarshalIndent(logRequest, "", "  "); err == nil {
			log.Printf("[PromptDebug][openai] Complete raw message sent to OpenAI:\n%s", string(jsonData))
		} else {
			log.Printf("[PromptDebug][openai] Complete raw message sent to OpenAI (fallback):\nModel: %s\nMessages: %+v\nParameters: %+v",
				o.ModelName, modelRequest.Messages, logRequest["parameters"])
		}
	}

	return modelRequest, nil
}

// FormatPrompt formats a prompt string with examples using system/user message format.
// This method is now simplified since the main formatting is handled in AdaptRequest.
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

	// Simple fallback format for backward compatibility
	return fmt.Sprintf("Query: %s\n\nJSON:", prompt), nil
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

// buildSystemMessage constructs a concise system message tailored for OpenAI models
// with strict JSON-only instructions and embedded few-shot examples.
func (o *OpenAIInputAdapter) buildSystemMessage(examples []types.Example) string {
	var b strings.Builder
	base := o.getSystemPromptWithFallback()
	if strings.TrimSpace(base) == "" {
		base = "You are an OpenShift audit query specialist. Convert natural language queries into structured JSON parameters for audit log analysis."
	}
	b.WriteString(base)
	b.WriteString("\n\nCRITICAL: Respond with ONLY valid JSON. No explanations, no markdown, no code fences, no additional text.\n")
	// Embed a few compact examples to bias output
	if len(examples) > 0 {
		b.WriteString("\nExamples:\n")
		for i, ex := range examples {
			if i >= 3 { // keep concise
				break
			}
			b.WriteString("Query: ")
			b.WriteString(strings.TrimSpace(ex.Input))
			b.WriteString("\n")
			// Use the provided example output directly (already JSON in config)
			b.WriteString(strings.TrimSpace(ex.Output))
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

// Ensure OpenAIInputAdapter implements the InputAdapter interface
var _ interfaces.InputAdapter = (*OpenAIInputAdapter)(nil)
