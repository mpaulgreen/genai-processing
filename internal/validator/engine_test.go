package validator

import (
	"context"
	"errors"
	"testing"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// MockRule implements ValidationRule for testing
type MockRule struct {
	name        string
	enabled     bool
	isValid     bool
	severity    string
	errors      []string
	warnings    []string
	delay       time.Duration // For testing timeouts
}

func NewMockRule(name string, enabled bool, isValid bool) *MockRule {
	return &MockRule{
		name:     name,
		enabled:  enabled,
		isValid:  isValid,
		severity: "info",
		errors:   []string{},
		warnings: []string{},
	}
}

func (m *MockRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	return &interfaces.ValidationResult{
		IsValid:         m.isValid,
		RuleName:        m.name,
		Severity:        m.severity,
		Message:         "Mock rule validation",
		Details:         map[string]interface{}{},
		Recommendations: []string{},
		Warnings:        m.warnings,
		Errors:          m.errors,
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}
}

func (m *MockRule) GetRuleName() string        { return m.name }
func (m *MockRule) GetRuleDescription() string { return "Mock rule for testing" }
func (m *MockRule) IsEnabled() bool            { return m.enabled }
func (m *MockRule) GetSeverity() string        { return m.severity }

func TestRuleEngine_NewRuleEngine(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: GetRuleEngineDefaults(),
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	if len(engine.rules) == 0 {
		t.Error("Expected some default rules to be registered")
	}

	if len(engine.priorities) == 0 {
		t.Error("Expected priorities to be initialized")
	}
}

func TestRuleEngine_RegisterRule(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: GetRuleEngineDefaults(),
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	rule := NewMockRule("test_rule", true, true)

	err := engine.RegisterRule("test_rule", rule)
	if err != nil {
		t.Errorf("Expected no error registering rule, got %v", err)
	}

	// Test duplicate registration
	err = engine.RegisterRule("test_rule", rule)
	if err == nil {
		t.Error("Expected error when registering duplicate rule")
	}
}

func TestRuleEngine_SetRuleDependency(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: GetRuleEngineDefaults(),
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	rule1 := NewMockRule("rule1", true, true)
	rule2 := NewMockRule("rule2", true, true)
	
	engine.RegisterRule("rule1", rule1)
	engine.RegisterRule("rule2", rule2)

	// Test valid dependency
	err := engine.SetRuleDependency("rule2", []string{"rule1"})
	if err != nil {
		t.Errorf("Expected no error setting dependency, got %v", err)
	}

	// Test invalid dependency
	err = engine.SetRuleDependency("rule2", []string{"nonexistent_rule"})
	if err == nil {
		t.Error("Expected error when setting dependency on nonexistent rule")
	}
}

func TestRuleEngine_EvaluateRules_Sequential(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: RuleEngineConfig{
			EnableParallelEvaluation: false,
			MaxConcurrentRules:       1,
			RuleTimeoutSeconds:       30,
			FailFastMode:            false,
			EnableRuleCaching:       false,
		},
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	rule1 := NewMockRule("rule1", true, true)
	rule2 := NewMockRule("rule2", true, false)
	rule2.errors = []string{"Rule2 failed"}
	rule2.severity = "critical"
	
	engine.RegisterRule("rule1", rule1)
	engine.RegisterRule("rule2", rule2)

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	result, err := engine.EvaluateRules(query)
	if err != nil {
		t.Errorf("Expected no error evaluating rules, got %v", err)
	}

	if result.IsValid {
		t.Error("Expected overall validation to fail")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors in result")
	}

	// Check that rule results are included
	if ruleResults, ok := result.Details["rule_results"].(map[string]*interfaces.ValidationResult); ok {
		if len(ruleResults) != 2 {
			t.Errorf("Expected 2 rule results, got %d", len(ruleResults))
		}
	} else {
		t.Error("Expected rule_results in details")
	}
}

func TestRuleEngine_EvaluateRules_Parallel(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: RuleEngineConfig{
			EnableParallelEvaluation: true,
			MaxConcurrentRules:       3,
			RuleTimeoutSeconds:       30,
			FailFastMode:            false,
			EnableRuleCaching:       false,
		},
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	rule1 := NewMockRule("rule1", true, true)
	rule2 := NewMockRule("rule2", true, true)
	rule3 := NewMockRule("rule3", true, true)
	
	engine.RegisterRule("rule1", rule1)
	engine.RegisterRule("rule2", rule2)
	engine.RegisterRule("rule3", rule3)

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	result, err := engine.EvaluateRules(query)
	if err != nil {
		t.Errorf("Expected no error evaluating rules, got %v", err)
	}

	if !result.IsValid {
		t.Error("Expected overall validation to pass")
	}

	// Check that all rules were evaluated
	if ruleResults, ok := result.Details["rule_results"].(map[string]*interfaces.ValidationResult); ok {
		if len(ruleResults) != 3 {
			t.Errorf("Expected 3 rule results, got %d", len(ruleResults))
		}
	} else {
		t.Error("Expected rule_results in details")
	}
}

func TestRuleEngine_FailFastMode(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: RuleEngineConfig{
			EnableParallelEvaluation: false,
			MaxConcurrentRules:       1,
			RuleTimeoutSeconds:       30,
			FailFastMode:            true,
			EnableRuleCaching:       false,
		},
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	rule1 := NewMockRule("rule1", true, false)
	rule1.errors = []string{"Critical failure"}
	rule1.severity = "critical"
	
	rule2 := NewMockRule("rule2", true, true)
	
	engine.RegisterRule("rule1", rule1)
	engine.RegisterRule("rule2", rule2)

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	result, err := engine.EvaluateRules(query)
	if err != nil {
		t.Errorf("Expected no error evaluating rules, got %v", err)
	}

	if result.IsValid {
		t.Error("Expected overall validation to fail")
	}

	// In fail-fast mode, rule2 should not be evaluated
	if ruleResults, ok := result.Details["rule_results"].(map[string]*interfaces.ValidationResult); ok {
		if len(ruleResults) > 1 {
			t.Errorf("Expected fail-fast to stop after first critical error, but got %d results", len(ruleResults))
		}
	}
}

func TestRuleEngine_RuleTimeout(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: RuleEngineConfig{
			EnableParallelEvaluation: false,
			MaxConcurrentRules:       1,
			RuleTimeoutSeconds:       1, // 1 second timeout
			FailFastMode:            false,
			EnableRuleCaching:       false,
		},
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	rule1 := NewMockRule("slow_rule", true, true)
	rule1.delay = 2 * time.Second // Longer than timeout
	
	engine.RegisterRule("slow_rule", rule1)

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	result, err := engine.EvaluateRules(query)
	if err != nil {
		t.Errorf("Expected no error evaluating rules, got %v", err)
	}

	if result.IsValid {
		t.Error("Expected validation to fail due to timeout")
	}

	// Check for timeout error
	found := false
	for _, errMsg := range result.Errors {
		if errMsg == "Rule evaluation timeout" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected timeout error in results")
	}
}

func TestRuleEngine_RuleDependencies(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: RuleEngineConfig{
			EnableParallelEvaluation: false,
			MaxConcurrentRules:       1,
			RuleTimeoutSeconds:       30,
			FailFastMode:            false,
			EnableRuleDependencies:  true,
			EnableRuleCaching:       false,
		},
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	baseRule := NewMockRule("base_rule", true, false) // Fails
	baseRule.severity = "critical"
	
	dependentRule := NewMockRule("dependent_rule", true, true)
	
	engine.RegisterRule("base_rule", baseRule)
	engine.RegisterRule("dependent_rule", dependentRule)
	
	// Set dependency
	engine.SetRuleDependency("dependent_rule", []string{"base_rule"})

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	result, err := engine.EvaluateRules(query)
	if err != nil {
		t.Errorf("Expected no error evaluating rules, got %v", err)
	}

	// Check that dependent rule was not evaluated due to failed dependency
	if ruleResults, ok := result.Details["rule_results"].(map[string]*interfaces.ValidationResult); ok {
		if _, exists := ruleResults["dependent_rule"]; exists {
			t.Error("Expected dependent rule not to be evaluated when dependency fails")
		}
	}
}

func TestRuleEngine_RuleConditions(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: GetRuleEngineDefaults(),
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	conditionalRule := NewMockRule("conditional_rule", true, true)
	engine.RegisterRule("conditional_rule", conditionalRule)
	
	// Set condition - only evaluate for kube-apiserver
	condition := &RuleCondition{
		Field:    "log_source",
		Operator: "eq",
		Value:    "kube-apiserver",
	}
	engine.SetRuleCondition("conditional_rule", condition)

	// Test with matching condition
	query1 := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	result1, err := engine.EvaluateRules(query1)
	if err != nil {
		t.Errorf("Expected no error evaluating rules, got %v", err)
	}

	if ruleResults, ok := result1.Details["rule_results"].(map[string]*interfaces.ValidationResult); ok {
		if _, exists := ruleResults["conditional_rule"]; !exists {
			t.Error("Expected conditional rule to be evaluated when condition matches")
		}
	}

	// Test with non-matching condition
	query2 := &types.StructuredQuery{
		LogSource: "oauth-server",
	}

	result2, err := engine.EvaluateRules(query2)
	if err != nil {
		t.Errorf("Expected no error evaluating rules, got %v", err)
	}

	if ruleResults, ok := result2.Details["rule_results"].(map[string]*interfaces.ValidationResult); ok {
		if _, exists := ruleResults["conditional_rule"]; exists {
			t.Error("Expected conditional rule not to be evaluated when condition doesn't match")
		}
	}
}

func TestRuleEngine_CalculateEvaluationOrder(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: RuleEngineConfig{
			EnableRuleDependencies: true,
			RulePriorities: map[string]int{
				"high_priority": 100,
				"med_priority":  50,
				"low_priority":  10,
			},
		},
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	highRule := NewMockRule("high_priority", true, true)
	medRule := NewMockRule("med_priority", true, true)
	lowRule := NewMockRule("low_priority", true, true)
	
	engine.RegisterRule("high_priority", highRule)
	engine.RegisterRule("med_priority", medRule)
	engine.RegisterRule("low_priority", lowRule)
	
	// Set dependency: med depends on low
	engine.SetRuleDependency("med_priority", []string{"low_priority"})

	order, err := engine.calculateEvaluationOrder()
	if err != nil {
		t.Errorf("Expected no error calculating order, got %v", err)
	}

	if len(order) != 3 {
		t.Errorf("Expected 3 rules in order, got %d", len(order))
	}

	// low_priority should come before med_priority due to dependency
	lowIndex := -1
	medIndex := -1
	for i, rule := range order {
		if rule == "low_priority" {
			lowIndex = i
		} else if rule == "med_priority" {
			medIndex = i
		}
	}

	if lowIndex == -1 || medIndex == -1 {
		t.Error("Expected both low_priority and med_priority in evaluation order")
	}

	if lowIndex >= medIndex {
		t.Error("Expected low_priority to come before med_priority due to dependency")
	}
}

func TestRuleEngine_CircularDependency(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: RuleEngineConfig{
			EnableRuleDependencies: true,
		},
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	rule1 := NewMockRule("rule1", true, true)
	rule2 := NewMockRule("rule2", true, true)
	
	engine.RegisterRule("rule1", rule1)
	engine.RegisterRule("rule2", rule2)
	
	// Create circular dependency
	engine.SetRuleDependency("rule1", []string{"rule2"})
	engine.SetRuleDependency("rule2", []string{"rule1"})

	_, err := engine.calculateEvaluationOrder()
	if err == nil {
		t.Error("Expected error due to circular dependency")
	}
}

func TestRuleEngine_GetEngineStats(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: GetRuleEngineDefaults(),
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)

	stats := engine.GetEngineStats()

	expectedKeys := []string{
		"total_rules", "enabled_rules", "dependencies_count", "conditions_count",
		"caching_enabled", "parallel_enabled", "max_concurrent", "rule_timeout_seconds",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stat key '%s' to be present", key)
		}
	}

	// Check some basic validations
	if totalRules, ok := stats["total_rules"].(int); !ok || totalRules < 0 {
		t.Error("Expected total_rules to be a non-negative integer")
	}

	if enabledRules, ok := stats["enabled_rules"].(int); !ok || enabledRules < 0 {
		t.Error("Expected enabled_rules to be a non-negative integer")
	}
}

func TestRuleEngine_ContextCancellation(t *testing.T) {
	config := &ValidationConfig{
		RuleEngine: RuleEngineConfig{
			EnableParallelEvaluation: false,
			MaxConcurrentRules:       1,
			RuleTimeoutSeconds:       30,
			FailFastMode:            false,
			EnableRuleCaching:       false,
		},
	}
	config.ApplyDefaults()

	engine := NewRuleEngine(config)
	
	// Clear default rules for clean test
	engine.rules = make(map[string]interfaces.ValidationRule)
	
	slowRule := NewMockRule("slow_rule", true, true)
	slowRule.delay = 2 * time.Second
	
	engine.RegisterRule("slow_rule", slowRule)

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := engine.EvaluateRulesWithContext(ctx, query)
	if err == nil {
		t.Error("Expected context cancellation error")
	}

	// Check if the error is context.DeadlineExceeded or contains it
	if err != context.DeadlineExceeded && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}