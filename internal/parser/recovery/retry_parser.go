package recovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/errors"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// RetryStrategy defines the different parsing strategies to try when parsing fails.
type RetryStrategy string

const (
	// StrategySpecific uses model-specific parsing with detailed extraction
	StrategySpecific RetryStrategy = "specific"
	// StrategyGeneric uses generic parsing with basic JSON extraction
	StrategyGeneric RetryStrategy = "generic"
	// StrategyError uses error recovery with minimal parsing
	StrategyError RetryStrategy = "error"
)

// RetryConfig contains configuration for retry behavior.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `json:"max_retries"`
	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration `json:"retry_delay"`
	// ConfidenceThreshold is the minimum confidence required for success
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	// EnableReprompting enables re-prompting for ambiguous responses
	EnableReprompting bool `json:"enable_reprompting"`
	// RepromptTemplate is the template for re-prompting
	RepromptTemplate string `json:"reprompt_template"`
}

// RetryResult contains the result of a retry attempt.
type RetryResult struct {
	// Success indicates whether the retry was successful
	Success bool `json:"success"`
	// Query contains the parsed structured query if successful
	Query *types.StructuredQuery `json:"query,omitempty"`
	// Strategy used for this retry attempt
	Strategy RetryStrategy `json:"strategy"`
	// Confidence score of the parsing attempt
	Confidence float64 `json:"confidence"`
	// Error contains error details if the retry failed
	Error *errors.ParsingError `json:"error,omitempty"`
	// AttemptNumber is the number of this retry attempt
	AttemptNumber int `json:"attempt_number"`
	// Duration is the time taken for this retry attempt
	Duration time.Duration `json:"duration"`
}

// RetryParser implements retry logic for handling parsing failures.
// It provides multiple parsing strategies and fallback mechanisms.
type RetryParser struct {
	// config contains retry configuration
	config *RetryConfig
	// parsers contains available parsers for different strategies
	parsers map[RetryStrategy]interfaces.Parser
	// llmEngine is used for re-prompting when needed
	llmEngine interfaces.LLMEngine
	// contextManager is used for context-aware re-prompting
	contextManager interfaces.ContextManager
}

// NewRetryParser creates a new RetryParser with the given configuration.
func NewRetryParser(config *RetryConfig, llmEngine interfaces.LLMEngine, contextManager interfaces.ContextManager) *RetryParser {
	if config == nil {
		config = &RetryConfig{
			MaxRetries:          3,
			RetryDelay:          time.Second * 2,
			ConfidenceThreshold: 0.7,
			EnableReprompting:   true,
			RepromptTemplate:    "The previous response was not in the expected JSON format. Please provide a valid JSON response for the following query: %s",
		}
	}

	return &RetryParser{
		config:         config,
		parsers:        make(map[RetryStrategy]interfaces.Parser),
		llmEngine:      llmEngine,
		contextManager: contextManager,
	}
}

// RegisterParser registers a parser for a specific retry strategy.
func (r *RetryParser) RegisterParser(strategy RetryStrategy, parser interfaces.Parser) {
	r.parsers[strategy] = parser
}

// ParseWithRetry attempts to parse a response using multiple strategies with retry logic.
// It implements a fallback chain: specific → generic → error.
func (r *RetryParser) ParseWithRetry(ctx context.Context, raw *types.RawResponse, modelType string, originalQuery string, sessionID string) (*types.StructuredQuery, error) {
	if raw == nil {
		return nil, errors.NewParsingError("raw response is nil", errors.ComponentParser, "retry_parser", 0.0, "")
	}

	var lastError *errors.ParsingError
	var bestResult *RetryResult

	// Define the retry strategy chain
	strategies := []RetryStrategy{StrategySpecific, StrategyGeneric, StrategyError}

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		for _, strategy := range strategies {
			result := r.tryParseWithStrategy(ctx, raw, modelType, strategy, attempt, originalQuery, sessionID)

			// Update best result if this attempt is better
			if result.Success && (bestResult == nil || result.Confidence > bestResult.Confidence) {
				bestResult = result
			}

			// If we have a successful result above threshold, return it
			if result.Success && result.Confidence >= r.config.ConfidenceThreshold {
				return result.Query, nil
			}

			// Store the last error for reporting
			if result.Error != nil {
				lastError = result.Error
			}

			// If this is a successful result but below threshold, try re-prompting
			if result.Success && result.Confidence < r.config.ConfidenceThreshold && r.config.EnableReprompting {
				if repromptResult := r.tryReprompting(ctx, raw, modelType, originalQuery, sessionID, result); repromptResult != nil {
					if repromptResult.Success && repromptResult.Confidence >= r.config.ConfidenceThreshold {
						return repromptResult.Query, nil
					}
					if repromptResult.Confidence > bestResult.Confidence {
						bestResult = repromptResult
					}
				}
			}
		}

		// If this is not the last attempt, wait before retrying
		if attempt < r.config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(r.config.RetryDelay):
				continue
			}
		}
	}

	// If we have a best result, return it even if below threshold
	if bestResult != nil && bestResult.Success {
		return bestResult.Query, nil
	}

	// Return the last error or create a comprehensive error
	if lastError != nil {
		return nil, lastError
	}

	return nil, errors.NewParsingError(
		"all parsing strategies failed after maximum retries",
		errors.ComponentParser,
		"retry_parser",
		0.0,
		raw.Content,
	).WithDetails("max_retries", r.config.MaxRetries).
		WithDetails("strategies_tried", strategies).
		WithSuggestions(
			"Check if the model response format is correct",
			"Try using a different parser",
			"Verify the response contains valid JSON",
			"Review the model's output format",
		)
}

// tryParseWithStrategy attempts to parse using a specific strategy.
func (r *RetryParser) tryParseWithStrategy(_ context.Context, raw *types.RawResponse, modelType string, strategy RetryStrategy, attempt int, _ string, _ string) *RetryResult {
	startTime := time.Now()
	result := &RetryResult{
		Strategy:      strategy,
		AttemptNumber: attempt,
		Success:       false,
		Confidence:    0.0,
	}

	parser, exists := r.parsers[strategy]
	if !exists {
		result.Error = errors.NewParsingError(
			fmt.Sprintf("no parser registered for strategy: %s", strategy),
			errors.ComponentParser,
			"retry_parser",
			0.0,
			raw.Content,
		)
		result.Duration = time.Since(startTime)
		return result
	}

	// Check if parser can handle this model type
	if !parser.CanHandle(modelType) && strategy != StrategyGeneric && strategy != StrategyError {
		result.Error = errors.NewParsingError(
			fmt.Sprintf("parser cannot handle model type: %s", modelType),
			errors.ComponentParser,
			"retry_parser",
			0.0,
			raw.Content,
		)
		result.Duration = time.Since(startTime)
		return result
	}

	// Attempt parsing
	query, err := parser.ParseResponse(raw, modelType)
	if err != nil {
		parsingErr := errors.NewParsingError(
			fmt.Sprintf("parsing failed with strategy %s: %v", strategy, err),
			errors.ComponentParser,
			"retry_parser",
			0.0,
			raw.Content,
		)
		parsingErr.WithDetails("strategy", string(strategy)).
			WithDetails("attempt", attempt)
		result.Error = parsingErr
		result.Duration = time.Since(startTime)
		return result
	}

	// Get confidence from parser
	confidence := parser.GetConfidence()

	result.Success = true
	result.Query = query
	result.Confidence = confidence
	result.Duration = time.Since(startTime)

	return result
}

// tryReprompting attempts to re-prompt the model for a clearer response.
func (r *RetryParser) tryReprompting(ctx context.Context, raw *types.RawResponse, modelType string, originalQuery string, sessionID string, lastResult *RetryResult) *RetryResult {
	if !r.config.EnableReprompting || r.llmEngine == nil {
		return nil
	}

	startTime := time.Now()
	result := &RetryResult{
		Strategy:      "reprompt",
		AttemptNumber: lastResult.AttemptNumber,
		Success:       false,
		Confidence:    0.0,
	}

	// Create re-prompt message
	repromptMessage := fmt.Sprintf(r.config.RepromptTemplate, originalQuery)

	// Get conversation context for re-prompting
	var context *types.ConversationContext
	if r.contextManager != nil {
		if ctx, err := r.contextManager.GetContext(sessionID); err == nil {
			context = ctx
		}
	}

	// Generate new response with re-prompt
	var newRaw *types.RawResponse
	var err error
	if context != nil {
		newRaw, err = r.llmEngine.ProcessQuery(ctx, repromptMessage, *context)
	} else {
		// Create empty context if none available
		emptyContext := types.ConversationContext{}
		newRaw, err = r.llmEngine.ProcessQuery(ctx, repromptMessage, emptyContext)
	}

	if err != nil {
		parsingErr := errors.NewParsingError(
			fmt.Sprintf("re-prompting failed: %v", err),
			errors.ComponentParser,
			"retry_parser",
			0.0,
			raw.Content,
		)
		parsingErr.WithDetails("reprompt_message", repromptMessage)
		result.Error = parsingErr
		result.Duration = time.Since(startTime)
		return result
	}

	// Try parsing the new response with the same strategy that had low confidence
	parser, exists := r.parsers[lastResult.Strategy]
	if !exists {
		result.Error = errors.NewParsingError(
			fmt.Sprintf("no parser available for reprompting with strategy: %s", lastResult.Strategy),
			errors.ComponentParser,
			"retry_parser",
			0.0,
			newRaw.Content,
		)
		result.Duration = time.Since(startTime)
		return result
	}

	// Parse the re-prompted response
	query, err := parser.ParseResponse(newRaw, modelType)
	if err != nil {
		parsingErr := errors.NewParsingError(
			fmt.Sprintf("parsing re-prompted response failed: %v", err),
			errors.ComponentParser,
			"retry_parser",
			0.0,
			newRaw.Content,
		)
		parsingErr.WithDetails("original_confidence", lastResult.Confidence)
		result.Error = parsingErr
		result.Duration = time.Since(startTime)
		return result
	}

	confidence := parser.GetConfidence()

	result.Success = true
	result.Query = query
	result.Confidence = confidence
	result.Duration = time.Since(startTime)

	return result
}

// GetRetryStatistics returns statistics about retry attempts.
func (r *RetryParser) GetRetryStatistics() map[string]interface{} {
	return map[string]interface{}{
		"max_retries":           r.config.MaxRetries,
		"retry_delay":           r.config.RetryDelay.String(),
		"confidence_threshold":  r.config.ConfidenceThreshold,
		"enable_reprompting":    r.config.EnableReprompting,
		"registered_strategies": r.getRegisteredStrategies(),
	}
}

// getRegisteredStrategies returns a list of registered parsing strategies.
func (r *RetryParser) getRegisteredStrategies() []string {
	strategies := make([]string, 0, len(r.parsers))
	for strategy := range r.parsers {
		strategies = append(strategies, string(strategy))
	}
	return strategies
}

// ValidateConfiguration validates the retry configuration.
func (r *RetryParser) ValidateConfiguration() error {
	if r.config.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	if r.config.RetryDelay < 0 {
		return fmt.Errorf("retry_delay must be non-negative")
	}
	if r.config.ConfidenceThreshold < 0.0 || r.config.ConfidenceThreshold > 1.0 {
		return fmt.Errorf("confidence_threshold must be between 0.0 and 1.0")
	}
	if r.config.EnableReprompting && r.config.RepromptTemplate == "" {
		return fmt.Errorf("reprompt_template is required when reprompting is enabled")
	}
	return nil
}

// SetConfiguration updates the retry configuration.
func (r *RetryParser) SetConfiguration(config *RetryConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	// Validate the new configuration
	tempParser := &RetryParser{config: config}
	if err := tempParser.ValidateConfiguration(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	r.config = config
	return nil
}

// GetConfiguration returns the current retry configuration.
func (r *RetryParser) GetConfiguration() *RetryConfig {
	return r.config
}

// IsRecoverableError determines if an error is recoverable through retry.
func (r *RetryParser) IsRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a parsing error
	if parsingErr, ok := err.(*errors.ParsingError); ok {
		// Consider parsing errors recoverable if they have low confidence
		return parsingErr.Confidence < r.config.ConfidenceThreshold
	}

	// Check if it's a processing error
	if processingErr, ok := err.(*errors.ProcessingError); ok {
		return processingErr.Recoverable
	}

	// Check for specific error patterns that indicate recoverable issues
	errorMsg := strings.ToLower(err.Error())
	recoverablePatterns := []string{
		"json",
		"parsing",
		"format",
		"malformed",
		"invalid",
		"unexpected",
		"timeout",
		"temporary",
		"retry",
	}

	for _, pattern := range recoverablePatterns {
		if strings.Contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}

// CreateFallbackQuery creates a minimal fallback query when all parsing strategies fail.
func (r *RetryParser) CreateFallbackQuery(raw *types.RawResponse, modelType string) *types.StructuredQuery {
	// Create a minimal query with basic information
	fallbackQuery := &types.StructuredQuery{
		LogSource: "kube-apiserver", // Default to most common source
		Limit:     20,               // Conservative limit
	}

	// Try to extract any useful information from the raw content
	content := raw.Content
	if content != "" {
		// Look for common patterns in the content
		if strings.Contains(strings.ToLower(content), "oauth") {
			fallbackQuery.LogSource = "oauth-server"
		} else if strings.Contains(strings.ToLower(content), "openshift") {
			fallbackQuery.LogSource = "openshift-apiserver"
		}

		// Try to extract timeframe information
		if strings.Contains(content, "today") {
			fallbackQuery.Timeframe = "today"
		} else if strings.Contains(content, "yesterday") {
			fallbackQuery.Timeframe = "yesterday"
		} else if strings.Contains(content, "hour") {
			fallbackQuery.Timeframe = "1_hour_ago"
		}
	}

	return fallbackQuery
}
