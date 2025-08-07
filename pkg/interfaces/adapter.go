package interfaces

import (
	"genai-processing/pkg/types"
)

// InputAdapter defines the interface for model-specific input formatting.
// This interface abstracts the conversion of generic internal requests to
// model-specific request formats, handling the unique requirements of different
// LLM providers (Claude, OpenAI, Ollama, etc.).
type InputAdapter interface {
	// AdaptRequest converts an internal request to a model-specific format.
	// This method handles the transformation of generic request structures into
	// the specific format required by the target LLM provider, including
	// message formatting, parameter adaptation, and request structure conversion.
	//
	// Parameters:
	//   - req: The internal request to be adapted for the specific model
	//
	// Returns:
	//   - ModelRequest: The adapted request in the model-specific format
	//   - error: Any error that occurred during adaptation
	AdaptRequest(req *types.InternalRequest) (*types.ModelRequest, error)

	// FormatPrompt formats a prompt string with examples for optimal model performance.
	// This method handles the model-specific prompt formatting, including
	// system prompt integration, few-shot example formatting, and any
	// provider-specific prompt structures (XML for Claude, system/user messages for OpenAI, etc.).
	//
	// Parameters:
	//   - prompt: The base prompt to be formatted
	//   - examples: Few-shot examples to include in the prompt
	//
	// Returns:
	//   - string: The formatted prompt ready for the model
	//   - error: Any error that occurred during formatting
	FormatPrompt(prompt string, examples []types.Example) (string, error)

	// GetAPIParameters returns model-specific API parameters and configuration.
	// This method provides the necessary parameters for API communication,
	// including authentication headers, endpoint URLs, and any provider-specific
	// configuration required for successful API calls.
	//
	// Returns:
	//   - map[string]interface{}: Model-specific API parameters and configuration
	GetAPIParameters() map[string]interface{}
}
