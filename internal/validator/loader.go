package validator

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RuleEngineConfig defines configuration for the rule evaluation engine
type RuleEngineConfig struct {
	// EnableParallelEvaluation controls whether rules can be evaluated concurrently
	EnableParallelEvaluation bool `yaml:"enable_parallel_evaluation"`
	
	// MaxConcurrentRules sets the maximum number of rules to evaluate concurrently
	MaxConcurrentRules int `yaml:"max_concurrent_rules"`
	
	// RuleTimeoutSeconds sets the timeout for individual rule evaluation
	RuleTimeoutSeconds int `yaml:"rule_timeout_seconds"`
	
	// FailFastMode stops evaluation on first critical error
	FailFastMode bool `yaml:"fail_fast_mode"`
	
	// EnableRuleDependencies allows rules to depend on other rules
	EnableRuleDependencies bool `yaml:"enable_rule_dependencies"`
	
	// RulePriorities defines rule execution priorities
	RulePriorities map[string]int `yaml:"rule_priorities,omitempty"`
	
	// EnableRuleCaching caches rule evaluation results for performance
	EnableRuleCaching bool `yaml:"enable_rule_caching"`
	
	// CacheTTLSeconds sets cache time-to-live in seconds
	CacheTTLSeconds int `yaml:"cache_ttl_seconds"`
}

// RuleCondition defines conditional rule execution
type RuleCondition struct {
	// Field specifies the query field to check
	Field string `yaml:"field"`
	
	// Operator defines the comparison operator (eq, ne, in, not_in, exists, not_exists)
	Operator string `yaml:"operator"`
	
	// Value is the value to compare against
	Value interface{} `yaml:"value"`
	
	// LogicalOperator combines conditions (and, or)
	LogicalOperator string `yaml:"logical_operator,omitempty"`
}

// ValidationConfig represents the comprehensive configuration structure for validation rules
type ValidationConfig struct {
	SafetyRules struct {
		AllowedLogSources  []string                 `yaml:"allowed_log_sources"`
		AllowedVerbs       []string                 `yaml:"allowed_verbs"`
		AllowedResources   []string                 `yaml:"allowed_resources"`
		ForbiddenPatterns  []string                 `yaml:"forbidden_patterns"`
		TimeframeLimits    map[string]interface{}   `yaml:"timeframe_limits"`
		Sanitization       map[string]interface{}   `yaml:"sanitization"`
		RequiredFields     []string                 `yaml:"required_fields"`
		QueryLimits        map[string]interface{}   `yaml:"query_limits"`
		BusinessHours      map[string]interface{}   `yaml:"business_hours"`
		AnalysisLimits     map[string]interface{}   `yaml:"analysis_limits"`
		ResponseStatus     map[string]interface{}   `yaml:"response_status"`
		AuthDecisions      map[string]interface{}   `yaml:"auth_decisions"`
		SeverityLevels     []string                 `yaml:"severity_levels"`
		RuleCategories     []string                 `yaml:"rule_categories"`
	} `yaml:"safety_rules"`
	
	// Enhanced configuration sections for advanced rules
	Sanitization          map[string]interface{} `yaml:"sanitization"`
	QueryLimits           map[string]interface{} `yaml:"query_limits"`
	BusinessHours         map[string]interface{} `yaml:"business_hours"`
	AnalysisLimits        map[string]interface{} `yaml:"analysis_limits"`
	ResponseStatus        map[string]interface{} `yaml:"response_status"`
	AuthDecisions         map[string]interface{} `yaml:"auth_decisions"`
	MultiSource           map[string]interface{} `yaml:"multi_source"`
	ComplianceFramework   map[string]interface{} `yaml:"compliance_framework"`
	BehavioralAnalytics   map[string]interface{} `yaml:"behavioral_analytics"`
	SecurityPatterns      map[string]interface{} `yaml:"security_patterns"`
	TimeBasedSecurity     map[string]interface{} `yaml:"time_based_security"`
	OpenShiftSecurity     map[string]interface{} `yaml:"openshift_security"`
	PerformanceLimits     map[string]interface{} `yaml:"performance_limits"`
	PromptValidation      map[string]interface{} `yaml:"prompt_validation"`
	
	// Rule engine configuration
	RuleEngine            RuleEngineConfig       `yaml:"rule_engine,omitempty"`
}

// LoadValidationConfig loads validation configuration from a YAML file
func LoadValidationConfig(configPath string) (*ValidationConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config ValidationConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Apply defaults for missing configuration
	config.ApplyDefaults()

	// Validate the configuration
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", configPath, err)
	}

	return &config, nil
}

// LoadDefaultValidationConfig loads the default validation configuration
func LoadDefaultValidationConfig() (*ValidationConfig, error) {
	configPath := "configs/rules.yaml"
	return LoadValidationConfig(configPath)
}

// GetRuleEngineDefaults returns default rule engine configuration
func GetRuleEngineDefaults() RuleEngineConfig {
	return RuleEngineConfig{
		EnableParallelEvaluation: true,
		MaxConcurrentRules:       5,
		RuleTimeoutSeconds:       30,
		FailFastMode:            true,
		EnableRuleDependencies:  false,
		RulePriorities:          map[string]int{
			"schema_validation":    100,
			"required_fields":      90,
			"whitelist":           80,
			"sanitization":        70,
			"patterns":            60,
			"timeframe":           50,
			"advanced_analysis":   40,
			"multi_source":        30,
			"behavioral_analytics": 20,
			"compliance":          15,
			"performance":         10,
		},
		EnableRuleCaching: true,
		CacheTTLSeconds:   300, // 5 minutes
	}
}

// ApplyDefaults applies default values to missing configuration sections
func (config *ValidationConfig) ApplyDefaults() {
	// Apply rule engine defaults if not configured
	if config.RuleEngine.MaxConcurrentRules == 0 {
		defaults := GetRuleEngineDefaults()
		config.RuleEngine = defaults
	}
	
	// Apply safety rules defaults if empty
	if len(config.SafetyRules.AllowedLogSources) == 0 {
		config.SafetyRules.AllowedLogSources = []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver", "node-auditd"}
	}
	
	if len(config.SafetyRules.RequiredFields) == 0 {
		config.SafetyRules.RequiredFields = []string{"log_source"}
	}
	
	// Apply other defaults as needed
	if config.PerformanceLimits == nil {
		config.PerformanceLimits = map[string]interface{}{
			"max_query_complexity_score": 100,
			"max_concurrent_sources":     5,
			"max_correlation_fields":     10,
			"max_behavioral_features":    50,
			"max_memory_usage_mb":        1024,
			"max_cpu_usage_percent":      50,
			"max_execution_time_seconds": 300,
		}
	}
}

// GetConfigSection safely retrieves a configuration section
func (config *ValidationConfig) GetConfigSection(sectionName string) map[string]interface{} {
	switch sectionName {
	case "sanitization":
		return config.Sanitization
	case "query_limits":
		return config.QueryLimits
	case "business_hours":
		return config.BusinessHours
	case "analysis_limits":
		return config.AnalysisLimits
	case "response_status":
		return config.ResponseStatus
	case "auth_decisions":
		return config.AuthDecisions
	case "multi_source":
		return config.MultiSource
	case "compliance_framework":
		return config.ComplianceFramework
	case "behavioral_analytics":
		return config.BehavioralAnalytics
	case "security_patterns":
		return config.SecurityPatterns
	case "time_based_security":
		return config.TimeBasedSecurity
	case "openshift_security":
		return config.OpenShiftSecurity
	case "performance_limits":
		return config.PerformanceLimits
	case "prompt_validation":
		return config.PromptValidation
	default:
		return nil
	}
}

// ValidateConfig performs basic validation on the loaded configuration
func (config *ValidationConfig) ValidateConfig() error {
	// Validate rule engine configuration
	if config.RuleEngine.MaxConcurrentRules < 1 || config.RuleEngine.MaxConcurrentRules > 20 {
		return fmt.Errorf("max_concurrent_rules must be between 1 and 20, got %d", config.RuleEngine.MaxConcurrentRules)
	}
	
	if config.RuleEngine.RuleTimeoutSeconds < 1 || config.RuleEngine.RuleTimeoutSeconds > 300 {
		return fmt.Errorf("rule_timeout_seconds must be between 1 and 300, got %d", config.RuleEngine.RuleTimeoutSeconds)
	}
	
	if config.RuleEngine.CacheTTLSeconds < 0 || config.RuleEngine.CacheTTLSeconds > 3600 {
		return fmt.Errorf("cache_ttl_seconds must be between 0 and 3600, got %d", config.RuleEngine.CacheTTLSeconds)
	}
	
	// Validate required safety rules
	if len(config.SafetyRules.AllowedLogSources) == 0 {
		return fmt.Errorf("allowed_log_sources cannot be empty")
	}
	
	if len(config.SafetyRules.RequiredFields) == 0 {
		return fmt.Errorf("required_fields cannot be empty")
	}
	
	return nil
}
