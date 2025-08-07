package interfaces

import (
	"context"
	"genai-processing/pkg/types"
)

// LLMEngine defines the interface for the main LLM processing engine.
// This interface orchestrates the complete natural language to structured query conversion process,
// coordinating between different LLM providers, input adapters, and output parsers.
type LLMEngine interface {
	// ProcessQuery processes a natural language query and converts it to structured format.
	// It handles the complete pipeline from input reception to validated output generation.
	// The method coordinates context management, model selection, input adaptation,
	// LLM processing, response parsing, and safety validation.
	//
	// Parameters:
	//   - ctx: Context for request cancellation and timeout management
	//   - query: The natural language query to be processed
	//   - context: Conversation context for multi-turn interactions
	//
	// Returns:
	//   - RawResponse: The raw response from the LLM provider
	//   - error: Any error that occurred during processing
	ProcessQuery(ctx context.Context, query string, context types.ConversationContext) (*types.RawResponse, error)

	// GetSupportedModels returns a list of all supported LLM models.
	// This method provides information about available models for client applications
	// to make informed decisions about model selection.
	//
	// Returns:
	//   - []string: List of supported model identifiers
	GetSupportedModels() []string

	// AdaptInput adapts an internal request to a model-specific format.
	// This method handles the conversion of generic internal requests to
	// model-specific request formats using appropriate input adapters.
	//
	// Parameters:
	//   - req: The internal request to be adapted
	//
	// Returns:
	//   - ModelRequest: The adapted model-specific request
	//   - error: Any error that occurred during adaptation
	AdaptInput(req *types.InternalRequest) (*types.ModelRequest, error)
}

// LLMProvider defines the interface for individual LLM provider implementations.
// This interface abstracts the specific details of different LLM APIs (Claude, OpenAI, Ollama, etc.)
// and provides a consistent interface for the engine to interact with various providers.
type LLMProvider interface {
	// GenerateResponse sends a request to the LLM provider and returns the raw response.
	// This method handles the actual API communication with the underlying LLM service,
	// including authentication, request formatting, and response handling.
	//
	// Parameters:
	//   - ctx: Context for request cancellation and timeout management
	//   - request: The model-specific request to send to the provider
	//
	// Returns:
	//   - RawResponse: The raw response from the LLM provider
	//   - error: Any error that occurred during the API call
	GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error)

	// GetModelInfo returns detailed information about the model provided by this provider.
	// This method provides metadata about the model's capabilities, version, and configuration
	// for monitoring, debugging, and client information purposes.
	//
	// Returns:
	//   - ModelInfo: Detailed information about the model
	GetModelInfo() types.ModelInfo

	// SupportsStreaming indicates whether this provider supports streaming responses.
	// Streaming allows for real-time response generation, which can improve user experience
	// for long-running queries by providing partial results as they become available.
	//
	// Returns:
	//   - bool: True if streaming is supported, false otherwise
	SupportsStreaming() bool
}
