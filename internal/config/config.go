package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"genai-processing/internal/context"
	"genai-processing/pkg/types"
)

// AppConfig represents the main application configuration
type AppConfig struct {
	Server  ServerConfig  `yaml:"server" validate:"required"`
	Models  ModelsConfig  `yaml:"models" validate:"required"`
	Prompts PromptsConfig `yaml:"prompts" validate:"required"`
	Rules   RulesConfig   `yaml:"rules" validate:"required"`
	Context ContextConfig `yaml:"context" validate:"required"`
}

// ServerConfig defines server-related configuration
type ServerConfig struct {
	Port            string        `yaml:"port" default:"8080"`
	Host            string        `yaml:"host" default:"0.0.0.0"`
	ReadTimeout     time.Duration `yaml:"read_timeout" default:"30s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" default:"30s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" default:"10s"`
	MaxRequestSize  int64         `yaml:"max_request_size" default:"1048576"` // 1MB
}

// ModelsConfig defines model-related configuration
type ModelsConfig struct {
	DefaultProvider string                 `yaml:"default_provider" default:"claude"`
	Providers       map[string]ModelConfig `yaml:"providers" validate:"required"`
}

// ModelConfig defines configuration for a specific model provider
type ModelConfig struct {
	Provider        string            `yaml:"provider" validate:"required"`
	Endpoint        string            `yaml:"endpoint" validate:"required"`
	APIKey          string            `yaml:"api_key" validate:"required"`
	ModelName       string            `yaml:"model_name" validate:"required"`
	MaxTokens       int               `yaml:"max_tokens" default:"4000"`
	Temperature     float64           `yaml:"temperature" default:"0.1"`
	Timeout         time.Duration     `yaml:"timeout" default:"60s"`
	RetryAttempts   int               `yaml:"retry_attempts" default:"3"`
	RetryDelay      time.Duration     `yaml:"retry_delay" default:"1s"`
	Parameters      map[string]string `yaml:"parameters,omitempty"`
	InputAdapter    string            `yaml:"input_adapter" default:"generic"`
	OutputParser    string            `yaml:"output_parser" default:"generic"`
	PromptFormatter string            `yaml:"prompt_formatter" default:"generic"`
}

// PromptsConfig defines prompt-related configuration
type PromptsConfig struct {
	SystemPrompts map[string]string `yaml:"system_prompts" validate:"required"`
	Examples      []types.Example   `yaml:"examples" validate:"required"`
	Formats       PromptFormats     `yaml:"formats" validate:"required"`
	Validation    PromptValidation  `yaml:"validation" validate:"required"`
}

// PromptExample removed in favor of types.Example

// PromptFormats defines model-specific prompt formatting
type PromptFormats struct {
	Claude  PromptFormat `yaml:"claude" validate:"required"`
	OpenAI  PromptFormat `yaml:"openai" validate:"required"`
	Generic PromptFormat `yaml:"generic" validate:"required"`
}

// PromptFormat defines the structure for a specific model's prompt format
type PromptFormat struct {
	Template   string            `yaml:"template" validate:"required"`
	Parameters map[string]string `yaml:"parameters,omitempty"`
}

// PromptValidation defines validation rules for prompts
type PromptValidation struct {
	MaxInputLength  int      `yaml:"max_input_length" default:"1000"`
	MaxOutputLength int      `yaml:"max_output_length" default:"2000"`
	RequiredFields  []string `yaml:"required_fields" default:"[\"log_source\"]"`
	ForbiddenWords  []string `yaml:"forbidden_words,omitempty"`
}

// RulesConfig defines safety and validation rules configuration
type RulesConfig struct {
	SafetyRules      SafetyRules      `yaml:"safety_rules" validate:"required"`
	Sanitization     Sanitization     `yaml:"sanitization" validate:"required"`
	QueryLimits      QueryLimits      `yaml:"query_limits" validate:"required"`
	BusinessHours    BusinessHours    `yaml:"business_hours" validate:"required"`
	AnalysisLimits   AnalysisLimits   `yaml:"analysis_limits" validate:"required"`
	ResponseStatus   ResponseStatus   `yaml:"response_status" validate:"required"`
	AuthDecisions    AuthDecisions    `yaml:"auth_decisions" validate:"required"`
}

// SafetyRules defines allowed and forbidden patterns for audit queries
type SafetyRules struct {
	AllowedLogSources    []string        `yaml:"allowed_log_sources" validate:"required"`
	AllowedVerbs         []string        `yaml:"allowed_verbs" validate:"required"`
	AllowedResources     []string        `yaml:"allowed_resources" validate:"required"`
	ForbiddenPatterns    []string        `yaml:"forbidden_patterns" validate:"required"`
	TimeframeLimits      TimeframeLimits `yaml:"timeframe_limits" validate:"required"`
	RequiredFields       []string        `yaml:"required_fields" validate:"required"`
}

// TimeframeLimits defines constraints on query time ranges
type TimeframeLimits struct {
	MaxDaysBack        int      `yaml:"max_days_back" validate:"min=1"`
	DefaultLimit       int      `yaml:"default_limit" validate:"min=1"`
	MaxLimit           int      `yaml:"max_limit" validate:"min=1"`
	MinLimit           int      `yaml:"min_limit" validate:"min=1"`
	AllowedTimeframes  []string `yaml:"allowed_timeframes" validate:"required"`
}

// Sanitization defines input sanitization rules
type Sanitization struct {
	MaxQueryLength          int      `yaml:"max_query_length" validate:"min=1"`
	MaxPatternLength        int      `yaml:"max_pattern_length" validate:"min=1"`
	MaxUserPatternLength    int      `yaml:"max_user_pattern_length" validate:"min=1"`
	MaxNamespacePatternLength int    `yaml:"max_namespace_pattern_length" validate:"min=1"`
	MaxResourcePatternLength int     `yaml:"max_resource_pattern_length" validate:"min=1"`
	ValidRegexPattern       string   `yaml:"valid_regex_pattern" validate:"required"`
	ValidIPPattern          string   `yaml:"valid_ip_pattern" validate:"required"`
	ValidNamespacePattern   string   `yaml:"valid_namespace_pattern" validate:"required"`
	ValidResourcePattern    string   `yaml:"valid_resource_pattern" validate:"required"`
	ForbiddenChars          []string `yaml:"forbidden_chars" validate:"required"`
}

// QueryLimits defines limits for query arrays and fields
type QueryLimits struct {
	MaxExcludeUsers           int `yaml:"max_exclude_users" validate:"min=1"`
	MaxExcludeResources       int `yaml:"max_exclude_resources" validate:"min=1"`
	MaxGroupByFields          int `yaml:"max_group_by_fields" validate:"min=1"`
	MaxSortFields             int `yaml:"max_sort_fields" validate:"min=1"`
	MaxVerbArraySize          int `yaml:"max_verb_array_size" validate:"min=1"`
	MaxResourceArraySize      int `yaml:"max_resource_array_size" validate:"min=1"`
	MaxNamespaceArraySize     int `yaml:"max_namespace_array_size" validate:"min=1"`
	MaxUserArraySize          int `yaml:"max_user_array_size" validate:"min=1"`
	MaxResponseStatusArraySize int `yaml:"max_response_status_array_size" validate:"min=1"`
	MaxSourceIPArraySize      int `yaml:"max_source_ip_array_size" validate:"min=1"`
}

// BusinessHours defines business hours configuration
type BusinessHours struct {
	DefaultStartHour int    `yaml:"default_start_hour" validate:"min=0,max=23"`
	DefaultEndHour   int    `yaml:"default_end_hour" validate:"min=0,max=23"`
	DefaultTimezone  string `yaml:"default_timezone" validate:"required"`
	MaxHourValue     int    `yaml:"max_hour_value" validate:"min=0,max=23"`
	MinHourValue     int    `yaml:"min_hour_value" validate:"min=0,max=23"`
}

// AnalysisLimits defines limits for analysis configuration
type AnalysisLimits struct {
	MaxThresholdValue      int      `yaml:"max_threshold_value" validate:"min=1"`
	MinThresholdValue      int      `yaml:"min_threshold_value" validate:"min=1"`
	AllowedAnalysisTypes   []string `yaml:"allowed_analysis_types" validate:"required"`
	AllowedTimeWindows     []string `yaml:"allowed_time_windows" validate:"required"`
	AllowedSortFields      []string `yaml:"allowed_sort_fields" validate:"required"`
	AllowedSortOrders      []string `yaml:"allowed_sort_orders" validate:"required"`
}

// ResponseStatus defines allowed response status codes
type ResponseStatus struct {
	AllowedStatusCodes []string `yaml:"allowed_status_codes" validate:"required"`
	MinStatusCode      int      `yaml:"min_status_code" validate:"min=100,max=599"`
	MaxStatusCode      int      `yaml:"max_status_code" validate:"min=100,max=599"`
}

// AuthDecisions defines allowed authentication decisions
type AuthDecisions struct {
	AllowedDecisions []string `yaml:"allowed_decisions" validate:"required"`
}

// ContextConfig defines context manager configuration
type ContextConfig struct {
	CleanupInterval        time.Duration `yaml:"cleanup_interval" default:"5m"`
	SessionTimeout         time.Duration `yaml:"session_timeout" default:"24h"`
	MaxSessions           int           `yaml:"max_sessions" default:"10000"`
	MaxMemoryMB           int           `yaml:"max_memory_mb" default:"100"`
	EnablePersistence     bool          `yaml:"enable_persistence" default:"true"`
	PersistencePath       string        `yaml:"persistence_path" default:"./sessions"`
	PersistenceFormat     string        `yaml:"persistence_format" default:"json"`
	PersistenceInterval   time.Duration `yaml:"persistence_interval" default:"30s"`
	EnableAsyncPersistence bool         `yaml:"enable_async_persistence" default:"true"`
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Validate validates the AppConfig and returns a ValidationResult
func (c *AppConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate server configuration
	if serverResult := c.Server.Validate(); !serverResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, serverResult.Errors...)
	}

	// Validate models configuration
	if modelsResult := c.Models.Validate(); !modelsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, modelsResult.Errors...)
	}

	// Validate prompts configuration
	if promptsResult := c.Prompts.Validate(); !promptsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, promptsResult.Errors...)
	}

	// Validate rules configuration
	if rulesResult := c.Rules.Validate(); !rulesResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, rulesResult.Errors...)
	}

	// Validate context configuration
	if contextResult := c.Context.Validate(); !contextResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, contextResult.Errors...)
	}

	return result
}

// Validate validates the ServerConfig
func (c *ServerConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate port
	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid port number")
	}

	// Validate host
	if c.Host != "0.0.0.0" && c.Host != "localhost" && c.Host != "127.0.0.1" {
		if ip := net.ParseIP(c.Host); ip == nil {
			result.Valid = false
			result.Errors = append(result.Errors, "invalid host address")
		}
	}

	// Validate timeouts
	if c.ReadTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "read_timeout must be positive")
	}

	if c.WriteTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "write_timeout must be positive")
	}

	if c.IdleTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "idle_timeout must be positive")
	}

	if c.ShutdownTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "shutdown_timeout must be positive")
	}

	// Validate request size
	if c.MaxRequestSize <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_request_size must be positive")
	}

	return result
}

// Validate validates the ModelsConfig
func (c *ModelsConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate default provider exists
	if c.DefaultProvider == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "default_provider is required")
	} else if _, exists := c.Providers[c.DefaultProvider]; !exists {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("default_provider '%s' not found in providers", c.DefaultProvider))
	}

	// Validate providers
	if len(c.Providers) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one provider must be configured")
	}

	for name, provider := range c.Providers {
		if providerResult := provider.Validate(); !providerResult.Valid {
			result.Valid = false
			for _, err := range providerResult.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("provider '%s': %s", name, err))
			}
		}
	}

	return result
}

// Validate validates the ModelConfig
func (c *ModelConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate required fields
	if c.Provider == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "provider is required")
	}

	if c.Endpoint == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "endpoint is required")
	}

	// API key is only required for external providers, not for local models
	if c.Provider != "ollama" && c.Provider != "local" && c.Provider != "generic" {
		if c.APIKey == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "api_key is required")
		} else if !isValidAPIKey(c.APIKey) {
			result.Valid = false
			result.Errors = append(result.Errors, "api_key must be a valid key or environment variable placeholder")
		}
	}

	if c.ModelName == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "model_name is required")
	}

	// Validate numeric fields
	if c.MaxTokens <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_tokens must be positive")
	}

	if c.Temperature < 0.0 || c.Temperature > 2.0 {
		result.Valid = false
		result.Errors = append(result.Errors, "temperature must be between 0.0 and 2.0")
	}

	if c.Timeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "timeout must be positive")
	}

	if c.RetryAttempts < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "retry_attempts must be non-negative")
	}

	if c.RetryDelay < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "retry_delay must be non-negative")
	}

	// Validate endpoint format
	if !strings.HasPrefix(c.Endpoint, "http://") && !strings.HasPrefix(c.Endpoint, "https://") {
		result.Valid = false
		result.Errors = append(result.Errors, "endpoint must be a valid HTTP/HTTPS URL")
	}

	return result
}

// Validate validates the PromptsConfig
func (c *PromptsConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate system prompts
	if len(c.SystemPrompts) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one system prompt must be configured")
	}

	// Check for required system prompts
	requiredPrompts := []string{"base"}
	for _, required := range requiredPrompts {
		if _, exists := c.SystemPrompts[required]; !exists {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("system prompt '%s' is required", required))
		}
	}

	// Validate examples
	if len(c.Examples) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one example must be configured")
	}

	// Basic non-empty validation for examples
	for i, example := range c.Examples {
		if strings.TrimSpace(example.Input) == "" || strings.TrimSpace(example.Output) == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("example %d: input and output cannot be empty", i+1))
		}
	}

	// Validate formats
	if formatsResult := c.Formats.Validate(); !formatsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, formatsResult.Errors...)
	}

	// Validate validation rules
	if validationResult := c.Validation.Validate(); !validationResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, validationResult.Errors...)
	}

	return result
}

// PromptExample validation removed; using basic non-empty checks for types.Example in PromptsConfig.Validate

// Validate validates the PromptFormats
func (f *PromptFormats) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate Claude format
	if claudeResult := f.Claude.Validate(); !claudeResult.Valid {
		result.Valid = false
		for _, err := range claudeResult.Errors {
			result.Errors = append(result.Errors, fmt.Sprintf("claude format: %s", err))
		}
	}

	// Validate OpenAI format
	if openaiResult := f.OpenAI.Validate(); !openaiResult.Valid {
		result.Valid = false
		for _, err := range openaiResult.Errors {
			result.Errors = append(result.Errors, fmt.Sprintf("openai format: %s", err))
		}
	}

	// Validate Generic format
	if genericResult := f.Generic.Validate(); !genericResult.Valid {
		result.Valid = false
		for _, err := range genericResult.Errors {
			result.Errors = append(result.Errors, fmt.Sprintf("generic format: %s", err))
		}
	}

	return result
}

// Validate validates the PromptFormat
func (f *PromptFormat) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if strings.TrimSpace(f.Template) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "template cannot be empty")
	}

	return result
}

// Validate validates the PromptValidation
func (v *PromptValidation) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if v.MaxInputLength <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_input_length must be positive")
	}

	if v.MaxOutputLength <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_output_length must be positive")
	}

	if len(v.RequiredFields) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one required field must be specified")
	}

	return result
}

// Validate validates the RulesConfig
func (c *RulesConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate safety rules
	if safetyResult := c.SafetyRules.Validate(); !safetyResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, safetyResult.Errors...)
	}

	// Validate sanitization
	if sanitizationResult := c.Sanitization.Validate(); !sanitizationResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, sanitizationResult.Errors...)
	}

	// Validate query limits
	if queryLimitsResult := c.QueryLimits.Validate(); !queryLimitsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, queryLimitsResult.Errors...)
	}

	// Validate business hours
	if businessHoursResult := c.BusinessHours.Validate(); !businessHoursResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, businessHoursResult.Errors...)
	}

	// Validate analysis limits
	if analysisLimitsResult := c.AnalysisLimits.Validate(); !analysisLimitsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, analysisLimitsResult.Errors...)
	}

	// Validate response status
	if responseStatusResult := c.ResponseStatus.Validate(); !responseStatusResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, responseStatusResult.Errors...)
	}

	// Validate auth decisions
	if authDecisionsResult := c.AuthDecisions.Validate(); !authDecisionsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, authDecisionsResult.Errors...)
	}

	return result
}

// Validate validates the SafetyRules
func (s *SafetyRules) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if len(s.AllowedLogSources) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one allowed log source must be specified")
	}

	if len(s.AllowedVerbs) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one allowed verb must be specified")
	}

	if len(s.RequiredFields) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one required field must be specified")
	}

	// Validate timeframe limits
	if timeframeLimitsResult := s.TimeframeLimits.Validate(); !timeframeLimitsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, timeframeLimitsResult.Errors...)
	}

	return result
}

// Validate validates the TimeframeLimits
func (t *TimeframeLimits) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if t.MaxDaysBack <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_days_back must be positive")
	}

	if t.DefaultLimit <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "default_limit must be positive")
	}

	if t.MaxLimit <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_limit must be positive")
	}

	if t.MinLimit <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "min_limit must be positive")
	}

	if t.DefaultLimit > t.MaxLimit {
		result.Valid = false
		result.Errors = append(result.Errors, "default_limit cannot be greater than max_limit")
	}

	if t.MinLimit > t.DefaultLimit {
		result.Valid = false
		result.Errors = append(result.Errors, "min_limit cannot be greater than default_limit")
	}

	if len(t.AllowedTimeframes) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one allowed timeframe must be specified")
	}

	return result
}

// Validate validates the Sanitization
func (s *Sanitization) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if s.MaxQueryLength <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_query_length must be positive")
	}

	if s.MaxPatternLength <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_pattern_length must be positive")
	}

	if s.ValidRegexPattern == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "valid_regex_pattern is required")
	}

	if s.ValidIPPattern == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "valid_ip_pattern is required")
	}

	return result
}

// Validate validates the QueryLimits
func (q *QueryLimits) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if q.MaxExcludeUsers <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_exclude_users must be positive")
	}

	if q.MaxGroupByFields <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_group_by_fields must be positive")
	}

	return result
}

// Validate validates the BusinessHours
func (b *BusinessHours) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if b.DefaultStartHour < 0 || b.DefaultStartHour > 23 {
		result.Valid = false
		result.Errors = append(result.Errors, "default_start_hour must be between 0 and 23")
	}

	if b.DefaultEndHour < 0 || b.DefaultEndHour > 23 {
		result.Valid = false
		result.Errors = append(result.Errors, "default_end_hour must be between 0 and 23")
	}

	if b.DefaultTimezone == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "default_timezone is required")
	}

	return result
}

// Validate validates the AnalysisLimits
func (a *AnalysisLimits) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if a.MaxThresholdValue <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_threshold_value must be positive")
	}

	if a.MinThresholdValue <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "min_threshold_value must be positive")
	}

	if a.MinThresholdValue > a.MaxThresholdValue {
		result.Valid = false
		result.Errors = append(result.Errors, "min_threshold_value cannot be greater than max_threshold_value")
	}

	if len(a.AllowedAnalysisTypes) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one allowed analysis type must be specified")
	}

	return result
}

// Validate validates the ResponseStatus
func (r *ResponseStatus) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if len(r.AllowedStatusCodes) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one allowed status code must be specified")
	}

	if r.MinStatusCode < 100 || r.MinStatusCode > 599 {
		result.Valid = false
		result.Errors = append(result.Errors, "min_status_code must be between 100 and 599")
	}

	if r.MaxStatusCode < 100 || r.MaxStatusCode > 599 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_status_code must be between 100 and 599")
	}

	if r.MinStatusCode > r.MaxStatusCode {
		result.Valid = false
		result.Errors = append(result.Errors, "min_status_code cannot be greater than max_status_code")
	}

	return result
}

// Validate validates the AuthDecisions
func (a *AuthDecisions) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if len(a.AllowedDecisions) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one allowed auth decision must be specified")
	}

	return result
}

// Validate validates the ContextConfig
func (c *ContextConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if c.CleanupInterval <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "cleanup_interval must be positive")
	}

	if c.SessionTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "session_timeout must be positive")
	}

	if c.MaxSessions <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_sessions must be positive")
	}

	if c.MaxMemoryMB <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_memory_mb must be positive")
	}

	if c.PersistencePath == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "persistence_path is required")
	}

	if c.PersistenceFormat != "json" && c.PersistenceFormat != "gob" {
		result.Valid = false
		result.Errors = append(result.Errors, "persistence_format must be 'json' or 'gob'")
	}

	if c.PersistenceInterval <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "persistence_interval must be positive")
	}

	return result
}

// isValidAPIKey checks if the API key is valid (either a real key or an environment variable placeholder)
func isValidAPIKey(apiKey string) bool {
	// Allow empty keys for local models
	if apiKey == "" {
		return true
	}

	// Check for environment variable placeholders with fallback values
	// Pattern: ${VAR_NAME:-default_value}
	if strings.HasPrefix(apiKey, "${") && strings.Contains(apiKey, ":-") && strings.HasSuffix(apiKey, "}") {
		return true
	}

	// Check for simple environment variable placeholders
	// Pattern: ${VAR_NAME}
	if strings.HasPrefix(apiKey, "${") && strings.HasSuffix(apiKey, "}") {
		return true
	}

	// Allow actual API keys (non-empty strings that don't look like placeholders)
	// Must not start with ${ and must not end with } unless it's a complete placeholder
	if len(strings.TrimSpace(apiKey)) > 0 && !strings.HasPrefix(apiKey, "${") && !strings.HasSuffix(apiKey, "}") {
		return true
	}

	return false
}

// GetDefaultConfig returns a default AppConfig with sensible defaults
func GetDefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Port:            "8080",
			Host:            "0.0.0.0",
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			MaxRequestSize:  1048576, // 1MB
		},
		Models: ModelsConfig{
			DefaultProvider: "claude",
			Providers: map[string]ModelConfig{
				"claude": {
					Provider:        "anthropic",
					Endpoint:        "https://api.anthropic.com/v1/messages",
					APIKey:          "${CLAUDE_API_KEY:-placeholder-key}",
					ModelName:       "claude-3-5-sonnet-20241022",
					MaxTokens:       4000,
					Temperature:     0.1,
					Timeout:         60 * time.Second,
					RetryAttempts:   3,
					RetryDelay:      1 * time.Second,
					InputAdapter:    "claude_input_adapter",
					OutputParser:    "claude_extractor",
					PromptFormatter: "claude_formatter",
				},
				"openai": {
					Provider:        "openai",
					Endpoint:        "https://api.openai.com/v1/chat/completions",
					APIKey:          "${OPENAI_API_KEY:-placeholder-key}",
					ModelName:       "gpt-4",
					MaxTokens:       4000,
					Temperature:     0.1,
					Timeout:         60 * time.Second,
					RetryAttempts:   3,
					RetryDelay:      1 * time.Second,
					InputAdapter:    "openai_input_adapter",
					OutputParser:    "openai_extractor",
					PromptFormatter: "openai_formatter",
				},
				"local_llama": {
					Provider:        "ollama",
					Endpoint:        "http://localhost:11434",
					APIKey:          "",
					ModelName:       "llama3.1:8b",
					MaxTokens:       4000,
					Temperature:     0.1,
					Timeout:         60 * time.Second,
					RetryAttempts:   3,
					RetryDelay:      1 * time.Second,
					InputAdapter:    "ollama_input_adapter",
					OutputParser:    "generic_extractor",
					PromptFormatter: "generic_formatter",
				},
			},
		},
		Prompts: PromptsConfig{
			SystemPrompts: map[string]string{
				"base": `You are an OpenShift audit query specialist. Convert natural language 
queries into structured JSON parameters for audit log analysis.

Always respond with valid JSON only. Do not include any markdown formatting,
explanations, or additional text outside the JSON structure.`,
			},
			Examples: []types.Example{
				{
					Input: "Who deleted the customer CRD yesterday?",
					Output: `{
  "log_source": "kube-apiserver",
  "verb": "delete",
  "resource": "customresourcedefinitions",
  "resource_name_pattern": "customer",
  "timeframe": "yesterday",
  "exclude_users": ["system:"],
  "limit": 20
}`,
				},
				{
					Input: "Show me all failed authentication attempts in the last hour",
					Output: `{
  "log_source": "oauth-server",
  "timeframe": "1_hour_ago",
  "auth_decision": "error",
  "limit": 20
}`,
				},
			},
			Formats: PromptFormats{
				Claude: PromptFormat{
					Template: `<instructions>
{system_prompt}
</instructions>

<examples>
{examples}
</examples>

<query>
{query}
</query>

JSON Response:`,
				},
				OpenAI: PromptFormat{
					Template: "System: {system_prompt}\n\nExamples:\n{examples}\n\nUser: Convert this query to JSON: {query}",
				},
				Generic: PromptFormat{
					Template: `{system_prompt}

Examples:
{examples}

Query: {query}

JSON Response:`,
				},
			},
			Validation: PromptValidation{
				MaxInputLength:  1000,
				MaxOutputLength: 2000,
				RequiredFields:  []string{"log_source"},
				ForbiddenWords:  []string{"rm -rf", "delete --all", "system:admin"},
			},
		},
		Rules:   *GetDefaultRulesConfig(),
		Context: *GetDefaultContextConfig(),
	}
}

// GetDefaultRulesConfig returns a default rules configuration
func GetDefaultRulesConfig() *RulesConfig {
	return &RulesConfig{
		SafetyRules: SafetyRules{
			AllowedLogSources: []string{
				"kube-apiserver",
				"openshift-apiserver",
				"oauth-server",
				"oauth-apiserver",
			},
			AllowedVerbs: []string{
				"get", "list", "create", "update", "patch", "delete", "watch",
			},
			AllowedResources: []string{
				"pods", "services", "deployments", "configmaps", "secrets",
				"namespaces", "serviceaccounts", "roles", "rolebindings",
				"clusterroles", "clusterrolebindings", "customresourcedefinitions",
				"persistentvolumeclaims", "persistentvolumes", "networkpolicies",
				"events", "nodes", "replicasets", "statefulsets", "daemonsets",
				"ingresses", "routes", "builds", "buildconfigs", "imagestreams",
				"imagestreamtags", "projects", "users", "groups", "oauthclients",
			},
			ForbiddenPatterns: []string{
				"rm -rf", "delete --all", "delete --force", "system:admin",
				"system:masters", "cluster-admin",
			},
			TimeframeLimits: TimeframeLimits{
				MaxDaysBack:   90,
				DefaultLimit:  20,
				MaxLimit:      1000,
				MinLimit:      1,
				AllowedTimeframes: []string{
					"today", "yesterday", "1_hour_ago", "2_hours_ago", "3_hours_ago",
					"6_hours_ago", "12_hours_ago", "1_day_ago", "2_days_ago",
					"3_days_ago", "7_days_ago", "14_days_ago", "30_days_ago",
					"60_days_ago", "90_days_ago",
				},
			},
			RequiredFields: []string{"log_source"},
		},
		Sanitization: Sanitization{
			MaxQueryLength:            10000,
			MaxPatternLength:          500,
			MaxUserPatternLength:      200,
			MaxNamespacePatternLength: 200,
			MaxResourcePatternLength:  200,
			ValidRegexPattern:         "^[a-zA-Z0-9\\-_\\*\\.\\?\\+\\[\\]\\{\\}\\(\\)\\|\\\\/\\s]+$",
			ValidIPPattern:            "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$",
			ValidNamespacePattern:     "^[a-z0-9]([a-z0-9\\-]*[a-z0-9])?$",
			ValidResourcePattern:      "^[a-z]([a-z0-9\\-]*[a-z0-9])?$",
			ForbiddenChars: []string{
				"<", ">", "&", "\"", "'", "`", "|", ";", "$", "!", "@", "#", "%", "^", "=", "~",
			},
		},
		QueryLimits: QueryLimits{
			MaxExcludeUsers:            50,
			MaxExcludeResources:        50,
			MaxGroupByFields:           5,
			MaxSortFields:              3,
			MaxVerbArraySize:           10,
			MaxResourceArraySize:       20,
			MaxNamespaceArraySize:      50,
			MaxUserArraySize:           100,
			MaxResponseStatusArraySize: 10,
			MaxSourceIPArraySize:       20,
		},
		BusinessHours: BusinessHours{
			DefaultStartHour: 9,
			DefaultEndHour:   17,
			DefaultTimezone:  "UTC",
			MaxHourValue:     23,
			MinHourValue:     0,
		},
		AnalysisLimits: AnalysisLimits{
			MaxThresholdValue: 10000,
			MinThresholdValue: 1,
			AllowedAnalysisTypes: []string{
				"multi_namespace_access", "excessive_reads", "privilege_escalation",
				"anomaly_detection", "correlation",
			},
			AllowedTimeWindows: []string{"short", "medium", "long"},
			AllowedSortFields:  []string{"timestamp", "user", "resource", "count"},
			AllowedSortOrders:  []string{"asc", "desc"},
		},
		ResponseStatus: ResponseStatus{
			AllowedStatusCodes: []string{
				"200", "201", "204", "400", "401", "403", "404", "409", "422", "500", "502", "503", "504",
			},
			MinStatusCode: 100,
			MaxStatusCode: 599,
		},
		AuthDecisions: AuthDecisions{
			AllowedDecisions: []string{"allow", "error", "forbid"},
		},
	}
}

// GetDefaultContextConfig returns a default context configuration
func GetDefaultContextConfig() *ContextConfig {
	return &ContextConfig{
		CleanupInterval:        5 * time.Minute,
		SessionTimeout:         24 * time.Hour,
		MaxSessions:           10000,
		MaxMemoryMB:           100,
		EnablePersistence:     true,
		PersistencePath:       "./sessions",
		PersistenceFormat:     "json",
		PersistenceInterval:   30 * time.Second,
		EnableAsyncPersistence: true,
	}
}

// ToContextManagerConfig converts ContextConfig to ContextManagerConfig for backward compatibility
func (c *ContextConfig) ToContextManagerConfig() *context.ContextManagerConfig {
	return &context.ContextManagerConfig{
		CleanupInterval:        c.CleanupInterval,
		SessionTimeout:         c.SessionTimeout,
		MaxSessions:           c.MaxSessions,
		MaxMemoryMB:           c.MaxMemoryMB,
		EnablePersistence:     c.EnablePersistence,
		PersistencePath:       c.PersistencePath,
		PersistenceFormat:     c.PersistenceFormat,
		PersistenceInterval:   c.PersistenceInterval,
		EnableAsyncPersistence: c.EnableAsyncPersistence,
	}
}
