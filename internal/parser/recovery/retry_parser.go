package recovery

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
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

// RetryMetrics tracks operational statistics for the retry parser.
type RetryMetrics struct {
	// Counters
	TotalAttempts     int64 `json:"total_attempts"`
	SuccessfulRetries int64 `json:"successful_retries"`
	FailedRetries     int64 `json:"failed_retries"`
	FallbackUsed      int64 `json:"fallback_used"`
	CircuitBreakerOpen int64 `json:"circuit_breaker_open"`

	// Strategy Usage
	StrategyUsage       map[RetryStrategy]int64   `json:"strategy_usage"`
	StrategySuccessRate map[RetryStrategy]float64 `json:"strategy_success_rate"`

	// Performance Metrics
	TotalDuration    time.Duration `json:"total_duration"`
	AverageLatency   time.Duration `json:"average_latency"`
	latencyHistory   []time.Duration // for percentile calculations
	P95Latency       time.Duration `json:"p95_latency"`
	P99Latency       time.Duration `json:"p99_latency"`

	// Error Patterns
	ErrorTypes        map[string]int64 `json:"error_types"`
	ModelTypeFailures map[string]int64 `json:"model_type_failures"`

	// Time-based metrics
	RequestsPerSecond float64   `json:"requests_per_second"`
	LastResetTime     time.Time `json:"last_reset_time"`
	StartTime         time.Time `json:"start_time"`

	// Synchronization
	mutex sync.RWMutex
}

// CircuitBreakerState represents the current state of the circuit breaker.
type CircuitBreakerState int

const (
	// StateClosed - Normal operation, requests pass through
	StateClosed CircuitBreakerState = iota
	// StateOpen - Requests fail fast, no calls made
	StateOpen
	// StateHalfOpen - Test requests allowed to check recovery
	StateHalfOpen
)

// String returns the string representation of the circuit breaker state.
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures required to open the circuit
	FailureThreshold int `json:"failure_threshold"`
	// RecoveryTimeout is the time to wait before transitioning to half-open
	RecoveryTimeout time.Duration `json:"recovery_timeout"`
	// RequestVolumeThreshold is the minimum number of requests before evaluation
	RequestVolumeThreshold int `json:"request_volume_threshold"`
	// SuccessThreshold is the number of successes needed to close the circuit
	SuccessThreshold int `json:"success_threshold"`
}

// CircuitBreaker tracks failures and manages state transitions.
type CircuitBreaker struct {
	config       *CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	successes    int
	requests     int
	lastFailTime time.Time
	mutex        sync.RWMutex
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

	// maxOutputLength, when >0, enforces a maximum raw response length before parsing
	maxOutputLength int

	// optional fallback handler to create a minimal query when all strategies fail
	fallbackHandler interfaces.FallbackHandler

	// metrics tracks operational statistics
	metrics *RetryMetrics
	// circuitBreaker prevents cascading failures
	circuitBreaker *CircuitBreaker
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
		metrics:        NewRetryMetrics(),
		circuitBreaker: NewCircuitBreaker(nil),
	}
}

// SetFallbackHandler sets a handler used to create a minimal structured query
// if all parsing attempts fail.
func (r *RetryParser) SetFallbackHandler(h interfaces.FallbackHandler) {
	r.fallbackHandler = h
}

// RegisterParser registers a parser for a specific retry strategy.
func (r *RetryParser) RegisterParser(strategy RetryStrategy, parser interfaces.Parser) {
	r.parsers[strategy] = parser
}

// NewRetryMetrics creates a new RetryMetrics instance.
func NewRetryMetrics() *RetryMetrics {
	now := time.Now()
	return &RetryMetrics{
		StrategyUsage:       make(map[RetryStrategy]int64),
		StrategySuccessRate: make(map[RetryStrategy]float64),
		ErrorTypes:          make(map[string]int64),
		ModelTypeFailures:   make(map[string]int64),
		latencyHistory:      make([]time.Duration, 0, 1000), // keep last 1000 latencies
		LastResetTime:       now,
		StartTime:           now,
	}
}

// NewCircuitBreaker creates a new CircuitBreaker with the given configuration.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			FailureThreshold:       5,
			RecoveryTimeout:        time.Second * 30,
			RequestVolumeThreshold: 10,
			SuccessThreshold:       3,
		}
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// SetCircuitBreakerConfig updates the circuit breaker configuration.
func (r *RetryParser) SetCircuitBreakerConfig(config *CircuitBreakerConfig) {
	if config != nil {
		r.circuitBreaker.config = config
	}
}

// GetMetrics returns a copy of the current metrics.
func (r *RetryParser) GetMetrics() *RetryMetrics {
	r.metrics.mutex.RLock()
	defer r.metrics.mutex.RUnlock()

	// Create a deep copy of metrics
	metricsCopy := &RetryMetrics{
		TotalAttempts:       r.metrics.TotalAttempts,
		SuccessfulRetries:   r.metrics.SuccessfulRetries,
		FailedRetries:       r.metrics.FailedRetries,
		FallbackUsed:        r.metrics.FallbackUsed,
		CircuitBreakerOpen:  r.metrics.CircuitBreakerOpen,
		TotalDuration:       r.metrics.TotalDuration,
		AverageLatency:      r.metrics.AverageLatency,
		P95Latency:          r.metrics.P95Latency,
		P99Latency:          r.metrics.P99Latency,
		RequestsPerSecond:   r.metrics.RequestsPerSecond,
		LastResetTime:       r.metrics.LastResetTime,
		StartTime:           r.metrics.StartTime,
		StrategyUsage:       make(map[RetryStrategy]int64),
		StrategySuccessRate: make(map[RetryStrategy]float64),
		ErrorTypes:          make(map[string]int64),
		ModelTypeFailures:   make(map[string]int64),
	}

	// Copy maps
	for k, v := range r.metrics.StrategyUsage {
		metricsCopy.StrategyUsage[k] = v
	}
	for k, v := range r.metrics.StrategySuccessRate {
		metricsCopy.StrategySuccessRate[k] = v
	}
	for k, v := range r.metrics.ErrorTypes {
		metricsCopy.ErrorTypes[k] = v
	}
	for k, v := range r.metrics.ModelTypeFailures {
		metricsCopy.ModelTypeFailures[k] = v
	}

	return metricsCopy
}

// ResetMetrics resets all metrics to zero.
func (r *RetryParser) ResetMetrics() {
	r.metrics.mutex.Lock()
	defer r.metrics.mutex.Unlock()

	now := time.Now()
	r.metrics.TotalAttempts = 0
	r.metrics.SuccessfulRetries = 0
	r.metrics.FailedRetries = 0
	r.metrics.FallbackUsed = 0
	r.metrics.CircuitBreakerOpen = 0
	r.metrics.TotalDuration = 0
	r.metrics.AverageLatency = 0
	r.metrics.P95Latency = 0
	r.metrics.P99Latency = 0
	r.metrics.RequestsPerSecond = 0
	r.metrics.LastResetTime = now
	r.metrics.latencyHistory = r.metrics.latencyHistory[:0]

	// Clear maps
	for k := range r.metrics.StrategyUsage {
		delete(r.metrics.StrategyUsage, k)
	}
	for k := range r.metrics.StrategySuccessRate {
		delete(r.metrics.StrategySuccessRate, k)
	}
	for k := range r.metrics.ErrorTypes {
		delete(r.metrics.ErrorTypes, k)
	}
	for k := range r.metrics.ModelTypeFailures {
		delete(r.metrics.ModelTypeFailures, k)
	}
}

// recordAttempt records metrics for a retry attempt.
func (r *RetryParser) recordAttempt(strategy RetryStrategy, modelType string, duration time.Duration, success bool, err error) {
	r.metrics.mutex.Lock()
	defer r.metrics.mutex.Unlock()

	r.metrics.TotalAttempts++
	r.metrics.StrategyUsage[strategy]++
	r.metrics.TotalDuration += duration

	// Record latency
	r.metrics.latencyHistory = append(r.metrics.latencyHistory, duration)
	if len(r.metrics.latencyHistory) > 1000 {
		// Keep only the most recent 1000 latencies
		copy(r.metrics.latencyHistory, r.metrics.latencyHistory[1:])
		r.metrics.latencyHistory = r.metrics.latencyHistory[:999]
	}

	if success {
		r.metrics.SuccessfulRetries++
	} else {
		r.metrics.FailedRetries++
		r.metrics.ModelTypeFailures[modelType]++
		if err != nil {
			r.metrics.ErrorTypes[err.Error()]++
		}
	}

	// Update derived metrics
	r.updateDerivedMetrics()
}

// recordFallbackUsed records when fallback handling is used.
func (r *RetryParser) recordFallbackUsed() {
	r.metrics.mutex.Lock()
	defer r.metrics.mutex.Unlock()
	r.metrics.FallbackUsed++
}

// recordCircuitBreakerOpen records when circuit breaker blocks a request.
func (r *RetryParser) recordCircuitBreakerOpen() {
	r.metrics.mutex.Lock()
	defer r.metrics.mutex.Unlock()
	r.metrics.CircuitBreakerOpen++
}

// updateDerivedMetrics updates calculated metrics (must be called with lock held).
func (r *RetryParser) updateDerivedMetrics() {
	// Update average latency
	if r.metrics.TotalAttempts > 0 {
		r.metrics.AverageLatency = r.metrics.TotalDuration / time.Duration(r.metrics.TotalAttempts)
	}

	// Update requests per second
	elapsed := time.Since(r.metrics.StartTime)
	if elapsed > 0 {
		r.metrics.RequestsPerSecond = float64(r.metrics.TotalAttempts) / elapsed.Seconds()
	}

	// Update strategy success rates
	for strategy := range r.metrics.StrategyUsage {
		total := r.metrics.StrategyUsage[strategy]
		if total > 0 {
			// Calculate success rate for this strategy (simplified for now)
			successRate := float64(r.metrics.SuccessfulRetries) / float64(r.metrics.TotalAttempts)
			r.metrics.StrategySuccessRate[strategy] = successRate
		}
	}

	// Update percentile latencies
	r.updateLatencyPercentiles()
}

// updateLatencyPercentiles calculates P95 and P99 latencies.
func (r *RetryParser) updateLatencyPercentiles() {
	if len(r.metrics.latencyHistory) == 0 {
		return
	}

	// Create a sorted copy
	sorted := make([]time.Duration, len(r.metrics.latencyHistory))
	copy(sorted, r.metrics.latencyHistory)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	length := len(sorted)
	if length > 0 {
		p95Index := int(float64(length) * 0.95)
		if p95Index >= length {
			p95Index = length - 1
		}
		r.metrics.P95Latency = sorted[p95Index]

		p99Index := int(float64(length) * 0.99)
		if p99Index >= length {
			p99Index = length - 1
		}
		r.metrics.P99Latency = sorted[p99Index]
	}
}

// AllowRequest checks if the circuit breaker allows the request to proceed.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if recovery timeout has passed
		if time.Since(cb.lastFailTime) > cb.config.RecoveryTimeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.requests = 0
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.requests++

	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			// Enough successes to close the circuit
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
			cb.requests = 0
		}
	}
}

// RecordFailure records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.requests++
	cb.lastFailTime = time.Now()

	if cb.state == StateHalfOpen {
		// Failure during recovery - back to open
		cb.state = StateOpen
		cb.failures = 0
		cb.successes = 0
		cb.requests = 0
		return
	}

	// Check if we should open the circuit
	if cb.requests >= cb.config.RequestVolumeThreshold &&
		cb.failures >= cb.config.FailureThreshold {
		cb.state = StateOpen
		cb.failures = 0
		cb.successes = 0
		cb.requests = 0
	}
}

// GetState returns the current circuit breaker state.
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics.
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"state":                    cb.state.String(),
		"failures":                 cb.failures,
		"successes":                cb.successes,
		"requests":                 cb.requests,
		"failure_threshold":        cb.config.FailureThreshold,
		"recovery_timeout":         cb.config.RecoveryTimeout.String(),
		"request_volume_threshold": cb.config.RequestVolumeThreshold,
		"success_threshold":        cb.config.SuccessThreshold,
		"last_fail_time":           cb.lastFailTime,
	}
}

// Reset resets the circuit breaker to its initial state.
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.requests = 0
	cb.lastFailTime = time.Time{}
}

// ParseWithRetry attempts to parse a response using multiple strategies with retry logic.
// It implements a fallback chain: specific → generic → error.
func (r *RetryParser) ParseWithRetry(ctx context.Context, raw *types.RawResponse, modelType string, originalQuery string, sessionID string) (*types.StructuredQuery, error) {
	startTime := time.Now()
	
	if raw == nil {
		return nil, errors.NewParsingError("raw response is nil", errors.ComponentParser, "retry_parser", 0.0, "")
	}

	// Check circuit breaker before attempting parsing
	if !r.circuitBreaker.AllowRequest() {
		r.recordCircuitBreakerOpen()
		return nil, errors.NewParsingError(
			"circuit breaker is OPEN - service temporarily unavailable",
			errors.ComponentParser,
			"retry_parser_circuit_breaker",
			0.0,
			raw.Content,
		)
	}

	// Enforce maximum output length if configured
	if r.maxOutputLength > 0 && len(raw.Content) > r.maxOutputLength {
		raw.Content = raw.Content[:r.maxOutputLength]
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
				duration := time.Since(startTime)
				r.recordAttempt(strategy, modelType, duration, true, nil)
				r.circuitBreaker.RecordSuccess()
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
						duration := time.Since(startTime)
						r.recordAttempt(strategy, modelType, duration, true, nil)
						r.circuitBreaker.RecordSuccess()
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
		duration := time.Since(startTime)
		r.recordAttempt(bestResult.Strategy, modelType, duration, true, nil)
		r.circuitBreaker.RecordSuccess()
		return bestResult.Query, nil
	}

	// Use fallback handler if configured and available
	if r.fallbackHandler != nil {
		if fallback, ferr := r.fallbackHandler.CreateMinimalQuery(raw, modelType, originalQuery); ferr == nil && fallback != nil {
			duration := time.Since(startTime)
			r.recordFallbackUsed()
			r.recordAttempt("fallback", modelType, duration, true, nil)
			r.circuitBreaker.RecordSuccess()
			return fallback, nil
		}
	}

	// If no fallback handler is configured, use a default one
	if r.fallbackHandler == nil {
		defaultHandler := NewFallbackHandler()
		if fallback, ferr := defaultHandler.CreateMinimalQuery(raw, modelType, originalQuery); ferr == nil && fallback != nil {
			duration := time.Since(startTime)
			r.recordFallbackUsed()
			r.recordAttempt("fallback", modelType, duration, true, nil)
			r.circuitBreaker.RecordSuccess()
			return fallback, nil
		}
	}

	// All strategies failed - record failure and return error
	duration := time.Since(startTime)
	r.circuitBreaker.RecordFailure()
	
	// Return the last error or create a comprehensive error
	if lastError != nil {
		r.recordAttempt("all_strategies", modelType, duration, false, lastError)
		return nil, lastError
	}

	finalError := errors.NewParsingError(
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
	
	r.recordAttempt("all_strategies", modelType, duration, false, finalError)
	return nil, finalError
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

// GetRetryStatistics returns comprehensive statistics about retry attempts.
func (r *RetryParser) GetRetryStatistics() map[string]interface{} {
	metrics := r.GetMetrics()
	circuitBreakerStats := r.circuitBreaker.GetStats()

	stats := map[string]interface{}{
		// Configuration
		"max_retries":           r.config.MaxRetries,
		"retry_delay":           r.config.RetryDelay.String(),
		"confidence_threshold":  r.config.ConfidenceThreshold,
		"enable_reprompting":    r.config.EnableReprompting,
		"registered_strategies": r.getRegisteredStrategies(),

		// Metrics
		"total_attempts":        metrics.TotalAttempts,
		"successful_retries":    metrics.SuccessfulRetries,
		"failed_retries":        metrics.FailedRetries,
		"fallback_used":         metrics.FallbackUsed,
		"circuit_breaker_open":  metrics.CircuitBreakerOpen,
		"average_latency":       metrics.AverageLatency.String(),
		"p95_latency":           metrics.P95Latency.String(),
		"p99_latency":           metrics.P99Latency.String(),
		"requests_per_second":   metrics.RequestsPerSecond,
		"strategy_usage":        metrics.StrategyUsage,
		"strategy_success_rate": metrics.StrategySuccessRate,
		"error_types":           metrics.ErrorTypes,
		"model_type_failures":   metrics.ModelTypeFailures,
		"start_time":            metrics.StartTime,
		"last_reset_time":       metrics.LastResetTime,

		// Circuit Breaker
		"circuit_breaker": circuitBreakerStats,
	}

	return stats
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

// SetMaxOutputLength sets a maximum length for raw model outputs to consider during parsing.
// A value <= 0 disables the enforcement.
func (r *RetryParser) SetMaxOutputLength(max int) {
	r.maxOutputLength = max
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

