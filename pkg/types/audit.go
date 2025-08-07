package types

import (
	"encoding/json"
	"time"
)

// StringOrArray represents a field that can be either a single string or an array of strings.
// This provides flexibility for handling fields that can accept either format.
type StringOrArray struct {
	value interface{}
}

// NewStringOrArray creates a new StringOrArray from a string or []string
func NewStringOrArray(value interface{}) *StringOrArray {
	return &StringOrArray{value: value}
}

// IsString returns true if the value is a single string
func (sa *StringOrArray) IsString() bool {
	_, ok := sa.value.(string)
	return ok
}

// IsArray returns true if the value is a string array
func (sa *StringOrArray) IsArray() bool {
	_, ok := sa.value.([]string)
	return ok
}

// GetString returns the string value if it's a single string, empty string otherwise
func (sa *StringOrArray) GetString() string {
	if str, ok := sa.value.(string); ok {
		return str
	}
	return ""
}

// GetArray returns the string array if it's an array, nil otherwise
func (sa *StringOrArray) GetArray() []string {
	if arr, ok := sa.value.([]string); ok {
		return arr
	}
	return nil
}

// GetValue returns the underlying value
func (sa *StringOrArray) GetValue() interface{} {
	return sa.value
}

// IsEmpty returns true if the value is nil or empty
func (sa *StringOrArray) IsEmpty() bool {
	if sa.value == nil {
		return true
	}
	if str, ok := sa.value.(string); ok {
		return str == ""
	}
	if arr, ok := sa.value.([]string); ok {
		return len(arr) == 0
	}
	return true
}

// MarshalJSON implements json.Marshaler interface
func (sa *StringOrArray) MarshalJSON() ([]byte, error) {
	return json.Marshal(sa.value)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (sa *StringOrArray) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		sa.value = str
		return nil
	}

	// Try to unmarshal as array
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		sa.value = arr
		return nil
	}

	return json.Unmarshal(data, &sa.value)
}

// TimeRange represents a custom time range with start and end timestamps.
// Used for precise time-based filtering of audit logs.
type TimeRange struct {
	// Start is the beginning timestamp in ISO 8601 format
	Start time.Time `json:"start" validate:"required"`

	// End is the ending timestamp in ISO 8601 format
	End time.Time `json:"end" validate:"required"`
}

// BusinessHours represents business hours filtering configuration.
// Used to filter audit logs based on business hours patterns.
type BusinessHours struct {
	// OutsideOnly indicates whether to filter for outside business hours only
	OutsideOnly bool `json:"outside_only,omitempty"`

	// StartHour is the business hours start hour (0-23)
	StartHour int `json:"start_hour" validate:"min=0,max=23"`

	// EndHour is the business hours end hour (0-23)
	EndHour int `json:"end_hour" validate:"min=0,max=23"`

	// Timezone is the timezone for business hours (default: UTC)
	Timezone string `json:"timezone,omitempty"`
}

// AnalysisConfig represents advanced analysis options for audit queries.
// Used for complex security investigations and correlation analysis.
type AnalysisConfig struct {
	// Type specifies the type of analysis to perform
	Type string `json:"type" validate:"required,oneof=multi_namespace_access excessive_reads privilege_escalation anomaly_detection correlation"`

	// GroupBy specifies fields to group results by
	GroupBy *StringOrArray `json:"group_by,omitempty"`

	// Threshold is the threshold value for analysis (e.g., number of events)
	Threshold int `json:"threshold,omitempty" validate:"min=1"`

	// TimeWindow specifies the time window for analysis
	TimeWindow string `json:"time_window,omitempty" validate:"omitempty,oneof=short medium long"`

	// SortBy specifies the field to sort results by
	SortBy string `json:"sort_by,omitempty" validate:"omitempty,oneof=timestamp user resource count"`

	// SortOrder specifies the sort direction
	SortOrder string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

// StructuredQuery represents the complete structured query for OpenShift audit log analysis.
// This struct contains all fields from the design document appendix for comprehensive audit querying.
type StructuredQuery struct {
	// LogSource specifies the source of audit logs (kube-apiserver, oauth-server, etc.)
	LogSource string `json:"log_source" validate:"required,oneof=kube-apiserver openshift-apiserver oauth-server oauth-apiserver"`

	// Verb specifies the HTTP verb(s) to filter on
	Verb StringOrArray `json:"verb,omitempty" validate:"omitempty"`

	// Resource specifies the Kubernetes resource type(s) to filter on
	Resource StringOrArray `json:"resource,omitempty" validate:"omitempty"`

	// Namespace specifies the specific namespace(s) to filter on
	Namespace StringOrArray `json:"namespace,omitempty" validate:"omitempty"`

	// User specifies the specific user(s) to filter on
	User StringOrArray `json:"user,omitempty" validate:"omitempty"`

	// Timeframe specifies the time period for filtering (today, yesterday, 1_hour_ago, etc.)
	Timeframe string `json:"timeframe,omitempty" validate:"omitempty"`

	// Limit specifies the maximum number of results to return
	Limit int `json:"limit,omitempty" validate:"omitempty,min=1,max=1000"`

	// ResponseStatus specifies HTTP response status filter
	ResponseStatus StringOrArray `json:"response_status,omitempty" validate:"omitempty"`

	// ExcludeUsers specifies user patterns to exclude from results
	ExcludeUsers []string `json:"exclude_users,omitempty" validate:"omitempty"`

	// ResourceNamePattern specifies regex pattern for resource name matching
	ResourceNamePattern string `json:"resource_name_pattern,omitempty" validate:"omitempty"`

	// UserPattern specifies regex pattern for user matching
	UserPattern string `json:"user_pattern,omitempty" validate:"omitempty"`

	// NamespacePattern specifies regex pattern for namespace matching
	NamespacePattern string `json:"namespace_pattern,omitempty" validate:"omitempty"`

	// RequestURIPattern specifies URI pattern matching
	RequestURIPattern string `json:"request_uri_pattern,omitempty" validate:"omitempty"`

	// AuthDecision specifies authentication decision filter
	AuthDecision string `json:"auth_decision,omitempty" validate:"omitempty,oneof=allow error forbid"`

	// SourceIP specifies source IP address filtering
	SourceIP StringOrArray `json:"source_ip,omitempty" validate:"omitempty"`

	// GroupBy specifies fields to group results by
	GroupBy StringOrArray `json:"group_by,omitempty" validate:"omitempty"`

	// SortBy specifies the field to sort results by
	SortBy string `json:"sort_by,omitempty" validate:"omitempty,oneof=timestamp user resource count"`

	// SortOrder specifies the sort direction
	SortOrder string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`

	// Subresource specifies Kubernetes subresource
	Subresource string `json:"subresource,omitempty" validate:"omitempty"`

	// IncludeChanges specifies whether to include before/after comparisons
	IncludeChanges bool `json:"include_changes,omitempty"`

	// TimeRange specifies custom time range with start/end timestamps
	TimeRange *TimeRange `json:"time_range,omitempty" validate:"omitempty"`

	// BusinessHours specifies business hours filtering configuration
	BusinessHours *BusinessHours `json:"business_hours,omitempty" validate:"omitempty"`

	// Analysis specifies advanced analysis options
	Analysis *AnalysisConfig `json:"analysis,omitempty" validate:"omitempty"`

	// RequestObjectFilter specifies filter based on request object content
	RequestObjectFilter string `json:"request_object_filter,omitempty" validate:"omitempty"`

	// ExcludeResources specifies resource patterns to exclude from results
	ExcludeResources []string `json:"exclude_resources,omitempty" validate:"omitempty"`

	// AuthorizationReasonPattern specifies pattern for authorization reason matching
	AuthorizationReasonPattern string `json:"authorization_reason_pattern,omitempty" validate:"omitempty"`

	// ResponseMessagePattern specifies pattern for response message matching
	ResponseMessagePattern string `json:"response_message_pattern,omitempty" validate:"omitempty"`

	// MissingAnnotation specifies annotation that should be missing
	MissingAnnotation string `json:"missing_annotation,omitempty" validate:"omitempty"`
}
