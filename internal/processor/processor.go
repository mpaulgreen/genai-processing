package processor

import (
	"context"
	"fmt"
	"log"
	"time"

	contextpkg "genai-processing/internal/context"
	"genai-processing/internal/engine/providers"
	"genai-processing/internal/parser/extractors"
	"genai-processing/internal/validator"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// GenAIProcessor implements the complete processing pipeline for natural language
// OpenShift audit queries. It orchestrates all components including context management,
// LLM processing, response parsing, and safety validation.
type GenAIProcessor struct {
	// Dependencies
	contextManager  interfaces.ContextManager
	llmEngine       interfaces.LLMEngine
	parser          interfaces.Parser
	safetyValidator interfaces.SafetyValidator

	// Configuration
	defaultModel string
	logger       *log.Logger
}

// NewGenAIProcessor creates a new instance of GenAIProcessor with all dependencies
// wired up. This constructor initializes the complete processing pipeline.
func NewGenAIProcessor() *GenAIProcessor {
	// Initialize dependencies
	contextManager := contextpkg.NewContextManager()

	// Create Claude provider as the default LLM engine
	claudeProvider := providers.NewClaudeProvider("", "") // API key will be set via config

	// Create parser with Claude extractor
	parser := extractors.NewClaudeExtractor()

	// Create safety validator
	safetyValidator := validator.NewSafetyValidator()

	processor := &GenAIProcessor{
		contextManager:  contextManager,
		llmEngine:       createLLMEngine(claudeProvider),
		parser:          parser,
		safetyValidator: safetyValidator,
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[GenAIProcessor] ", log.LstdFlags),
	}

	return processor
}

// ProcessQuery orchestrates the complete processing pipeline:
// 1. Context resolution using ContextManager
// 2. Input adaptation and LLM processing
// 3. Response parsing
// 4. Safety validation
// 5. Context update
func (p *GenAIProcessor) ProcessQuery(ctx context.Context, req *types.ProcessingRequest) (*types.ProcessingResponse, error) {
	startTime := time.Now()
	p.logger.Printf("Starting query processing for session: %s", req.SessionID)

	// Step 1: Context resolution
	resolvedQuery, err := p.resolveContext(req.Query, req.SessionID)
	if err != nil {
		p.logger.Printf("Context resolution failed: %v", err)
		return p.createErrorResponse("context_resolution_failed", err), nil
	}

	// Step 2: Prepare internal request for LLM processing (for future use)
	_ = &types.InternalRequest{
		RequestID:         generateRequestID(),
		ProcessingRequest: *req,
		ProcessingOptions: map[string]interface{}{
			"model_type":     p.defaultModel,
			"resolved_query": resolvedQuery,
		},
	}

	// Step 3: Get conversation context for LLM
	convContext, err := p.contextManager.GetContext(req.SessionID)
	if err != nil {
		// Create new context if session doesn't exist
		convContext = &types.ConversationContext{
			SessionID:    req.SessionID,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
	}

	// Step 4: LLM processing
	p.logger.Printf("Sending query to LLM: %s", resolvedQuery)
	rawResponse, err := p.llmEngine.ProcessQuery(ctx, resolvedQuery, *convContext)
	if err != nil {
		p.logger.Printf("LLM processing failed: %v", err)
		return p.createErrorResponse("llm_processing_failed", err), nil
	}

	// Step 5: Response parsing
	p.logger.Printf("Parsing LLM response")
	structuredQuery, err := p.parser.ParseResponse(rawResponse, p.defaultModel)
	if err != nil {
		p.logger.Printf("Response parsing failed: %v", err)
		return p.createErrorResponse("parsing_failed", err), nil
	}

	// Step 6: Safety validation
	p.logger.Printf("Validating query safety")
	validationResult, err := p.safetyValidator.ValidateQuery(structuredQuery)
	if err != nil {
		p.logger.Printf("Safety validation failed: %v", err)
		return p.createErrorResponse("validation_failed", err), nil
	}

	// Step 7: Update context with new query/response
	if err := p.contextManager.UpdateContext(req.SessionID, req.Query, structuredQuery); err != nil {
		p.logger.Printf("Context update failed: %v", err)
		// Don't fail the entire request for context update issues
	}

	// Step 8: Create response
	processingTime := time.Since(startTime)
	p.logger.Printf("Query processing completed in %v", processingTime)

	response := &types.ProcessingResponse{
		StructuredQuery: structuredQuery,
		Confidence:      p.parser.GetConfidence(),
		ValidationInfo:  validationResult,
	}

	return response, nil
}

// resolveContext resolves pronouns and references in the query using conversation context
func (p *GenAIProcessor) resolveContext(query, sessionID string) (string, error) {
	p.logger.Printf("Resolving context for query: %s", query)

	resolvedQuery, err := p.contextManager.ResolvePronouns(query, sessionID)
	if err != nil {
		return query, fmt.Errorf("failed to resolve context: %w", err)
	}

	if resolvedQuery != query {
		p.logger.Printf("Query resolved from '%s' to '%s'", query, resolvedQuery)
	}

	return resolvedQuery, nil
}

// createErrorResponse creates a standardized error response
func (p *GenAIProcessor) createErrorResponse(errorType string, err error) *types.ProcessingResponse {
	return &types.ProcessingResponse{
		StructuredQuery: nil,
		Confidence:      0.0,
		ValidationInfo:  nil,
		Error:           fmt.Sprintf("%s: %v", errorType, err),
	}
}

// createLLMEngine creates a simple LLM engine that wraps the Claude provider
func createLLMEngine(provider interfaces.LLMProvider) interfaces.LLMEngine {
	return &simpleLLMEngine{
		provider: provider,
	}
}

// simpleLLMEngine is a basic implementation of the LLMEngine interface
// that wraps a single provider. In a full implementation, this would handle
// multiple providers, load balancing, fallbacks, etc.
type simpleLLMEngine struct {
	provider interfaces.LLMProvider
}

// ProcessQuery implements the LLMEngine interface
func (e *simpleLLMEngine) ProcessQuery(ctx context.Context, query string, context types.ConversationContext) (*types.RawResponse, error) {
	// Create a simple model request
	modelReq := &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": getSystemPrompt(),
			},
			map[string]interface{}{
				"role":    "user",
				"content": query,
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
			"system":      getSystemPrompt(),
		},
	}

	return e.provider.GenerateResponse(ctx, modelReq)
}

// GetSupportedModels returns the list of supported models
func (e *simpleLLMEngine) GetSupportedModels() []string {
	return []string{"claude-3-5-sonnet-20241022"}
}

// AdaptInput adapts an internal request to model-specific format
func (e *simpleLLMEngine) AdaptInput(req *types.InternalRequest) (*types.ModelRequest, error) {
	// For now, return a basic model request
	// In a full implementation, this would use input adapters
	return &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": getSystemPrompt(),
			},
			map[string]interface{}{
				"role":    "user",
				"content": req.ProcessingRequest.Query,
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
			"system":      getSystemPrompt(),
		},
	}, nil
}

// getSystemPrompt returns the system prompt for OpenShift audit queries
func getSystemPrompt() string {
	return `You are an OpenShift audit query specialist. Convert natural language queries into structured JSON parameters for audit log analysis.

Always respond with valid JSON only. Do not include any markdown formatting, explanations, or additional text outside the JSON structure.

The JSON should follow this structure:
{
  "log_source": "kube-apiserver|openshift-apiserver|oauth-server|oauth-apiserver",
  "verb": "get|list|create|update|patch|delete",
  "resource": "pods|services|deployments|namespaces|etc",
  "namespace": "namespace_name",
  "user": "username",
  "timeframe": "today|yesterday|1_hour_ago|7_days_ago",
  "limit": 20,
  "exclude_users": ["system:", "kube-"],
  "resource_name_pattern": "pattern",
  "auth_decision": "allow|error|forbid"
}

Examples:
- "Who deleted the customer CRD yesterday?" → {"log_source": "kube-apiserver", "verb": "delete", "resource": "customresourcedefinitions", "resource_name_pattern": "customer", "timeframe": "yesterday", "exclude_users": ["system:"], "limit": 20}
- "Show me all failed authentication attempts in the last hour" → {"log_source": "oauth-server", "timeframe": "1_hour_ago", "auth_decision": "error", "limit": 20}`
}

// generateRequestID generates a unique request ID for tracking
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
