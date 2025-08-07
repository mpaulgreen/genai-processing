package adapters

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/errors"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// ClaudeInputAdapter implements the InputAdapter interface for Claude models.
// It handles Claude-specific input formatting including XML-style instructions,
// message structure, and API parameters.
type ClaudeInputAdapter struct {
	// APIKey is the authentication key for Claude API
	APIKey string

	// ModelName is the specific Claude model to use
	ModelName string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int

	// Temperature controls the randomness of responses (0.0 to 1.0)
	Temperature float64

	// SystemPrompt is the hard-coded system prompt for OpenShift audit queries
	SystemPrompt string
}

// ClaudeMessage represents a single message in Claude's message format
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest represents the request payload for Claude API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	Messages    []ClaudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	System      string          `json:"system,omitempty"`
}

// NewClaudeInputAdapter creates a new ClaudeInputAdapter with default configuration
func NewClaudeInputAdapter(apiKey string) *ClaudeInputAdapter {
	return &ClaudeInputAdapter{
		APIKey:       apiKey,
		ModelName:    "claude-3-5-sonnet-20241022",
		MaxTokens:    4000,
		Temperature:  0.1,
		SystemPrompt: getDefaultSystemPrompt(),
	}
}

// AdaptRequest converts an internal request to Claude-specific format.
// This method handles the transformation of generic request structures into
// the specific format required by Claude API, including XML-style instructions
// and message formatting.
func (c *ClaudeInputAdapter) AdaptRequest(req *types.InternalRequest) (*types.ModelRequest, error) {
	if req == nil {
		return nil, errors.NewInputAdapterError(
			"internal request cannot be nil",
			errors.ComponentInputAdapter,
			"claude",
			"claude_input_adapter",
			false,
		)
	}

	// Format the prompt with XML-style instructions and examples
	formattedPrompt, err := c.FormatPrompt(req.ProcessingRequest.Query, []types.Example{})
	if err != nil {
		return nil, errors.NewInputAdapterError(
			fmt.Sprintf("failed to format prompt: %v", err),
			errors.ComponentInputAdapter,
			"claude",
			"claude_input_adapter",
			true,
		).WithDetails("query", req.ProcessingRequest.Query)
	}

	// Create Claude message structure
	messages := []ClaudeMessage{
		{
			Role:    "user",
			Content: formattedPrompt,
		},
	}

	// Create Claude request payload
	claudeRequest := ClaudeRequest{
		Model:       c.ModelName,
		Messages:    messages,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
		System:      c.SystemPrompt,
	}

	// Convert to generic ModelRequest
	modelRequest := &types.ModelRequest{
		Model:      c.ModelName,
		Messages:   []interface{}{claudeRequest},
		Parameters: c.GetAPIParameters(),
	}

	return modelRequest, nil
}

// FormatPrompt formats a prompt string with examples using XML-style instructions.
// This method handles Claude-specific prompt formatting, including system prompt
// integration, few-shot example formatting, and XML structure.
func (c *ClaudeInputAdapter) FormatPrompt(prompt string, examples []types.Example) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.NewInputAdapterError(
			"prompt cannot be empty",
			errors.ComponentInputAdapter,
			"claude",
			"claude_input_adapter",
			false,
		)
	}

	// Build XML-style prompt structure
	var builder strings.Builder

	// Add instructions section
	builder.WriteString("<instructions>\n")
	builder.WriteString(c.SystemPrompt)
	builder.WriteString("\n</instructions>\n\n")

	// Add examples section if provided
	if len(examples) > 0 {
		builder.WriteString("<examples>\n")
		for i, example := range examples {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("Input: %s\n", example.Input))
			builder.WriteString(fmt.Sprintf("Output: %s\n", example.Output))
		}
		builder.WriteString("</examples>\n\n")
	}

	// Add query section
	builder.WriteString("<query>\n")
	builder.WriteString(prompt)
	builder.WriteString("\n</query>\n\n")

	// Add response instruction
	builder.WriteString("JSON Response:\n")

	return builder.String(), nil
}

// GetAPIParameters returns Claude-specific API parameters and configuration.
// This method provides the necessary parameters for API communication,
// including authentication headers, endpoint URLs, and provider-specific
// configuration required for successful API calls.
func (c *ClaudeInputAdapter) GetAPIParameters() map[string]interface{} {
	return map[string]interface{}{
		"api_key":      c.APIKey,
		"endpoint":     "https://api.anthropic.com/v1/messages",
		"method":       "POST",
		"content_type": "application/json",
		"headers": map[string]string{
			"x-api-key":         c.APIKey,
			"anthropic-version": "2023-06-01",
			"content-type":      "application/json",
		},
		"model_name":  c.ModelName,
		"max_tokens":  c.MaxTokens,
		"temperature": c.Temperature,
		"provider":    "anthropic",
		"created_at":  time.Now().UTC(),
	}
}

// ValidateRequest validates the Claude-specific request format.
// This method ensures the request meets Claude's requirements and constraints.
func (c *ClaudeInputAdapter) ValidateRequest(req *types.ModelRequest) error {
	if req == nil {
		return errors.NewInputAdapterError(
			"model request cannot be nil",
			errors.ComponentInputAdapter,
			"claude",
			"claude_input_adapter",
			false,
		)
	}

	if req.Model == "" {
		return errors.NewInputAdapterError(
			"model name is required",
			errors.ComponentInputAdapter,
			"claude",
			"claude_input_adapter",
			false,
		)
	}

	if len(req.Messages) == 0 {
		return errors.NewInputAdapterError(
			"at least one message is required",
			errors.ComponentInputAdapter,
			"claude",
			"claude_input_adapter",
			false,
		)
	}

	// Validate Claude-specific parameters
	if params, ok := req.Parameters["max_tokens"]; ok {
		if maxTokens, ok := params.(int); ok {
			if maxTokens <= 0 || maxTokens > 4096 {
				return errors.NewInputAdapterError(
					fmt.Sprintf("max_tokens must be between 1 and 4096, got %d", maxTokens),
					errors.ComponentInputAdapter,
					"claude",
					"claude_input_adapter",
					false,
				)
			}
		}
	}

	if params, ok := req.Parameters["temperature"]; ok {
		if temp, ok := params.(float64); ok {
			if temp < 0.0 || temp > 1.0 {
				return errors.NewInputAdapterError(
					fmt.Sprintf("temperature must be between 0.0 and 1.0, got %f", temp),
					errors.ComponentInputAdapter,
					"claude",
					"claude_input_adapter",
					false,
				)
			}
		}
	}

	return nil
}

// SetModelName sets the Claude model name to use
func (c *ClaudeInputAdapter) SetModelName(modelName string) {
	c.ModelName = modelName
}

// SetMaxTokens sets the maximum number of tokens to generate
func (c *ClaudeInputAdapter) SetMaxTokens(maxTokens int) error {
	if maxTokens <= 0 || maxTokens > 4096 {
		return errors.NewInputAdapterError(
			fmt.Sprintf("max_tokens must be between 1 and 4096, got %d", maxTokens),
			errors.ComponentInputAdapter,
			"claude",
			"claude_input_adapter",
			false,
		)
	}
	c.MaxTokens = maxTokens
	return nil
}

// SetTemperature sets the temperature parameter
func (c *ClaudeInputAdapter) SetTemperature(temperature float64) error {
	if temperature < 0.0 || temperature > 1.0 {
		return errors.NewInputAdapterError(
			fmt.Sprintf("temperature must be between 0.0 and 1.0, got %f", temperature),
			errors.ComponentInputAdapter,
			"claude",
			"claude_input_adapter",
			false,
		)
	}
	c.Temperature = temperature
	return nil
}

// SetSystemPrompt sets a custom system prompt
func (c *ClaudeInputAdapter) SetSystemPrompt(prompt string) {
	c.SystemPrompt = prompt
}

// getDefaultSystemPrompt returns the hard-coded system prompt for OpenShift audit queries
func getDefaultSystemPrompt() string {
	return `You are an OpenShift audit query specialist. Your task is to convert natural language queries into structured JSON parameters for audit log analysis.

You must respond with valid JSON only. Do not include any markdown formatting, explanations, or additional text outside the JSON structure.

The JSON response should follow this schema:
{
  "log_source": "string or array",           // Source of audit logs (kube-apiserver, oauth-server, etc.)
  "verb": "string or array",                 // HTTP verb(s) to filter on
  "resource": "string or array",             // Kubernetes resource type(s)
  "namespace": "string or array",            // Specific namespace(s) to filter
  "user": "string or array",                 // Specific user(s) to filter
  "timeframe": "string",                     // Time period (today, yesterday, 1_hour_ago, etc.)
  "limit": "number",                         // Maximum number of results to return
  "response_status": "string or array",      // HTTP response status filter
  "exclude_users": ["array of strings"],     // User patterns to exclude
  "resource_name_pattern": "string",         // Regex pattern for resource name matching
  "user_pattern": "string",                  // Regex pattern for user matching
  "namespace_pattern": "string",             // Regex pattern for namespace matching
  "request_uri_pattern": "string",           // URI pattern matching
  "auth_decision": "string",                 // Authentication decision filter
  "source_ip": "string or array",            // Source IP address filtering
  "group_by": "string or array",             // Fields to group results by
  "sort_by": "string",                       // Sort field
  "sort_order": "string",                    // Sort direction (asc, desc)
  "time_range": {                            // Custom time range
    "start": "string (ISO 8601)",
    "end": "string (ISO 8601)"
  },
  "business_hours": {                        // Business hours filtering
    "outside_only": "boolean",
    "start_hour": "number (0-23)",
    "end_hour": "number (0-23)"
  },
  "analysis": {                              // Analysis configuration
    "type": "string",
    "group_by": "string or array",
    "threshold": "number"
  }
}

Key guidelines:
1. Always include "log_source" field
2. Use "kube-apiserver" for Kubernetes API operations
3. Use "oauth-server" for authentication events
4. Exclude system users by default using "exclude_users": ["system:", "kube-"]
5. Set reasonable limits (default 20, max 1000)
6. Use appropriate timeframes for security investigations
7. Include pattern matching for flexible resource name filtering

Examples of common patterns:
- CRD operations: resource="customresourcedefinitions"
- Pod operations: resource="pods"
- Authentication failures: log_source="oauth-server", auth_decision="error"
- Privilege escalation: verb=["create", "update", "patch"], resource=["roles", "rolebindings"]
- Data exfiltration: verb="get", resource=["secrets", "configmaps"]`
}

// Ensure ClaudeInputAdapter implements the InputAdapter interface
var _ interfaces.InputAdapter = (*ClaudeInputAdapter)(nil)
