package types

import (
	"encoding/json"
	"strings"
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

// Value is exported accessor to support construction in other packages without
// relying on struct literals. Prefer NewStringOrArray for creation.
func (sa StringOrArray) Value() interface{} { return sa.value }

// IsEmpty returns true if the value is nil or empty
func (sa *StringOrArray) IsEmpty() bool {
	if sa == nil {
		return true
	}
	if sa.value == nil {
		return true
	}
	if str, ok := sa.value.(string); ok {
		return strings.TrimSpace(str) == ""
	}
	if arr, ok := sa.value.([]string); ok {
		if len(arr) == 0 {
			return true
		}
		// Consider empty if all entries are empty/whitespace
		for _, s := range arr {
			if strings.TrimSpace(s) != "" {
				return false
			}
		}
		return true
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

// StatisticalAnalysisConfig represents statistical analysis parameters for behavioral analytics.
type StatisticalAnalysisConfig struct {
	// PatternDeviationThreshold is the threshold for pattern deviation analysis (0.1-10.0)
	PatternDeviationThreshold float64 `json:"pattern_deviation_threshold,omitempty" validate:"omitempty,min=0.1,max=10.0"`

	// ConfidenceInterval is the confidence interval for statistical analysis (0.5-0.99)
	ConfidenceInterval float64 `json:"confidence_interval,omitempty" validate:"omitempty,min=0.5,max=0.99"`

	// SampleSizeMinimum is the minimum sample size for statistical validity
	SampleSizeMinimum int `json:"sample_size_minimum,omitempty" validate:"omitempty,min=10"`

	// BaselineWindow specifies the time window for baseline establishment
	BaselineWindow string `json:"baseline_window,omitempty"`
}

// MultiSourceConfig represents multi-source correlation configuration.
// Used for correlating events across different OpenShift audit log sources.
// Example: Correlating kube-apiserver events with oauth-server authentication events.
type MultiSourceConfig struct {
	// PrimarySource is the main audit log source for the query
	PrimarySource string `json:"primary_source" validate:"required,oneof=kube-apiserver openshift-apiserver oauth-server oauth-apiserver node-auditd"`

	// SecondarySources are additional log sources to correlate with the primary source
	SecondarySources []string `json:"secondary_sources,omitempty" validate:"omitempty"`

	// CorrelationWindow is the time window for correlating events across sources
	CorrelationWindow string `json:"correlation_window,omitempty" validate:"omitempty"`

	// CorrelationFields are the fields to use for correlation (user, source_ip, timestamp, etc.)
	CorrelationFields []string `json:"correlation_fields,omitempty" validate:"omitempty"`

	// JoinType specifies how to join the correlation results (inner, left, right, full)
	JoinType string `json:"join_type,omitempty" validate:"omitempty,oneof=inner left right full"`
}

// AdvancedAnalysisConfig represents enhanced analysis configuration for complex security investigations.
// This replaces the basic AnalysisConfig with comprehensive threat hunting and security analysis capabilities.
type AdvancedAnalysisConfig struct {
	// Type specifies the type of analysis to perform (supports all advanced analysis types)
	Type string `json:"type" validate:"required"`

	// KillChainPhase specifies the MITRE ATT&CK kill chain phase for APT detection
	KillChainPhase string `json:"kill_chain_phase,omitempty" validate:"omitempty"`

	// MultiStageCorrelation enables multi-stage attack correlation analysis
	MultiStageCorrelation bool `json:"multi_stage_correlation,omitempty"`

	// StatisticalAnalysis contains parameters for statistical and behavioral analysis
	StatisticalAnalysis *StatisticalAnalysisConfig `json:"statistical_analysis,omitempty"`

	// Threshold is the threshold value for analysis (e.g., number of events)
	Threshold int `json:"threshold,omitempty" validate:"omitempty,min=1"`

	// TimeWindow specifies the time window for analysis
	TimeWindow string `json:"time_window,omitempty" validate:"omitempty"`

	// GroupBy specifies fields to group results by
	GroupBy *StringOrArray `json:"group_by,omitempty"`

	// SortBy specifies the field to sort results by
	SortBy string `json:"sort_by,omitempty" validate:"omitempty,oneof=timestamp user resource count namespace"`

	// SortOrder specifies the sort direction
	SortOrder string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

// RiskScoringConfig represents risk scoring algorithm configuration.
type RiskScoringConfig struct {
	// Enabled indicates whether risk scoring is enabled
	Enabled bool `json:"enabled,omitempty"`

	// Algorithm specifies the risk scoring algorithm (weighted_sum, composite, ml_based)
	Algorithm string `json:"algorithm,omitempty" validate:"omitempty,oneof=weighted_sum composite ml_based"`

	// RiskFactors are the factors to consider in risk calculation
	RiskFactors []string `json:"risk_factors,omitempty"`

	// WeightingScheme defines how risk factors are weighted
	WeightingScheme map[string]float64 `json:"weighting_scheme,omitempty"`
}

// AnomalyDetectionConfig represents anomaly detection algorithm configuration.
type AnomalyDetectionConfig struct {
	// Algorithm specifies the anomaly detection algorithm
	Algorithm string `json:"algorithm,omitempty" validate:"omitempty,oneof=isolation_forest z_score statistical threshold_based"`

	// Contamination is the expected proportion of anomalies (0.0-1.0)
	Contamination float64 `json:"contamination,omitempty" validate:"omitempty,min=0.0,max=1.0"`

	// Sensitivity controls the sensitivity of anomaly detection (0.0-1.0)
	Sensitivity float64 `json:"sensitivity,omitempty" validate:"omitempty,min=0.0,max=1.0"`

	// Threshold is the anomaly threshold value
	Threshold float64 `json:"threshold,omitempty"`
}

// BehavioralAnalysisConfig represents user and system behavior analytics configuration.
// Used for user profiling, baseline establishment, and behavioral anomaly detection.
type BehavioralAnalysisConfig struct {
	// UserProfiling enables user behavior profiling and baseline establishment
	UserProfiling bool `json:"user_profiling,omitempty"`

	// BaselineComparison enables comparison against established behavioral baselines
	BaselineComparison bool `json:"baseline_comparison,omitempty"`

	// RiskScoring contains risk scoring algorithm configuration
	RiskScoring *RiskScoringConfig `json:"risk_scoring,omitempty"`

	// AnomalyDetection contains anomaly detection algorithm configuration
	AnomalyDetection *AnomalyDetectionConfig `json:"anomaly_detection,omitempty"`

	// BaselineWindow specifies the time window for baseline establishment
	BaselineWindow string `json:"baseline_window,omitempty"`

	// LearningPeriod is the period for learning normal behavior patterns
	LearningPeriod string `json:"learning_period,omitempty"`
}

// ThreatIntelligenceConfig represents threat intelligence correlation and analysis configuration.
// Used for IOC correlation, threat actor attribution, and attack pattern matching.
type ThreatIntelligenceConfig struct {
	// IOCCorrelation enables indicator of compromise correlation
	IOCCorrelation bool `json:"ioc_correlation,omitempty"`

	// AttackPatternMatching enables MITRE ATT&CK pattern matching
	AttackPatternMatching bool `json:"attack_pattern_matching,omitempty"`

	// ThreatActorAttribution enables threat actor attribution analysis
	ThreatActorAttribution bool `json:"threat_actor_attribution,omitempty"`

	// FeedSources specifies the threat intelligence feed sources to use
	FeedSources []string `json:"feed_sources,omitempty"`

	// ConfidenceThreshold is the minimum confidence threshold for threat matches (0.0-1.0)
	ConfidenceThreshold float64 `json:"confidence_threshold,omitempty" validate:"omitempty,min=0.0,max=1.0"`

	// TTPAnalysis enables tactics, techniques, and procedures analysis
	TTPAnalysis bool `json:"ttp_analysis,omitempty"`
}

// FeatureEngineeringConfig represents machine learning feature engineering configuration.
type FeatureEngineeringConfig struct {
	// TemporalFeatures enables time-based feature extraction
	TemporalFeatures bool `json:"temporal_features,omitempty"`

	// BehavioralFeatures enables behavioral pattern feature extraction
	BehavioralFeatures bool `json:"behavioral_features,omitempty"`

	// NetworkFeatures enables network-based feature extraction
	NetworkFeatures bool `json:"network_features,omitempty"`

	// SequentialFeatures enables sequence-based feature extraction
	SequentialFeatures bool `json:"sequential_features,omitempty"`
}

// MachineLearningConfig represents machine learning analysis parameters and model configuration.
// Used for ML-based anomaly detection, behavioral analysis, and predictive security modeling.
type MachineLearningConfig struct {
	// ModelType specifies the machine learning model type
	ModelType string `json:"model_type,omitempty" validate:"omitempty,oneof=anomaly_detection classification clustering regression time_series"`

	// FeatureEngineering contains feature engineering configuration
	FeatureEngineering *FeatureEngineeringConfig `json:"feature_engineering,omitempty"`

	// TrainingWindow specifies the time window for model training
	TrainingWindow string `json:"training_window,omitempty"`

	// PredictionThreshold is the threshold for ML predictions (0.0-1.0)
	PredictionThreshold float64 `json:"prediction_threshold,omitempty" validate:"omitempty,min=0.0,max=1.0"`

	// ModelParameters contains model-specific parameters
	ModelParameters map[string]interface{} `json:"model_parameters,omitempty"`

	// ValidationMethod specifies the model validation method
	ValidationMethod string `json:"validation_method,omitempty" validate:"omitempty,oneof=cross_validation holdout time_series_split"`
}

// RapidOperationsConfig represents configuration for rapid operations detection.
type RapidOperationsConfig struct {
	// Threshold is the number of operations within the time window to trigger detection
	Threshold int `json:"threshold" validate:"required,min=1"`

	// TimeWindow is the time window for rapid operations detection
	TimeWindow string `json:"time_window" validate:"required"`
}

// DetectionCriteriaConfig represents specific detection criteria for security analysis.
// Used for configuring various security detection algorithms and thresholds.
type DetectionCriteriaConfig struct {
	// RapidOperations configures rapid operations detection
	RapidOperations *RapidOperationsConfig `json:"rapid_operations,omitempty"`

	// PrivilegeEscalationIndicators enables privilege escalation detection
	PrivilegeEscalationIndicators bool `json:"privilege_escalation_indicators,omitempty"`

	// LateralMovementPatterns enables lateral movement detection
	LateralMovementPatterns bool `json:"lateral_movement_patterns,omitempty"`

	// DataAccessAnomalies enables data access anomaly detection
	DataAccessAnomalies bool `json:"data_access_anomalies,omitempty"`

	// ReconnaissanceIndicators enables reconnaissance activity detection
	ReconnaissanceIndicators bool `json:"reconnaissance_indicators,omitempty"`

	// UnusualAPIPatterns enables unusual API usage pattern detection
	UnusualAPIPatterns bool `json:"unusual_api_patterns,omitempty"`

	// PersistenceMechanisms enables persistence mechanism detection
	PersistenceMechanisms bool `json:"persistence_mechanisms,omitempty"`

	// DefenseEvasion enables defense evasion technique detection
	DefenseEvasion bool `json:"defense_evasion,omitempty"`
}

// SecurityContextConfig represents security context and constraint analysis configuration.
// Used for OpenShift-specific security monitoring including SCC violations and pod security standards.
type SecurityContextConfig struct {
	// SCCViolations enables Security Context Constraint violation monitoring
	SCCViolations bool `json:"scc_violations,omitempty"`

	// PodSecurityStandards specifies the pod security standard to enforce
	PodSecurityStandards string `json:"pod_security_standards,omitempty" validate:"omitempty,oneof=privileged baseline restricted"`

	// PrivilegeAnalysis enables privilege analysis and escalation detection
	PrivilegeAnalysis bool `json:"privilege_analysis,omitempty"`

	// CapabilityMonitoring specifies Linux capabilities to monitor
	CapabilityMonitoring []string `json:"capability_monitoring,omitempty"`

	// HostAccessMonitoring enables host access pattern monitoring
	HostAccessMonitoring bool `json:"host_access_monitoring,omitempty"`

	// SELinuxViolations enables SELinux policy violation detection
	SELinuxViolations bool `json:"selinux_violations,omitempty"`
}

// ComplianceReportingConfig represents compliance reporting configuration.
type ComplianceReportingConfig struct {
	// Format specifies the compliance report format
	Format string `json:"format,omitempty" validate:"omitempty,oneof=summary detailed audit_trail evidence_chain"`

	// IncludeEvidence indicates whether to include evidence in reports
	IncludeEvidence bool `json:"include_evidence,omitempty"`

	// RetentionPeriod specifies how long to retain compliance evidence
	RetentionPeriod string `json:"retention_period,omitempty"`

	// DigitalSignature enables digital signing of compliance reports
	DigitalSignature bool `json:"digital_signature,omitempty"`
}

// ComplianceFrameworkConfig represents compliance framework monitoring configuration.
// Used for automated compliance monitoring for SOX, PCI-DSS, GDPR, HIPAA, and other regulations.
type ComplianceFrameworkConfig struct {
	// Standards specifies the compliance standards to monitor
	Standards []string `json:"standards,omitempty" validate:"omitempty"`

	// Controls specifies the specific controls to monitor within the standards
	Controls []string `json:"controls,omitempty"`

	// Reporting contains compliance reporting configuration
	Reporting *ComplianceReportingConfig `json:"reporting,omitempty"`

	// AuditTrail enables comprehensive audit trail generation
	AuditTrail bool `json:"audit_trail,omitempty"`

	// ViolationThreshold specifies the threshold for compliance violations
	ViolationThreshold int `json:"violation_threshold,omitempty" validate:"omitempty,min=1"`

	// EvidenceCollection enables automated evidence collection
	EvidenceCollection bool `json:"evidence_collection,omitempty"`
}

// TemporalAnalysisConfig represents advanced time-based pattern analysis configuration.
// Used for seasonal pattern detection, anomaly detection in temporal sequences, and trend analysis.
type TemporalAnalysisConfig struct {
	// PatternType specifies the type of temporal pattern to analyze
	PatternType string `json:"pattern_type,omitempty" validate:"omitempty,oneof=periodic irregular trending cyclical seasonal"`

	// IntervalDetection enables automatic interval detection in time series
	IntervalDetection bool `json:"interval_detection,omitempty"`

	// AnomalyThreshold is the threshold for temporal anomaly detection
	AnomalyThreshold float64 `json:"anomaly_threshold,omitempty" validate:"omitempty,min=0.1,max=10.0"`

	// BaselineWindow specifies the baseline window for temporal analysis
	BaselineWindow string `json:"baseline_window,omitempty"`

	// SeasonalityDetection enables seasonal pattern detection
	SeasonalityDetection bool `json:"seasonality_detection,omitempty"`

	// TrendAnalysis enables trend analysis and forecasting
	TrendAnalysis bool `json:"trend_analysis,omitempty"`

	// CorrelationWindow specifies the time window for temporal correlation
	CorrelationWindow string `json:"correlation_window,omitempty"`
}

// StructuredQuery represents the complete structured query for OpenShift audit log analysis.
// This struct contains all fields from the enhanced JSON schema for comprehensive audit querying,
// including advanced threat hunting, behavioral analytics, and compliance monitoring capabilities.
type StructuredQuery struct {
	// LogSource specifies the source of audit logs (kube-apiserver, oauth-server, etc.)
	// Enhanced to include node-auditd for system-level monitoring
	LogSource string `json:"log_source" validate:"required,oneof=kube-apiserver openshift-apiserver oauth-server oauth-apiserver node-auditd"`

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

	// CorrelationFields specifies fields to correlate across log entries
	// Used for basic correlation support before multi-source correlation
	CorrelationFields []string `json:"correlation_fields,omitempty" validate:"omitempty"`

	// GroupBy specifies fields to group results by
	GroupBy StringOrArray `json:"group_by,omitempty" validate:"omitempty"`

	// SortBy specifies the field to sort results by
	SortBy string `json:"sort_by,omitempty" validate:"omitempty,oneof=timestamp user resource count namespace"`

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

	// === ADVANCED SECURITY MONITORING FIELDS ===

	// MultiSource specifies multi-source correlation configuration
	// Enables correlation of events across different OpenShift audit log sources
	// Example: Correlating kube-apiserver resource access with oauth-server authentication
	MultiSource *MultiSourceConfig `json:"multi_source,omitempty" validate:"omitempty"`

	// Analysis specifies enhanced analysis configuration for complex security investigations
	// Replaces the basic AnalysisConfig with comprehensive threat hunting capabilities
	// Supports APT detection, kill chain analysis, and statistical analysis
	Analysis *AdvancedAnalysisConfig `json:"analysis,omitempty" validate:"omitempty"`

	// BehavioralAnalysis configures user and system behavior analytics
	// Enables user profiling, baseline establishment, risk scoring, and anomaly detection
	// Used for detecting unusual user behavior patterns and potential insider threats
	BehavioralAnalysis *BehavioralAnalysisConfig `json:"behavioral_analysis,omitempty" validate:"omitempty"`

	// ThreatIntelligence configures threat intelligence correlation and analysis
	// Enables IOC correlation, threat actor attribution, and attack pattern matching
	// Integrates with MITRE ATT&CK framework and external threat feeds
	ThreatIntelligence *ThreatIntelligenceConfig `json:"threat_intelligence,omitempty" validate:"omitempty"`

	// MachineLearning configures machine learning analysis parameters
	// Enables ML-based anomaly detection, behavioral analysis, and predictive modeling
	// Supports various ML algorithms for security event analysis
	MachineLearning *MachineLearningConfig `json:"machine_learning,omitempty" validate:"omitempty"`

	// DetectionCriteria configures specific detection criteria for security analysis
	// Enables various security detection algorithms including rapid operations,
	// privilege escalation, lateral movement, and reconnaissance detection
	DetectionCriteria *DetectionCriteriaConfig `json:"detection_criteria,omitempty" validate:"omitempty"`

	// SecurityContext configures security context and constraint analysis
	// Enables OpenShift-specific security monitoring including SCC violations,
	// pod security standards, and privilege analysis
	SecurityContext *SecurityContextConfig `json:"security_context,omitempty" validate:"omitempty"`

	// ComplianceFramework configures compliance framework monitoring
	// Enables automated compliance monitoring for SOX, PCI-DSS, GDPR, HIPAA
	// Supports compliance reporting and evidence collection
	ComplianceFramework *ComplianceFrameworkConfig `json:"compliance_framework,omitempty" validate:"omitempty"`

	// TemporalAnalysis configures advanced time-based pattern analysis
	// Enables seasonal pattern detection, anomaly detection in temporal sequences,
	// trend analysis, and cyclical pattern recognition
	TemporalAnalysis *TemporalAnalysisConfig `json:"temporal_analysis,omitempty" validate:"omitempty"`
}
