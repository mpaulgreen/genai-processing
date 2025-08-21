package types

// InputValidationConfig defines consolidated input validation configuration
type InputValidationConfig struct {
	Enabled bool `yaml:"enabled"`
	
	RequiredFields      RequiredFieldsConfig      `yaml:"required_fields"`
	CharacterValidation CharacterValidationConfig `yaml:"character_validation"`
	SecurityPatterns    SecurityPatternsConfig    `yaml:"security_patterns"`
	FieldValues         FieldValuesConfig         `yaml:"field_values"`
	PerformanceLimits   PerformanceLimitsConfig   `yaml:"performance_limits"`
}

// RequiredFieldsConfig defines required field validation
type RequiredFieldsConfig struct {
	Mandatory   []string `yaml:"mandatory"`
	Conditional []string `yaml:"conditional"`
}

// CharacterValidationConfig defines character and format validation
type CharacterValidationConfig struct {
	MaxQueryLength     int      `yaml:"max_query_length"`
	MaxPatternLength   int      `yaml:"max_pattern_length"`
	ForbiddenChars     []string `yaml:"forbidden_chars"`
	ValidRegexPattern  string   `yaml:"valid_regex_pattern"`
	ValidIPPattern     string   `yaml:"valid_ip_pattern"`
}

// SecurityPatternsConfig defines security pattern validation
type SecurityPatternsConfig struct {
	ForbiddenPatterns []string `yaml:"forbidden_patterns"`
}

// FieldValuesConfig defines allowed field values
type FieldValuesConfig struct {
	AllowedLogSources     []string `yaml:"allowed_log_sources"`
	AllowedVerbs          []string `yaml:"allowed_verbs"`
	AllowedResources      []string `yaml:"allowed_resources"`
	AllowedAuthDecisions  []string `yaml:"allowed_auth_decisions"`
	AllowedResponseStatus []string `yaml:"allowed_response_status"`
}

// PerformanceLimitsConfig defines performance and limit validation
type PerformanceLimitsConfig struct {
	MaxResultLimit      int      `yaml:"max_result_limit"`
	MaxArrayElements    int      `yaml:"max_array_elements"`
	MaxDaysBack         int      `yaml:"max_days_back"`
	AllowedTimeframes   []string `yaml:"allowed_timeframes"`
}