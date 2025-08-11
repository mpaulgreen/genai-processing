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
	norm "genai-processing/internal/parser/normalizers"
	"genai-processing/internal/parser/recovery"
	promptformatters "genai-processing/internal/prompts/formatters"
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

	// Prompt validation settings from prompts.yaml
	promptValidation config.PromptValidation
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

	// Register parsers via extractor factory
	claudeExtractor := extractors.NewClaudeExtractor()
	openaiExtractor := extractors.NewOpenAIExtractor()
	ollamaExtractor := extractors.NewOllamaExtractor()
	genericExtractor := extractors.NewGenericExtractor()

	extractorFactory := extractors.NewExtractorFactory()
	extractorFactory.Register("claude", claudeExtractor, "anthropic")
	extractorFactory.Register("openai", openaiExtractor)
	extractorFactory.Register("ollama", ollamaExtractor, "llama", "llama3", "llama2", "local_llama")
	extractorFactory.SetGeneric(genericExtractor)

	// Specific: delegating parser across model types
	retryParser.RegisterParser(recovery.StrategySpecific, extractorFactory.CreateDelegatingParser())
	// Generic: model-agnostic, regex-based JSON extraction as a universal fallback
	retryParser.RegisterParser(recovery.StrategyGeneric, genericExtractor)

	// Create safety validator
	safetyValidator := validator.NewSafetyValidator()

	// Configure fallback handler for retry parser
	retryParser.SetFallbackHandler(recovery.NewFallbackHandler())

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

	// Helper: choose system prompt from prompts.yaml - now simplified to use base only
	chooseSystemPrompt := func() (string, string) {
		sys := ""
		key := "base"
		if appConfig != nil && appConfig.Prompts.SystemPrompts != nil {
			sys = appConfig.Prompts.SystemPrompts[key]
		}
		return sys, key
	}

	// Examples are already []types.Example in config

	// Helper: select formatter implementation based on mc.PromptFormatter or provider type
	// Falls back gracefully when templates are missing
	makeFormatter := func(providerType string, formatterName string) interfaces.PromptFormatter {
		// Prefer explicit formatterName if provided
		normalized := strings.ToLower(formatterName)
		kind := providerType
		if strings.Contains(normalized, "openai") {
			kind = "openai"
		} else if strings.Contains(normalized, "claude") {
			kind = "claude"
		} else if strings.Contains(normalized, "generic") {
			kind = "generic"
		}
		switch kind {
		case "claude":
			return promptformatters.NewClaudeFormatter(appConfig.Prompts.Formats.Claude.Template)
		case "openai":
			of := appConfig.Prompts.Formats.OpenAI
			return promptformatters.NewOpenAIFormatter(of.Template)
		default:
			return promptformatters.NewGenericFormatter(appConfig.Prompts.Formats.Generic.Template)
		}
	}

	// Build input adapter based on config
	var adapter interfaces.InputAdapter
	switch mc.InputAdapter {
	case "claude_input_adapter":
		claude := adapters.NewClaudeInputAdapter(mc.APIKey)
		claude.SetModelName(mc.ModelName)
		_ = claude.SetMaxTokens(mc.MaxTokens)
		_ = claude.SetTemperature(mc.Temperature)
		// System prompt precedence: models.yaml parameters.system > prompts.yaml
		if sys, ok := activeCfg.Parameters["system"].(string); ok && sys != "" {
			claude.SetSystemPrompt(sys)
			logger.Printf("system prompt: override from models.yaml parameters.system for provider claude")
		} else if sp, key := chooseSystemPrompt(); sp != "" {
			claude.SetSystemPrompt(sp)
			logger.Printf("system prompt: using '%s' for provider claude", key)
		}
		// Wire examples from prompts.yaml
		claude.SetExamples(appConfig.Prompts.Examples)
		// Formatter (mc.PromptFormatter overrides provider default)
		claude.SetFormatter(makeFormatter("claude", mc.PromptFormatter))
		adapter = claude
	case "openai_input_adapter":
		openai := adapters.NewOpenAIInputAdapter(mc.APIKey)
		openai.SetModelName(mc.ModelName)
		_ = openai.SetMaxTokens(mc.MaxTokens)
		_ = openai.SetTemperature(mc.Temperature)
		if sys, ok := activeCfg.Parameters["system"].(string); ok && sys != "" {
			openai.SetSystemPrompt(sys)
			logger.Printf("system prompt: override from models.yaml parameters.system for provider openai")
		} else if sp, key := chooseSystemPrompt(); sp != "" {
			openai.SetSystemPrompt(sp)
			logger.Printf("system prompt: using '%s' for provider openai", key)
		}
		openai.SetExamples(appConfig.Prompts.Examples)
		openai.SetFormatter(makeFormatter("openai", mc.PromptFormatter))
		adapter = openai
	case "ollama_input_adapter":
		ollama := adapters.NewOllamaInputAdapter(mc.APIKey)
		ollama.SetModelName(mc.ModelName)
		_ = ollama.SetMaxTokens(mc.MaxTokens)
		_ = ollama.SetTemperature(mc.Temperature)
		if sys, ok := activeCfg.Parameters["system"].(string); ok && sys != "" {
			ollama.SetSystemPrompt(sys)
			logger.Printf("system prompt: override from models.yaml parameters.system for provider ollama")
		} else if sp, key := chooseSystemPrompt(); sp != "" {
			ollama.SetSystemPrompt(sp)
			logger.Printf("system prompt: using '%s' for provider ollama", key)
		}
		ollama.SetExamples(appConfig.Prompts.Examples)
		ollama.SetFormatter(makeFormatter("generic", mc.PromptFormatter))
		adapter = ollama
	default:
		generic := adapters.NewGenericInputAdapter(mc.APIKey)
		generic.SetModelName(mc.ModelName)
		_ = generic.SetMaxTokens(mc.MaxTokens)
		_ = generic.SetTemperature(mc.Temperature)
		generic.SetExamples(appConfig.Prompts.Examples)
		if sys, ok := activeCfg.Parameters["system"].(string); ok && sys != "" {
			generic.SetSystemPrompt(sys)
			logger.Printf("system prompt: override from models.yaml parameters.system for provider generic")
		} else if sp, key := chooseSystemPrompt(); sp != "" {
			generic.SetSystemPrompt(sp)
			logger.Printf("system prompt: using '%s' for provider generic", key)
		}
		generic.SetFormatter(makeFormatter("generic", mc.PromptFormatter))
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
	// Enforce LLM output length from prompts validation if specified
	if appConfig.Prompts.Validation.MaxOutputLength > 0 {
		retryParser.SetMaxOutputLength(appConfig.Prompts.Validation.MaxOutputLength)
	}

	// Parser preferences based on OutputParser using factory
	claudeExtractor := extractors.NewClaudeExtractor()
	openaiExtractor := extractors.NewOpenAIExtractor()
	ollamaExtractor := extractors.NewOllamaExtractor()
	genericExtractor := extractors.NewGenericExtractor()

	extractorFactory := extractors.NewExtractorFactory()
	extractorFactory.Register("claude", claudeExtractor, "anthropic")
	extractorFactory.Register("openai", openaiExtractor)
	extractorFactory.Register("ollama", ollamaExtractor, "llama", "llama3", "llama2", "local_llama")
	extractorFactory.SetGeneric(genericExtractor)

	switch mc.OutputParser {
	case "claude_extractor":
		retryParser.RegisterParser(recovery.StrategySpecific, claudeExtractor)
	case "openai_extractor":
		retryParser.RegisterParser(recovery.StrategySpecific, openaiExtractor)
	case "ollama_extractor":
		retryParser.RegisterParser(recovery.StrategySpecific, ollamaExtractor)
	default:
		// Delegate by model type via extractor factory
		retryParser.RegisterParser(recovery.StrategySpecific, extractorFactory.CreateDelegatingParser())
	}
	retryParser.RegisterParser(recovery.StrategyGeneric, genericExtractor)

	// Safety validator
	safetyValidator := validator.NewSafetyValidator()

	// Configure fallback handler for retry parser
	retryParser.SetFallbackHandler(recovery.NewFallbackHandler())

	// Log prompt formatter selection (now active via formatters package)
	if mc.PromptFormatter != "" {
		logger.Printf("prompt_formatter active: %s", mc.PromptFormatter)
	}

	proc := &GenAIProcessor{
		contextManager:   contextManager,
		llmEngine:        llmEngine,
		RetryParser:      retryParser,
		safetyValidator:  safetyValidator,
		defaultModel:     mc.ModelName,
		logger:           logger,
		providerTimeout:  mc.Timeout,
		retryAttempts:    mc.RetryAttempts,
		retryDelay:       mc.RetryDelay,
		promptValidation: appConfig.Prompts.Validation,
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
	case "ollama":
		return "ollama"
	case "generic":
		return "generic"
	}
	switch providerName {
	case "anthropic":
		return "claude"
	case "openai":
		return "openai"
	case "ollama":
		return "ollama"
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

	// Step 0: Enforce configured input length from prompts validation (if available)
	// Enforce max input length from prompts validation if configured on processor
	if p.promptValidation.MaxInputLength > 0 && len(req.Query) > p.promptValidation.MaxInputLength {
		req.Query = req.Query[:p.promptValidation.MaxInputLength]
	}

	// Basic forbidden words sanitization
	if len(p.promptValidation.ForbiddenWords) > 0 {
		q := req.Query
		for _, w := range p.promptValidation.ForbiddenWords {
			if w == "" {
				continue
			}
			q = strings.ReplaceAll(q, w, "")
		}
		req.Query = q
	}

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

	// Step 6: Normalization pipeline (JSONNormalizer → FieldMapper → SchemaValidator)
	p.logger.Printf("Normalizing structured query")
	jsonNormalizer := norm.NewJSONNormalizer()
	fieldMapper := norm.NewFieldMapper()
	schemaValidator := norm.NewSchemaValidator()

	if structuredQuery, err = jsonNormalizer.Normalize(structuredQuery); err != nil {
		p.logger.Printf("Normalization (JSON) failed: %v", err)
		return p.createErrorResponse("normalization_failed", err), nil
	}
	if structuredQuery, err = fieldMapper.MapFields(structuredQuery); err != nil {
		p.logger.Printf("Normalization (FieldMapper) failed: %v", err)
		return p.createErrorResponse("normalization_failed", err), nil
	}
	if err = schemaValidator.ValidateSchema(structuredQuery); err != nil {
		p.logger.Printf("Normalization (SchemaValidator) failed: %v", err)
		return p.createErrorResponse("normalization_failed", err), nil
	}

	// Step 7a: Enhanced prompt validation required fields
	if sq := structuredQuery; sq != nil {
		for _, rf := range p.promptValidation.RequiredFields {
			switch strings.ToLower(rf) {
			case "log_source", "logsource":
				if strings.TrimSpace(sq.LogSource) == "" {
					return p.createErrorResponse("validation_failed", fmt.Errorf("required field missing: log_source")), nil
				}
			case "verb":
				if sq.Verb.IsEmpty() {
					return p.createErrorResponse("validation_failed", fmt.Errorf("required field missing: verb")), nil
				}
			case "resource":
				if sq.Resource.IsEmpty() {
					return p.createErrorResponse("validation_failed", fmt.Errorf("required field missing: resource")), nil
				}
			case "timeframe":
				if strings.TrimSpace(sq.Timeframe) == "" {
					return p.createErrorResponse("validation_failed", fmt.Errorf("required field missing: timeframe")), nil
				}
			case "user":
				if sq.User.IsEmpty() {
					return p.createErrorResponse("validation_failed", fmt.Errorf("required field missing: user")), nil
				}
			case "namespace":
				if sq.Namespace.IsEmpty() {
					return p.createErrorResponse("validation_failed", fmt.Errorf("required field missing: namespace")), nil
				}
			case "limit":
				if sq.Limit <= 0 {
					return p.createErrorResponse("validation_failed", fmt.Errorf("required field missing or invalid: limit")), nil
				}
			default:
				p.logger.Printf("warning: unknown required field '%s' in validation config", rf)
			}
		}
	}

	// Step 7b: Safety validation
	p.logger.Printf("Validating query safety")
	validationResult, err := p.safetyValidator.ValidateQuery(structuredQuery)
	if err != nil {
		p.logger.Printf("Safety validation failed: %v", err)
		return p.createErrorResponse("validation_failed", err), nil
	}

	// Step 8: Update context with new query/response, including user identity if available
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

	// Step 9: Create response
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
	// Minimal fallback system prompt for backward compatibility
	fallbackPrompt := "Convert natural language queries to JSON parameters for audit log analysis."
	// Create a simple model request
	modelReq := &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": fallbackPrompt,
			},
			map[string]interface{}{
				"role":    "user",
				"content": query,
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
			"system":      fallbackPrompt,
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
	// For now, return a basic model request with minimal fallback prompt
	fallbackPrompt := "Convert natural language queries to JSON parameters for audit log analysis."
	return &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": fallbackPrompt,
			},
			map[string]interface{}{
				"role":    "user",
				"content": req.ProcessingRequest.Query,
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
			"system":      fallbackPrompt,
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

// multiModelParser was replaced by a factory-backed delegating parser. The prior
// implementation has been removed as unused to satisfy staticcheck.

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
// System prompts are managed via prompts.yaml. This minimal fallback remains only for backward compatibility.
