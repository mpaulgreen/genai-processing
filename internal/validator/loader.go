package validator

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ValidationConfig represents the configuration structure for validation rules
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

	return &config, nil
}

// LoadDefaultValidationConfig loads the default validation configuration
func LoadDefaultValidationConfig() (*ValidationConfig, error) {
	configPath := "configs/rules.yaml"
	return LoadValidationConfig(configPath)
}
