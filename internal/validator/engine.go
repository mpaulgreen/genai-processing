package validator

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"genai-processing/internal/validator/rules"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// RuleEngine manages rule evaluation with dynamic conditions, dependencies, and priorities
type RuleEngine struct {
	config          *ValidationConfig
	rules           map[string]interfaces.ValidationRule
	dependencies    map[string][]string
	priorities      map[string]int
	conditions      map[string]*RuleCondition
	cache           *RuleCache
	enableCaching   bool
	enableParallel  bool
	maxConcurrent   int
	ruleTimeout     time.Duration
	mu              sync.RWMutex
}

// RuleEvaluationContext contains context for rule evaluation
type RuleEvaluationContext struct {
	Query           *types.StructuredQuery
	RuleResults     map[string]*interfaces.ValidationResult
	EvaluationOrder []string
	StartTime       time.Time
	Context         context.Context
}

// RuleCache provides caching for rule evaluation results
type RuleCache struct {
	cache   map[string]*CacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

// CacheEntry represents a cached rule evaluation result
type CacheEntry struct {
	Result    *interfaces.ValidationResult
	Timestamp time.Time
	Hash      string
}

// NewRuleEngine creates a new rule evaluation engine
func NewRuleEngine(config *ValidationConfig) *RuleEngine {
	engine := &RuleEngine{
		config:        config,
		rules:         make(map[string]interfaces.ValidationRule),
		dependencies:  make(map[string][]string),
		priorities:    make(map[string]int),
		conditions:    make(map[string]*RuleCondition),
		enableCaching: config.RuleEngine.EnableRuleCaching,
		enableParallel: config.RuleEngine.EnableParallelEvaluation,
		maxConcurrent: config.RuleEngine.MaxConcurrentRules,
		ruleTimeout:   time.Duration(config.RuleEngine.RuleTimeoutSeconds) * time.Second,
	}

	// Initialize cache if enabled
	if engine.enableCaching {
		engine.cache = &RuleCache{
			cache: make(map[string]*CacheEntry),
			ttl:   time.Duration(config.RuleEngine.CacheTTLSeconds) * time.Second,
		}
	}

	// Initialize rule priorities
	engine.initializePriorities()

	// Initialize default rules
	engine.initializeRules()

	return engine
}

// RegisterRule registers a validation rule with the engine
func (e *RuleEngine) RegisterRule(name string, rule interfaces.ValidationRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.rules[name]; exists {
		return fmt.Errorf("rule '%s' already registered", name)
	}

	e.rules[name] = rule
	return nil
}

// SetRuleDependency sets a dependency relationship between rules
func (e *RuleEngine) SetRuleDependency(ruleName string, dependencies []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate that all dependency rules exist
	for _, dep := range dependencies {
		if _, exists := e.rules[dep]; !exists {
			return fmt.Errorf("dependency rule '%s' not found", dep)
		}
	}

	e.dependencies[ruleName] = dependencies
	return nil
}

// SetRuleCondition sets a condition for rule evaluation
func (e *RuleEngine) SetRuleCondition(ruleName string, condition *RuleCondition) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.rules[ruleName]; !exists {
		return fmt.Errorf("rule '%s' not found", ruleName)
	}

	e.conditions[ruleName] = condition
	return nil
}

// EvaluateRules evaluates all applicable rules for a query
func (e *RuleEngine) EvaluateRules(query *types.StructuredQuery) (*interfaces.ValidationResult, error) {
	ctx := context.Background()
	return e.EvaluateRulesWithContext(ctx, query)
}

// EvaluateRulesWithContext evaluates rules with context for cancellation
func (e *RuleEngine) EvaluateRulesWithContext(ctx context.Context, query *types.StructuredQuery) (*interfaces.ValidationResult, error) {
	evalCtx := &RuleEvaluationContext{
		Query:       query,
		RuleResults: make(map[string]*interfaces.ValidationResult),
		StartTime:   time.Now(),
		Context:     ctx,
	}

	// Determine evaluation order based on dependencies and priorities
	evaluationOrder, err := e.calculateEvaluationOrder()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate evaluation order: %w", err)
	}
	evalCtx.EvaluationOrder = evaluationOrder

	// Evaluate rules
	if e.enableParallel {
		return e.evaluateRulesParallel(evalCtx)
	}
	return e.evaluateRulesSequential(evalCtx)
}

// evaluateRulesSequential evaluates rules one by one
func (e *RuleEngine) evaluateRulesSequential(evalCtx *RuleEvaluationContext) (*interfaces.ValidationResult, error) {
	for _, ruleName := range evalCtx.EvaluationOrder {
		// Check context cancellation
		select {
		case <-evalCtx.Context.Done():
			return nil, evalCtx.Context.Err()
		default:
		}

		// Evaluate rule if conditions are met
		if e.shouldEvaluateRule(ruleName, evalCtx) {
			result, err := e.evaluateSingleRule(ruleName, evalCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate rule '%s': %w", ruleName, err)
			}

			evalCtx.RuleResults[ruleName] = result

			// Fail fast on critical errors if enabled
			if e.config.RuleEngine.FailFastMode && !result.IsValid && result.Severity == "critical" {
				return e.aggregateResults(evalCtx), nil
			}
		}
	}

	return e.aggregateResults(evalCtx), nil
}

// evaluateRulesParallel evaluates rules in parallel where possible
func (e *RuleEngine) evaluateRulesParallel(evalCtx *RuleEvaluationContext) (*interfaces.ValidationResult, error) {
	// Group rules by dependency levels
	dependencyLevels := e.groupRulesByDependencyLevel(evalCtx.EvaluationOrder)

	for _, levelRules := range dependencyLevels {
		// Check context cancellation
		select {
		case <-evalCtx.Context.Done():
			return nil, evalCtx.Context.Err()
		default:
		}

		// Evaluate rules at this level in parallel
		err := e.evaluateRuleLevel(levelRules, evalCtx)
		if err != nil {
			return nil, err
		}

		// Check for fail fast conditions
		if e.config.RuleEngine.FailFastMode {
			for _, ruleName := range levelRules {
				if result, exists := evalCtx.RuleResults[ruleName]; exists {
					if !result.IsValid && result.Severity == "critical" {
						return e.aggregateResults(evalCtx), nil
					}
				}
			}
		}
	}

	return e.aggregateResults(evalCtx), nil
}

// evaluateRuleLevel evaluates a level of rules in parallel
func (e *RuleEngine) evaluateRuleLevel(ruleNames []string, evalCtx *RuleEvaluationContext) error {
	// Create worker pool
	maxWorkers := e.maxConcurrent
	if len(ruleNames) < maxWorkers {
		maxWorkers = len(ruleNames)
	}

	jobChan := make(chan string, len(ruleNames))
	resultChan := make(chan struct{ name string; result *interfaces.ValidationResult; err error }, len(ruleNames))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ruleName := range jobChan {
				if e.shouldEvaluateRule(ruleName, evalCtx) {
					result, err := e.evaluateSingleRule(ruleName, evalCtx)
					resultChan <- struct{ name string; result *interfaces.ValidationResult; err error }{ruleName, result, err}
				} else {
					resultChan <- struct{ name string; result *interfaces.ValidationResult; err error }{ruleName, nil, nil}
				}
			}
		}()
	}

	// Send jobs
	for _, ruleName := range ruleNames {
		jobChan <- ruleName
	}
	close(jobChan)

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		if result.err != nil {
			return fmt.Errorf("failed to evaluate rule '%s': %w", result.name, result.err)
		}
		if result.result != nil {
			evalCtx.RuleResults[result.name] = result.result
		}
	}

	return nil
}

// evaluateSingleRule evaluates a single rule with timeout and caching
func (e *RuleEngine) evaluateSingleRule(ruleName string, evalCtx *RuleEvaluationContext) (*interfaces.ValidationResult, error) {
	// Check cache if enabled
	if e.enableCaching {
		if cached := e.getCachedResult(ruleName, evalCtx.Query); cached != nil {
			return cached, nil
		}
	}

	rule := e.rules[ruleName]
	if rule == nil {
		return nil, fmt.Errorf("rule '%s' not found", ruleName)
	}

	// Create context with timeout (shorter of parent context or rule timeout)
	ctx, cancel := context.WithTimeout(evalCtx.Context, e.ruleTimeout)
	defer cancel()

	// Evaluate rule
	resultChan := make(chan *interfaces.ValidationResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		result := rule.Validate(evalCtx.Query)
		result.Timestamp = time.Now().Format(time.RFC3339)
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		// Cache result if enabled
		if e.enableCaching {
			e.cacheResult(ruleName, evalCtx.Query, result)
		}
		return result, nil
	case <-ctx.Done():
		// Check if it's the parent context that was cancelled vs rule timeout
		if evalCtx.Context.Err() != nil {
			// Parent context was cancelled
			return nil, evalCtx.Context.Err()
		}
		// Rule-specific timeout
		return &interfaces.ValidationResult{
			IsValid:   false,
			RuleName:  ruleName,
			Severity:  "critical",
			Message:   fmt.Sprintf("Rule evaluation timed out after %v", e.ruleTimeout),
			Errors:    []string{"Rule evaluation timeout"},
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	case err := <-errorChan:
		return nil, err
	}
}

// shouldEvaluateRule determines if a rule should be evaluated based on conditions and dependencies
func (e *RuleEngine) shouldEvaluateRule(ruleName string, evalCtx *RuleEvaluationContext) bool {
	rule := e.rules[ruleName]
	if rule == nil || !rule.IsEnabled() {
		return false
	}

	// Check dependencies
	if deps, hasDeps := e.dependencies[ruleName]; hasDeps {
		for _, dep := range deps {
			depResult, exists := evalCtx.RuleResults[dep]
			if !exists || !depResult.IsValid {
				return false // Dependency not met
			}
		}
	}

	// Check conditions
	if condition, hasCondition := e.conditions[ruleName]; hasCondition {
		return e.evaluateCondition(condition, evalCtx.Query)
	}

	return true
}

// evaluateCondition evaluates a rule condition
func (e *RuleEngine) evaluateCondition(condition *RuleCondition, query *types.StructuredQuery) bool {
	// Simple condition evaluation - can be extended for more complex logic
	switch condition.Field {
	case "log_source":
		return e.evaluateFieldCondition(query.LogSource, condition)
	case "analysis":
		return (query.Analysis != nil) == (condition.Operator == "exists")
	case "multi_source":
		return (query.MultiSource != nil) == (condition.Operator == "exists")
	case "behavioral_analysis":
		return (query.BehavioralAnalysis != nil) == (condition.Operator == "exists")
	case "compliance_framework":
		return (query.ComplianceFramework != nil) == (condition.Operator == "exists")
	default:
		return true // Unknown field, allow evaluation
	}
}

// evaluateFieldCondition evaluates a condition for a specific field
func (e *RuleEngine) evaluateFieldCondition(fieldValue string, condition *RuleCondition) bool {
	switch condition.Operator {
	case "eq":
		if strValue, ok := condition.Value.(string); ok {
			return fieldValue == strValue
		}
	case "ne":
		if strValue, ok := condition.Value.(string); ok {
			return fieldValue != strValue
		}
	case "in":
		if arrayValue, ok := condition.Value.([]string); ok {
			for _, item := range arrayValue {
				if fieldValue == item {
					return true
				}
			}
			return false
		}
	case "not_in":
		if arrayValue, ok := condition.Value.([]string); ok {
			for _, item := range arrayValue {
				if fieldValue == item {
					return false
				}
			}
			return true
		}
	case "exists":
		return fieldValue != ""
	case "not_exists":
		return fieldValue == ""
	}
	return false
}

// calculateEvaluationOrder determines the order of rule evaluation based on dependencies and priorities
func (e *RuleEngine) calculateEvaluationOrder() ([]string, error) {
	// Get all enabled rules
	enabledRules := make([]string, 0, len(e.rules))
	for name, rule := range e.rules {
		if rule.IsEnabled() {
			enabledRules = append(enabledRules, name)
		}
	}

	// Topological sort for dependency resolution
	sorted, err := e.topologicalSort(enabledRules)
	if err != nil {
		return nil, err
	}

	// Sort by priority within dependency levels
	return e.sortByPriority(sorted), nil
}

// topologicalSort performs topological sorting for dependency resolution
func (e *RuleEngine) topologicalSort(rules []string) ([]string, error) {
	// Simplified topological sort
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	result := make([]string, 0, len(rules))

	var visit func(string) error
	visit = func(rule string) error {
		if visiting[rule] {
			return fmt.Errorf("circular dependency detected involving rule '%s'", rule)
		}
		if visited[rule] {
			return nil
		}

		visiting[rule] = true
		
		// Visit dependencies first
		if deps, hasDeps := e.dependencies[rule]; hasDeps {
			for _, dep := range deps {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		visiting[rule] = false
		visited[rule] = true
		result = append(result, rule)
		return nil
	}

	for _, rule := range rules {
		if !visited[rule] {
			if err := visit(rule); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// sortByPriority sorts rules by priority while maintaining dependency order
func (e *RuleEngine) sortByPriority(rules []string) []string {
	// Group rules by dependency level
	levels := e.groupRulesByDependencyLevel(rules)
	
	result := make([]string, 0, len(rules))
	for _, level := range levels {
		// Sort each level by priority
		sort.Slice(level, func(i, j int) bool {
			priorityI := e.priorities[level[i]]
			priorityJ := e.priorities[level[j]]
			return priorityI > priorityJ // Higher priority first
		})
		result = append(result, level...)
	}

	return result
}

// groupRulesByDependencyLevel groups rules by their dependency depth
func (e *RuleEngine) groupRulesByDependencyLevel(rules []string) [][]string {
	levels := make(map[int][]string)
	depths := make(map[string]int)

	// Calculate depth for each rule
	var calculateDepth func(string) int
	calculateDepth = func(rule string) int {
		if depth, exists := depths[rule]; exists {
			return depth
		}

		maxDepth := 0
		if deps, hasDeps := e.dependencies[rule]; hasDeps {
			for _, dep := range deps {
				depDepth := calculateDepth(dep)
				if depDepth >= maxDepth {
					maxDepth = depDepth + 1
				}
			}
		}

		depths[rule] = maxDepth
		return maxDepth
	}

	// Group by depth
	for _, rule := range rules {
		depth := calculateDepth(rule)
		levels[depth] = append(levels[depth], rule)
	}

	// Convert to slice
	maxLevel := 0
	for level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	result := make([][]string, maxLevel+1)
	for i := 0; i <= maxLevel; i++ {
		result[i] = levels[i]
	}

	return result
}

// aggregateResults combines all rule results into a final validation result
func (e *RuleEngine) aggregateResults(evalCtx *RuleEvaluationContext) *interfaces.ValidationResult {
	aggregated := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "rule_engine_validation",
		Severity:        "info",
		Message:         "All rules passed validation",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   evalCtx.Query,
	}

	// Aggregate results
	criticalCount := 0
	for _, result := range evalCtx.RuleResults {
		if !result.IsValid {
			aggregated.IsValid = false
			aggregated.Errors = append(aggregated.Errors, result.Errors...)
			if result.Severity == "critical" {
				criticalCount++
			}
		}
		aggregated.Warnings = append(aggregated.Warnings, result.Warnings...)
		aggregated.Recommendations = append(aggregated.Recommendations, result.Recommendations...)
	}

	// Set final severity and message
	if !aggregated.IsValid {
		if criticalCount > 0 {
			aggregated.Severity = "critical"
			aggregated.Message = fmt.Sprintf("Validation failed: %d critical errors", criticalCount)
		} else {
			aggregated.Severity = "error"
			aggregated.Message = "Validation failed"
		}
	} else if len(aggregated.Warnings) > 0 {
		aggregated.Severity = "warning"
		aggregated.Message = fmt.Sprintf("Validation passed with %d warnings", len(aggregated.Warnings))
	}

	// Add engine details
	aggregated.Details["rule_results"] = evalCtx.RuleResults
	aggregated.Details["evaluation_order"] = evalCtx.EvaluationOrder
	aggregated.Details["total_rules_evaluated"] = len(evalCtx.RuleResults)
	aggregated.Details["evaluation_time_ms"] = time.Since(evalCtx.StartTime).Milliseconds()
	aggregated.Details["engine_mode"] = map[string]interface{}{
		"parallel_evaluation": e.enableParallel,
		"caching_enabled":     e.enableCaching,
		"fail_fast_mode":      e.config.RuleEngine.FailFastMode,
	}

	return aggregated
}

// initializePriorities sets up default rule priorities
func (e *RuleEngine) initializePriorities() {
	// Use priorities from config if available
	if e.config.RuleEngine.RulePriorities != nil {
		e.priorities = e.config.RuleEngine.RulePriorities
	} else {
		// Default priorities
		e.priorities = map[string]int{
			"schema_validation":      100,
			"required_fields":        90,
			"sanitization":          80,
			"patterns":              70,
			"field_values":          60,
			"advanced_analysis":     50,
			"multi_source":          40,
			"behavioral_analytics":  30,
			"compliance":            20,
		}
	}
}

// initializeRules creates and registers default validation rules
func (e *RuleEngine) initializeRules() {
	// Register advanced rules
	e.RegisterRule("advanced_analysis", rules.NewAdvancedAnalysisRule(e.config.GetConfigSection("analysis_limits")))
	e.RegisterRule("multi_source", rules.NewMultiSourceRule(e.config.GetConfigSection("multi_source")))
	e.RegisterRule("behavioral_analytics", rules.NewBehavioralAnalyticsRule(e.config.GetConfigSection("behavioral_analytics")))
	e.RegisterRule("compliance", rules.NewComplianceRule(e.config.GetConfigSection("compliance_framework")))
	e.RegisterRule("field_values", rules.NewFieldValuesRule(nil))

	// Set up default rule conditions
	e.setDefaultConditions()
}

// setDefaultConditions sets up default conditions for rule evaluation
func (e *RuleEngine) setDefaultConditions() {
	// Advanced analysis rule only applies when analysis config is present
	e.SetRuleCondition("advanced_analysis", &RuleCondition{
		Field:    "analysis",
		Operator: "exists",
	})

	// Multi-source rule only applies when multi-source config is present
	e.SetRuleCondition("multi_source", &RuleCondition{
		Field:    "multi_source",
		Operator: "exists",
	})

	// Behavioral analytics rule only applies when behavioral config is present
	e.SetRuleCondition("behavioral_analytics", &RuleCondition{
		Field:    "behavioral_analysis",
		Operator: "exists",
	})

	// Compliance rule only applies when compliance config is present
	e.SetRuleCondition("compliance", &RuleCondition{
		Field:    "compliance_framework",
		Operator: "exists",
	})

	// Performance rule always applies
}

// Cache methods
func (e *RuleEngine) getCachedResult(ruleName string, query *types.StructuredQuery) *interfaces.ValidationResult {
	if e.cache == nil {
		return nil
	}

	e.cache.mu.RLock()
	defer e.cache.mu.RUnlock()

	hash := e.generateCacheKey(ruleName, query)
	if entry, exists := e.cache.cache[hash]; exists {
		if time.Since(entry.Timestamp) < e.cache.ttl {
			return entry.Result
		}
		// Entry expired, remove it
		delete(e.cache.cache, hash)
	}
	return nil
}

func (e *RuleEngine) cacheResult(ruleName string, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	if e.cache == nil {
		return
	}

	e.cache.mu.Lock()
	defer e.cache.mu.Unlock()

	hash := e.generateCacheKey(ruleName, query)
	e.cache.cache[hash] = &CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
		Hash:      hash,
	}
}

func (e *RuleEngine) generateCacheKey(ruleName string, query *types.StructuredQuery) string {
	// Simple hash generation - in production would use more sophisticated hashing
	return fmt.Sprintf("%s_%s_%s_%d", ruleName, query.LogSource, query.Timeframe, query.Limit)
}

// GetEngineStats returns statistics about the rule engine
func (e *RuleEngine) GetEngineStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := map[string]interface{}{
		"total_rules":         len(e.rules),
		"enabled_rules":       0,
		"dependencies_count":  len(e.dependencies),
		"conditions_count":    len(e.conditions),
		"caching_enabled":     e.enableCaching,
		"parallel_enabled":    e.enableParallel,
		"max_concurrent":      e.maxConcurrent,
		"rule_timeout_seconds": e.ruleTimeout.Seconds(),
	}

	// Count enabled rules
	for _, rule := range e.rules {
		if rule.IsEnabled() {
			stats["enabled_rules"] = stats["enabled_rules"].(int) + 1
		}
	}

	if e.cache != nil {
		e.cache.mu.RLock()
		stats["cache_entries"] = len(e.cache.cache)
		stats["cache_ttl_seconds"] = e.cache.ttl.Seconds()
		e.cache.mu.RUnlock()
	}

	return stats
}