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

// GenericInputAdapter implements the InputAdapter interface with a simple prompt format
// suitable for OpenAI-compatible chat APIs.
type GenericInputAdapter struct {
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

// GenericMessage represents a single chat message
type GenericMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewGenericInputAdapter creates a new GenericInputAdapter with sensible defaults
func NewGenericInputAdapter(apiKey string) *GenericInputAdapter {
	return &GenericInputAdapter{
		APIKey:      apiKey,
		ModelName:   "generic-model",
		MaxTokens:   4000,
		Temperature: 0.1,
	}
}

// AdaptRequest converts an internal request to a generic OpenAI-compatible request format
func (g *GenericInputAdapter) AdaptRequest(req *types.InternalRequest) (*types.ModelRequest, error) {
	if req == nil {
		return nil, errors.NewInputAdapterError(
			"internal request cannot be nil",
			errors.ComponentInputAdapter,
			"generic",
			"generic_input_adapter",
			false,
		)
	}

	formattedPrompt, err := g.FormatPrompt(req.ProcessingRequest.Query, g.examples)
	if err != nil {
		return nil, errors.NewInputAdapterError(
			fmt.Sprintf("failed to format prompt: %v", err),
			errors.ComponentInputAdapter,
			"generic",
			"generic_input_adapter",
			true,
		).WithDetails("query", req.ProcessingRequest.Query)
	}

	// For compatibility with providers that expect role/content messages, include a basic message
	messages := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": formattedPrompt,
		},
	}

	modelRequest := &types.ModelRequest{
		Model:      g.ModelName,
		Messages:   messages,
		Parameters: g.GetAPIParameters(),
	}

	// Log prompts if enabled - for temporary debugging
	if os.Getenv("LOG_PROMPTS") == "true" || os.Getenv("LOG_PROMPTS") == "1" {
		log.Printf("[PromptDebug][generic] System prompt:\n%s", g.SystemPrompt)
		log.Printf("[PromptDebug][generic] User prompt:\n%s", formattedPrompt)
	}

	// Log complete raw message if enabled
	if os.Getenv("LOG_PROMPTS") == "true" || os.Getenv("LOG_PROMPTS") == "1" {
		// Create a clean version of the request for logging (without sensitive data)
		logRequest := map[string]interface{}{
			"model": g.ModelName,
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": formattedPrompt,
				},
			},
			"parameters": map[string]interface{}{
				"max_tokens":  g.MaxTokens,
				"temperature": g.Temperature,
				"provider":    "generic",
			},
		}

		// Convert to JSON for pretty logging
		if jsonData, err := json.MarshalIndent(logRequest, "", "  "); err == nil {
			log.Printf("[PromptDebug][generic] Complete raw message sent to Generic API:\n%s", string(jsonData))
		} else {
			log.Printf("[PromptDebug][generic] Complete raw message sent to Generic API (fallback):\nModel: %s\nMessages: %+v\nParameters: %+v",
				g.ModelName, modelRequest.Messages, logRequest["parameters"])
		}
	}

	return modelRequest, nil
}

// FormatPrompt creates a minimal prompt asking the model to return JSON only
func (g *GenericInputAdapter) FormatPrompt(prompt string, examples []types.Example) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.NewInputAdapterError(
			"prompt cannot be empty",
			errors.ComponentInputAdapter,
			"generic",
			"generic_input_adapter",
			false,
		)
	}

	// Prefer formatter when available
	if g.formatter != nil {
		return g.formatter.FormatComplete(g.SystemPrompt, examples, prompt)
	}

	var builder strings.Builder
	// Include system prompt if available
	if strings.TrimSpace(g.SystemPrompt) != "" {
		builder.WriteString(g.SystemPrompt)
		builder.WriteString("\n\n")
	}
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

	return builder.String(), nil
}

// GetAPIParameters returns generic API parameters suitable for OpenAI-compatible APIs
func (g *GenericInputAdapter) GetAPIParameters() map[string]interface{} {
	return map[string]interface{}{
		"api_key": g.APIKey,
		// Endpoint intentionally empty here; the provider decides the final endpoint.
		"endpoint":     "",
		"method":       "POST",
		"content_type": "application/json",
		"headers": map[string]string{
			"Authorization": "Bearer " + g.APIKey,
			"Content-Type":  "application/json",
		},
		"model_name":  g.ModelName,
		"max_tokens":  g.MaxTokens,
		"temperature": g.Temperature,
		"provider":    "generic",
		"created_at":  time.Now().UTC(),
		"system":      g.SystemPrompt,
	}
}

// ValidateRequest validates generic request parameters
func (g *GenericInputAdapter) ValidateRequest(req *types.ModelRequest) error {
	if req == nil {
		return errors.NewInputAdapterError(
			"model request cannot be nil",
			errors.ComponentInputAdapter,
			"generic",
			"generic_input_adapter",
			false,
		)
	}

	if req.Model == "" {
		return errors.NewInputAdapterError(
			"model name is required",
			errors.ComponentInputAdapter,
			"generic",
			"generic_input_adapter",
			false,
		)
	}

	if len(req.Messages) == 0 {
		return errors.NewInputAdapterError(
			"at least one message is required",
			errors.ComponentInputAdapter,
			"generic",
			"generic_input_adapter",
			false,
		)
	}

	if params, ok := req.Parameters["max_tokens"]; ok {
		if maxTokens, ok := params.(int); ok {
			if maxTokens <= 0 || maxTokens > 4096 {
				return errors.NewInputAdapterError(
					fmt.Sprintf("max_tokens must be between 1 and 4096, got %d", maxTokens),
					errors.ComponentInputAdapter,
					"generic",
					"generic_input_adapter",
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
					"generic",
					"generic_input_adapter",
					false,
				)
			}
		}
	}

	return nil
}

// Setters for configuration
func (g *GenericInputAdapter) SetModelName(modelName string) { g.ModelName = modelName }

func (g *GenericInputAdapter) SetMaxTokens(maxTokens int) error {
	if maxTokens <= 0 || maxTokens > 4096 {
		return errors.NewInputAdapterError(
			fmt.Sprintf("max_tokens must be between 1 and 4096, got %d", maxTokens),
			errors.ComponentInputAdapter,
			"generic",
			"generic_input_adapter",
			false,
		)
	}
	g.MaxTokens = maxTokens
	return nil
}

func (g *GenericInputAdapter) SetTemperature(temperature float64) error {
	if temperature < 0.0 || temperature > 2.0 {
		return errors.NewInputAdapterError(
			fmt.Sprintf("temperature must be between 0.0 and 2.0, got %f", temperature),
			errors.ComponentInputAdapter,
			"generic",
			"generic_input_adapter",
			false,
		)
	}
	g.Temperature = temperature
	return nil
}

// Ensure GenericInputAdapter implements the InputAdapter interface
var _ interfaces.InputAdapter = (*GenericInputAdapter)(nil)

// SetExamples sets few-shot examples to include in formatting
func (g *GenericInputAdapter) SetExamples(examples []types.Example) { g.examples = examples }

// SetFormatter sets a custom prompt formatter
func (g *GenericInputAdapter) SetFormatter(formatter interfaces.PromptFormatter) {
	g.formatter = formatter
}

// SetSystemPrompt sets a custom system prompt
func (g *GenericInputAdapter) SetSystemPrompt(prompt string) { g.SystemPrompt = prompt }
