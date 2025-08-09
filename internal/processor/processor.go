package processor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"genai-processing/internal/config"
	contextpkg "genai-processing/internal/context"
	"genai-processing/internal/engine"
	"genai-processing/internal/engine/adapters"
	"genai-processing/internal/engine/providers"
	"genai-processing/internal/parser/extractors"
	"genai-processing/internal/parser/recovery"
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
	RetryParser     *recovery.RetryParser
	safetyValidator interfaces.SafetyValidator

	// Configuration
	defaultModel string
	logger       *log.Logger

	// Provider execution controls
	providerTimeout time.Duration
	retryAttempts   int
	retryDelay      time.Duration
}

// NewGenAIProcessorWithDeps creates a new instance of GenAIProcessor with injected dependencies.
// This constructor is primarily used for testing with mock components.
func NewGenAIProcessorWithDeps(
	contextManager interfaces.ContextManager,
	llmEngine interfaces.LLMEngine,
	retryParser *recovery.RetryParser,
	safetyValidator interfaces.SafetyValidator,
) *GenAIProcessor {
	return &GenAIProcessor{
		contextManager:  contextManager,
		llmEngine:       llmEngine,
		RetryParser:     retryParser,
		safetyValidator: safetyValidator,
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[GenAIProcessor] ", log.LstdFlags),
	}
}

// NewGenAIProcessor creates a new instance of GenAIProcessor with all dependencies
// wired up. This constructor initializes the complete processing pipeline.
func NewGenAIProcessor() *GenAIProcessor {
	// Initialize dependencies
	contextManager := contextpkg.NewContextManager()

	// Create Claude provider as the default LLM engine
	claudeProvider := providers.NewClaudeProvider("", "") // API key will be set via config
	llmEngine := createLLMEngine(claudeProvider)

	// Create and configure RetryParser with multiple parsing strategies
	retryConfig := &recovery.RetryConfig{
		MaxRetries:          3,
		RetryDelay:          time.Second * 2,
		ConfidenceThreshold: 0.7,
		EnableReprompting:   true,
		RepromptTemplate:    "The previous response was not in the expected JSON format. Please provide a valid JSON response for the following query: %s",
	}

	retryParser := recovery.NewRetryParser(retryConfig, llmEngine, contextManager)

	// Register parsers for different strategies
	claudeExtractor := extractors.NewClaudeExtractor()
	openaiExtractor := extractors.NewOpenAIExtractor()
	genericExtractor := extractors.NewGenericExtractor()

	// Combine Claude and OpenAI into a single "specific" parser that selects by modelType
	specificParser := &multiModelParser{
		claude: claudeExtractor,
		openai: openaiExtractor,
	}

	// Specific: model-specific parsing (Claude or OpenAI)
	retryParser.RegisterParser(recovery.StrategySpecific, specificParser)
	// Generic: model-agnostic, regex-based JSON extraction as a universal fallback
	retryParser.RegisterParser(recovery.StrategyGeneric, genericExtractor)

	// Create safety validator
	safetyValidator := validator.NewSafetyValidator()

	return NewGenAIProcessorWithDeps(
		contextManager,
		llmEngine,
		retryParser,
		safetyValidator,
	)
}

// NewGenAIProcessorFromConfig creates a new GenAIProcessor using the provided AppConfig.
// It wires providers, adapters, engine, retry parser, and validator based on models.yaml.
func NewGenAIProcessorFromConfig(appConfig *config.AppConfig) (*GenAIProcessor, error) {
	if appConfig == nil {
		return nil, fmt.Errorf("appConfig cannot be nil")
	}
	if v := appConfig.Validate(); !v.Valid {
		return nil, fmt.Errorf("invalid configuration: %v", v.Errors)
	}

	logger := log.New(log.Writer(), "[GenAIProcessor] ", log.LstdFlags)

	// Initialize context manager
	contextManager := contextpkg.NewContextManager()

	// Build provider factory and map configs
	factory := providers.NewProviderFactory()

	// Helper to convert ModelConfig.Parameters (map[string]string) to map[string]interface{}
	toIfaceParams := func(mc config.ModelConfig) map[string]interface{} {
		result := map[string]interface{}{}
		if mc.Parameters != nil {
			for k, v := range mc.Parameters {
				// Try to parse int
				var intVal int
				var floatVal float64
				if _, err := fmt.Sscanf(v, "%d", &intVal); err == nil {
					result[k] = intVal
					continue
				}
				if _, err := fmt.Sscanf(v, "%f", &floatVal); err == nil {
					result[k] = floatVal
					continue
				}
				// fallback to string
				result[k] = v
			}
		}
		// Ensure core numeric params present from top-level too
		if mc.MaxTokens > 0 {
			result["max_tokens"] = mc.MaxTokens
		}
		if mc.Temperature >= 0 {
			result["temperature"] = mc.Temperature
		}
		return result
	}

	// Register all providers (best-effort; duplicates for 'generic' may overwrite)
	for name, mc := range appConfig.Models.Providers {
		cfg := &types.ProviderConfig{
			APIKey:     mc.APIKey,
			Endpoint:   mc.Endpoint,
			ModelName:  mc.ModelName,
			Parameters: toIfaceParams(mc),
		}

		providerType := mapProviderType(name, mc.Provider)
		if err := factory.RegisterProvider(providerType, cfg); err != nil {
			// Log and continue to allow others to register
			logger.Printf("warning: failed to register provider '%s' as '%s': %v", name, providerType, err)
		}
	}

	// Select active provider
	defaultKey := appConfig.Models.DefaultProvider
	mc, ok := appConfig.Models.Providers[defaultKey]
	if !ok {
		return nil, fmt.Errorf("default_provider '%s' not found in providers", defaultKey)
	}
	activeCfg := &types.ProviderConfig{
		APIKey:     mc.APIKey,
		Endpoint:   mc.Endpoint,
		ModelName:  mc.ModelName,
		Parameters: toIfaceParams(mc),
	}
	providerType := mapProviderType(defaultKey, mc.Provider)

	// Create concrete provider from the selected config
	provider, err := factory.CreateProviderWithConfig(providerType, activeCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider '%s': %w", providerType, err)
	}

	// Build input adapter based on config
	var adapter interfaces.InputAdapter
	switch mc.InputAdapter {
	case "claude_input_adapter":
		claude := adapters.NewClaudeInputAdapter(mc.APIKey)
		claude.SetModelName(mc.ModelName)
		_ = claude.SetMaxTokens(mc.MaxTokens)
		_ = claude.SetTemperature(mc.Temperature)
		if sys, ok := activeCfg.Parameters["system"].(string); ok && sys != "" {
			claude.SetSystemPrompt(sys)
		}
		adapter = claude
	case "openai_input_adapter":
		openai := adapters.NewOpenAIInputAdapter(mc.APIKey)
		openai.SetModelName(mc.ModelName)
		_ = openai.SetMaxTokens(mc.MaxTokens)
		_ = openai.SetTemperature(mc.Temperature)
		if sys, ok := activeCfg.Parameters["system"].(string); ok && sys != "" {
			openai.SetSystemPrompt(sys)
		}
		adapter = openai
	default:
		generic := adapters.NewGenericInputAdapter(mc.APIKey)
		generic.SetModelName(mc.ModelName)
		_ = generic.SetMaxTokens(mc.MaxTokens)
		_ = generic.SetTemperature(mc.Temperature)
		// Generic adapter has no SetSystemPrompt
		adapter = generic
	}

	// Create LLM engine
	llmEngine := engine.NewLLMEngine(provider, adapter)

	// Retry parser configuration derives from model retry settings
	retryConfig := &recovery.RetryConfig{
		MaxRetries:          mc.RetryAttempts,
		RetryDelay:          mc.RetryDelay,
		ConfidenceThreshold: 0.7,
		EnableReprompting:   true,
		RepromptTemplate:    "The previous response was not in the expected JSON format. Please provide a valid JSON response for the following query: %s",
	}
	retryParser := recovery.NewRetryParser(retryConfig, llmEngine, contextManager)

	// Parser preferences based on OutputParser
	claudeExtractor := extractors.NewClaudeExtractor()
	openaiExtractor := extractors.NewOpenAIExtractor()
	genericExtractor := extractors.NewGenericExtractor()

	switch mc.OutputParser {
	case "claude_extractor":
		retryParser.RegisterParser(recovery.StrategySpecific, claudeExtractor)
	case "openai_extractor":
		retryParser.RegisterParser(recovery.StrategySpecific, openaiExtractor)
	default:
		// Keep previous behavior: delegate by model type
		specific := &multiModelParser{claude: claudeExtractor, openai: openaiExtractor}
		retryParser.RegisterParser(recovery.StrategySpecific, specific)
	}
	retryParser.RegisterParser(recovery.StrategyGeneric, genericExtractor)

	// Safety validator
	safetyValidator := validator.NewSafetyValidator()

	// Log prompt formatter selection for future handling
	if mc.PromptFormatter != "" {
		logger.Printf("prompt_formatter: %s (TODO: integrate custom formatting)", mc.PromptFormatter)
	}

	proc := &GenAIProcessor{
		contextManager:  contextManager,
		llmEngine:       llmEngine,
		RetryParser:     retryParser,
		safetyValidator: safetyValidator,
		defaultModel:    mc.ModelName,
		logger:          logger,
		providerTimeout: mc.Timeout,
		retryAttempts:   mc.RetryAttempts,
		retryDelay:      mc.RetryDelay,
	}

	return proc, nil
}

// mapProviderType maps config provider name/key to factory provider type
func mapProviderType(key string, providerName string) string {
	switch key {
	case "claude":
		return "claude"
	case "openai":
		return "openai"
	case "generic":
		return "generic"
	}
	switch providerName {
	case "anthropic":
		return "claude"
	case "openai":
		return "openai"
	default:
		return "generic"
	}
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

	// Step 2: Prepare internal request for LLM processing via input adapters
	internalReq := &types.InternalRequest{
		RequestID: fmt.Sprintf("%s-%d", req.SessionID, time.Now().UnixNano()),
		ProcessingRequest: types.ProcessingRequest{
			Query:     resolvedQuery,
			SessionID: req.SessionID,
			ModelType: req.ModelType,
		},
		ProcessingOptions: map[string]interface{}{
			"original_query": req.Query,
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

	// Step 4: Adapt input using engine's adapter and send to provider
	p.logger.Printf("Adapting input via LLM engine adapter")
	modelReq, err := p.llmEngine.AdaptInput(internalReq)
	if err != nil {
		p.logger.Printf("Input adaptation failed: %v", err)
		return p.createErrorResponse("input_adaptation_failed", err), nil
	}

	p.logger.Printf("Sending adapted request to LLM provider")
	// Prefer direct provider call if engine exposes provider; otherwise, fall back to existing ProcessQuery path
	var rawResponse *types.RawResponse
	type engineWithProvider interface {
		GetProvider() interfaces.LLMProvider
	}
	if ep, ok := p.llmEngine.(engineWithProvider); ok {
		provider := ep.GetProvider()

		// Apply timeout and retry logic using configuration values
		attempts := p.retryAttempts
		if attempts < 0 {
			attempts = 0
		}
		var lastErr error
		for attempt := 0; attempt <= attempts; attempt++ {
			callCtx := ctx
			if p.providerTimeout > 0 {
				var cancel context.CancelFunc
				callCtx, cancel = context.WithTimeout(ctx, p.providerTimeout)
				defer cancel()
			}

			rawResponse, err = provider.GenerateResponse(callCtx, modelReq)
			if err == nil {
				break
			}
			lastErr = err

			// Decide retry
			if attempt < attempts && isTransientError(err) {
				p.logger.Printf("Transient provider error (attempt %d/%d): %v", attempt+1, attempts+1, err)
				td := p.retryDelay
				if td < 0 {
					td = 0
				}
				timer := time.NewTimer(td)
				select {
				case <-ctx.Done():
					p.logger.Printf("Context canceled during retry wait: %v", ctx.Err())
					return p.createErrorResponse("llm_processing_failed", ctx.Err()), nil
				case <-timer.C:
				}
				continue
			}

			// Non-retryable or out of attempts
			p.logger.Printf("Provider call failed: %v", err)
			return p.createErrorResponse("llm_processing_failed", err), nil
		}
		if lastErr != nil && rawResponse == nil {
			p.logger.Printf("Provider call failed after retries: %v", lastErr)
			return p.createErrorResponse("llm_processing_failed", lastErr), nil
		}
	} else {
		// Backward compatibility: if engine cannot send ModelRequest directly, use existing ProcessQuery
		p.logger.Printf("Engine does not expose provider send; using fallback ProcessQuery path")
		rawResponse, err = p.llmEngine.ProcessQuery(ctx, resolvedQuery, *convContext)
		if err != nil {
			p.logger.Printf("LLM processing failed: %v", err)
			return p.createErrorResponse("llm_processing_failed", err), nil
		}
	}

	// Step 5: Response parsing with retry mechanism
	p.logger.Printf("Parsing LLM response with retry mechanism")
	structuredQuery, err := p.RetryParser.ParseWithRetry(ctx, rawResponse, p.defaultModel, req.Query, req.SessionID)
	if err != nil {
		p.logger.Printf("Response parsing failed after retries: %v", err)
		return p.createErrorResponse("parsing_failed", err), nil
	}

	// Step 6: Safety validation
	p.logger.Printf("Validating query safety")
	validationResult, err := p.safetyValidator.ValidateQuery(structuredQuery)
	if err != nil {
		p.logger.Printf("Safety validation failed: %v", err)
		return p.createErrorResponse("validation_failed", err), nil
	}

	// Step 7: Update context with new query/response, including user identity if available
	if userID, ok := ctx.Value(types.ContextKeyUserID).(string); ok && userID != "" {
		_ = p.contextManager.UpdateContextWithUser(req.SessionID, userID, req.Query, structuredQuery)
	} else {
		if convContext != nil && convContext.UserID != "" {
			_ = p.contextManager.UpdateContextWithUser(req.SessionID, convContext.UserID, req.Query, structuredQuery)
		} else {
			if err := p.contextManager.UpdateContext(req.SessionID, req.Query, structuredQuery); err != nil {
				p.logger.Printf("Context update failed: %v", err)
			}
		}
	}
	// Don't fail the entire request for context update issues

	// Step 8: Create response
	processingTime := time.Since(startTime)
	p.logger.Printf("Query processing completed in %v", processingTime)

	// Get confidence from retry parser statistics or use default
	confidence := 0.8 // Default confidence for successful parsing
	if stats := p.RetryParser.GetRetryStatistics(); stats != nil {
		if threshold, ok := stats["confidence_threshold"].(float64); ok {
			confidence = threshold
		}
	}

	response := &types.ProcessingResponse{
		StructuredQuery: structuredQuery,
		Confidence:      confidence,
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

// GetProvider exposes the underlying provider for direct calls when required
func (e *simpleLLMEngine) GetProvider() interfaces.LLMProvider {
	return e.provider
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

// ValidateConnection checks if the provider connection is working
func (e *simpleLLMEngine) ValidateConnection() error {
	// Check if provider supports connection validation
	if validator, ok := e.provider.(interface{ ValidateConnection() error }); ok {
		return validator.ValidateConnection()
	}

	// Fallback: try a simple test request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testReq := &types.ModelRequest{
		Model: "test",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "test",
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens": 1,
		},
	}

	_, err := e.provider.GenerateResponse(ctx, testReq)
	if err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	return nil
}

// multiModelParser delegates to Claude or OpenAI extractors based on modelType
type multiModelParser struct {
	claude interfaces.Parser
	openai interfaces.Parser
}

func (m *multiModelParser) ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error) {
	// Prefer a model-specific parser if it CanHandle the given modelType
	if m.claude != nil && m.claude.CanHandle(modelType) {
		return m.claude.ParseResponse(raw, modelType)
	}
	if m.openai != nil && m.openai.CanHandle(modelType) {
		return m.openai.ParseResponse(raw, modelType)
	}
	// Default to OpenAI parser if modelType is unknown (it has robust JSON handling)
	if m.openai != nil {
		return m.openai.ParseResponse(raw, modelType)
	}
	// Fallback to Claude if OpenAI unavailable
	if m.claude != nil {
		return m.claude.ParseResponse(raw, modelType)
	}
	return nil, fmt.Errorf("no underlying parser available")
}

func (m *multiModelParser) CanHandle(modelType string) bool {
	if m.claude != nil && m.claude.CanHandle(modelType) {
		return true
	}
	if m.openai != nil && m.openai.CanHandle(modelType) {
		return true
	}
	// Accept unknown to allow delegation in ParseResponse
	return true
}

func (m *multiModelParser) GetConfidence() float64 {
	// Return a conservative default
	return 0.8
}

// isTransientError determines whether a provider error is likely transient
// based on common network/timeouts and HTTP semantics embedded in error text.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Network/timeout categories
	transientSnippets := []string{
		"timeout", "deadline exceeded", "temporarily unavailable", "temporary", "try again",
		"connection reset", "connection refused", "no such host", "tls handshake timeout", "eof",
		"rate limit", "too many requests", "429", "503", "502", "504",
	}
	for _, s := range transientSnippets {
		if strings.Contains(msg, s) {
			return true
		}
	}
	// Some net errors have Timeout() bool
	var nerr interface{ Timeout() bool }
	if ok := errors.As(err, &nerr); ok && nerr.Timeout() {
		return true
	}
	return false
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
Examples:
- "Who deleted the customer CRD yesterday?" → {"log_source": "kube-apiserver", "verb": "delete", "resource": "customresourcedefinitions", "resource_name_pattern": "customer", "timeframe": "yesterday", "exclude_users": ["system:"], "limit": 20}
- "Show me all failed authentication attempts in the last hour" → {"log_source": "oauth-server", "timeframe": "1_hour_ago", "auth_decision": "error", "limit": 20}`
}
