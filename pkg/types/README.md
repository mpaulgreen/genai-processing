# Types Package

This package contains all type definitions for the GenAI-Powered OpenShift Audit Query System. It provides comprehensive data structures for request/response handling, model management, session context, and the enhanced audit query schema with advanced security monitoring capabilities.

## Overview

The types package serves as the central type definition hub for the entire GenAI processing system, providing:

### Core Functionality
- **Request/Response Processing**: Complete pipeline for natural language to structured query conversion
- **Model Management**: LLM provider configuration, token usage tracking, and performance metrics
- **Session Context**: Multi-turn conversation support with reference resolution
- **Audit Query Schema**: Enhanced schema supporting basic to enterprise-grade security analysis

### Advanced Security Features
- **Multi-source correlation** across different OpenShift audit log sources
- **Advanced threat hunting** with APT detection and kill chain analysis  
- **Behavioral analytics** with user profiling and risk scoring
- **Machine learning integration** for anomaly detection
- **Compliance automation** for SOX, PCI-DSS, GDPR, HIPAA monitoring
- **Temporal analysis** for pattern detection and seasonality

## Package Components

This package is organized into several key modules, each serving specific aspects of the GenAI processing pipeline:

### 📁 audit.go
**Core audit query types with enhanced security monitoring**
- `StructuredQuery` - Main audit query structure with advanced security features
- `StringOrArray` - Flexible string/array type for query fields
- `TimeRange` - Custom time range specifications
- `BusinessHours` - Business hours filtering configuration
- Advanced configuration types (MultiSource, BehavioralAnalysis, ThreatIntelligence, etc.)

### 📁 query.go  
**Request/response pipeline types**
- `ProcessingRequest` - Input request for natural language processing
- `ProcessingResponse` - Output response with structured query results
- `InternalRequest` - Internal system request structure
- `ModelRequest` - LLM API request structure
- `RawResponse` - Raw LLM API response structure

### 📁 models.go
**LLM model management and configuration**
- `ModelInfo` - Model metadata and capabilities
- `ModelConfig` - Model configuration parameters
- `TokenUsage` - Token consumption tracking and cost estimation
- `Example` - Few-shot examples for prompt engineering
- `OllamaRequest/Response` - Ollama-specific API structures

### 📁 session.go
**Conversation context and session management**
- `ConversationContext` - Complete session state management
- `ConversationEntry` - Individual conversation interactions
- `ResolvedReference` - Reference resolution for multi-turn conversations

### 📁 provider.go
**LLM provider configuration**
- `ProviderConfig` - Provider-specific configuration structure

### 📁 context_keys.go
**Context management constants**
- `ContextKey` - Typed keys for context.Context storage
- Context key constants for user identification

## Core Types Overview

### Request/Response Pipeline

#### ProcessingRequest
The entry point for natural language query processing.
```go
type ProcessingRequest struct {
    Query     string `json:"query"`          // Natural language query
    SessionID string `json:"session_id"`     // Session identifier for context
    ModelType string `json:"model_type"`     // Optional model specification
}
```

#### ProcessingResponse
The output containing structured query results.
```go
type ProcessingResponse struct {
    StructuredQuery interface{} `json:"structured_query"` // Parsed query
    Confidence      float64     `json:"confidence"`       // Confidence score (0.0-1.0)
    ValidationInfo  interface{} `json:"validation_info"`  // Validation results
    Error          string      `json:"error,omitempty"`  // Error details
}
```

### Model Management

#### ModelInfo
Comprehensive model metadata and capabilities.
```go
type ModelInfo struct {
    Name               string                 `json:"name"`                 // Model identifier
    Provider           string                 `json:"provider"`             // Provider name
    Version            string                 `json:"version"`              // Model version
    ModelType          string                 `json:"model_type"`           // Model type
    ContextWindow      int                    `json:"context_window"`       // Max context tokens
    MaxOutputTokens    int                    `json:"max_output_tokens"`    // Max output tokens
    SupportedLanguages []string               `json:"supported_languages"`  // Supported languages
    PricingInfo        map[string]interface{} `json:"pricing_info"`         // Pricing details
}
```

#### TokenUsage
Token consumption tracking and cost estimation.
```go
type TokenUsage struct {
    PromptTokens     int           `json:"prompt_tokens"`      // Input tokens
    CompletionTokens int           `json:"completion_tokens"`  // Output tokens
    TotalTokens      int           `json:"total_tokens"`       // Total tokens
    EstimatedCost    float64       `json:"estimated_cost"`     // Cost estimate
    ProcessingTime   time.Duration `json:"processing_time"`    // Processing time
    TokensPerSecond  float64       `json:"tokens_per_second"`  // Processing rate
}
```

### Session Management

#### ConversationContext
Complete conversation state and context management.
```go
type ConversationContext struct {
    SessionID           string                        `json:"session_id"`           // Session ID
    UserID              string                        `json:"user_id"`              // User ID
    CreatedAt           time.Time                     `json:"created_at"`           // Creation time
    LastActivity        time.Time                     `json:"last_activity"`        // Last activity
    ConversationHistory []ConversationEntry           `json:"conversation_history"` // Full history
    ResolvedReferences  map[string]ResolvedReference  `json:"resolved_references"`  // Reference resolution
    ContextEnrichment   map[string]interface{}        `json:"context_enrichment"`   // Additional context
}
```

**Key Methods:**
- `NewConversationContext(sessionID, userID string)` - Create new context
- `AddConversationEntry(query, response, refs)` - Add interaction
- `UpdateResolvedReference(key, type, value, confidence)` - Update references

#### ConversationEntry
Individual conversation interactions with metadata.
```go
type ConversationEntry struct {
    Timestamp          time.Time               `json:"timestamp"`           // Interaction time
    Query              string                  `json:"query"`               // User query
    Response           *StructuredQuery        `json:"response"`            // System response
    ResolvedReferences map[string]string       `json:"resolved_references"` // Resolved refs
}
```

### Utility Types

#### StringOrArray
Flexible type supporting both single strings and string arrays.
```go
type StringOrArray struct {
    value interface{} // Internal value storage
}
```

**Key Methods:**
- `IsString()` / `IsArray()` - Type checking
- `GetString()` / `GetArray()` - Value extraction
- `IsEmpty()` - Empty state checking
- JSON marshaling/unmarshaling support

#### ProviderConfig
LLM provider configuration structure.
```go
type ProviderConfig struct {
    APIKey     string                 `json:"api_key"`     // API key
    Endpoint   string                 `json:"endpoint"`    // API endpoint
    ModelName  string                 `json:"model_name"`  // Model name
    Parameters map[string]interface{} `json:"parameters"`  // Provider parameters
}
```

## Enhanced Audit Query Schema

### StructuredQuery

The main query structure that represents complete audit queries with all advanced security monitoring capabilities.

#### Basic Fields
```go
type StructuredQuery struct {
    // Core required fields
    LogSource string `json:"log_source"` // Enhanced to include "node-auditd"
    
    // Basic filtering
    Verb      StringOrArray `json:"verb,omitempty"`
    Resource  StringOrArray `json:"resource,omitempty"`
    Namespace StringOrArray `json:"namespace,omitempty"`
    User      StringOrArray `json:"user,omitempty"`
    Timeframe string        `json:"timeframe,omitempty"`
    Limit     int           `json:"limit,omitempty"`
    
    // Advanced filtering
    CorrelationFields []string `json:"correlation_fields,omitempty"`
    // ... other basic fields
}
```

#### Advanced Security Monitoring Fields
```go
// Multi-source correlation
MultiSource *MultiSourceConfig `json:"multi_source,omitempty"`

// Enhanced analysis with threat hunting capabilities  
Analysis *AdvancedAnalysisConfig `json:"analysis,omitempty"`

// Behavioral analytics and user profiling
BehavioralAnalysis *BehavioralAnalysisConfig `json:"behavioral_analysis,omitempty"`

// Threat intelligence correlation
ThreatIntelligence *ThreatIntelligenceConfig `json:"threat_intelligence,omitempty"`

// Machine learning analysis
MachineLearning *MachineLearningConfig `json:"machine_learning,omitempty"`

// Security detection criteria
DetectionCriteria *DetectionCriteriaConfig `json:"detection_criteria,omitempty"`

// OpenShift security context analysis
SecurityContext *SecurityContextConfig `json:"security_context,omitempty"`

// Compliance framework monitoring
ComplianceFramework *ComplianceFrameworkConfig `json:"compliance_framework,omitempty"`

// Time-based pattern analysis
TemporalAnalysis *TemporalAnalysisConfig `json:"temporal_analysis,omitempty"`
```

### Advanced Configuration Types

#### MultiSourceConfig
Enables correlation of events across different OpenShift audit log sources.

```go
type MultiSourceConfig struct {
    PrimarySource     string   `json:"primary_source"`      // Main log source
    SecondarySources  []string `json:"secondary_sources"`   // Additional sources
    CorrelationWindow string   `json:"correlation_window"`  // Time window
    CorrelationFields []string `json:"correlation_fields"`  // Fields to correlate
    JoinType          string   `json:"join_type"`           // inner, left, right, full
}
```

**Example**: Correlating kube-apiserver resource access with oauth-server authentication events
```json
{
  "multi_source": {
    "primary_source": "kube-apiserver",
    "secondary_sources": ["oauth-server"],
    "correlation_window": "30_minutes",
    "correlation_fields": ["user", "source_ip"]
  }
}
```

#### AdvancedAnalysisConfig
Enhanced analysis configuration for complex security investigations and threat hunting.

```go
type AdvancedAnalysisConfig struct {
    Type                  string                    `json:"type"`                    // Analysis type
    KillChainPhase        string                    `json:"kill_chain_phase"`        // MITRE ATT&CK phase
    MultiStageCorrelation bool                      `json:"multi_stage_correlation"` // Multi-stage attacks
    StatisticalAnalysis   *StatisticalAnalysisConfig `json:"statistical_analysis"`   // Statistical parameters
    // ... other fields
}
```

**Supported Analysis Types**:
- Basic: `multi_namespace_access`, `excessive_reads`, `privilege_escalation`, `anomaly_detection`
- Advanced: `apt_reconnaissance_detection`, `lateral_movement_detection`, `data_exfiltration_detection`
- Threat Hunting: `persistence_mechanism_detection`, `defense_evasion_detection`, `credential_harvesting_detection`

**Example**: APT reconnaissance detection
```json
{
  "analysis": {
    "type": "apt_reconnaissance_detection",
    "kill_chain_phase": "reconnaissance",
    "multi_stage_correlation": true,
    "statistical_analysis": {
      "pattern_deviation_threshold": 2.5,
      "confidence_interval": 0.95
    }
  }
}
```

#### BehavioralAnalysisConfig
User and system behavior analytics for insider threat detection and anomaly identification.

```go
type BehavioralAnalysisConfig struct {
    UserProfiling      bool                    `json:"user_profiling"`       // Enable user profiling
    BaselineComparison bool                    `json:"baseline_comparison"`  // Compare to baseline
    RiskScoring        *RiskScoringConfig      `json:"risk_scoring"`         // Risk calculation
    AnomalyDetection   *AnomalyDetectionConfig `json:"anomaly_detection"`    // Anomaly algorithms
    BaselineWindow     string                  `json:"baseline_window"`      // Baseline time window
}
```

**Example**: User behavior anomaly detection
```json
{
  "behavioral_analysis": {
    "user_profiling": true,
    "risk_scoring": {
      "enabled": true,
      "algorithm": "weighted_sum",
      "risk_factors": ["privilege_level", "resource_sensitivity", "timing_anomaly"]
    },
    "anomaly_detection": {
      "algorithm": "isolation_forest",
      "sensitivity": 0.8
    }
  }
}
```

#### ThreatIntelligenceConfig
Threat intelligence correlation with IOC matching and threat actor attribution.

```go
type ThreatIntelligenceConfig struct {
    IOCCorrelation         bool     `json:"ioc_correlation"`          // IOC correlation
    AttackPatternMatching  bool     `json:"attack_pattern_matching"`  // MITRE ATT&CK patterns
    ThreatActorAttribution bool     `json:"threat_actor_attribution"` // Threat actor analysis
    FeedSources           []string  `json:"feed_sources"`             // Threat feeds
    ConfidenceThreshold   float64   `json:"confidence_threshold"`     // Confidence threshold
}
```

#### MachineLearningConfig
Machine learning analysis for advanced anomaly detection and predictive modeling.

```go
type MachineLearningConfig struct {
    ModelType           string                    `json:"model_type"`            // ML model type
    FeatureEngineering  *FeatureEngineeringConfig `json:"feature_engineering"`   // Feature config
    TrainingWindow      string                    `json:"training_window"`       // Training period
    PredictionThreshold float64                   `json:"prediction_threshold"`  // Prediction threshold
}
```

#### DetectionCriteriaConfig
Specific detection criteria for various security analysis patterns.

```go
type DetectionCriteriaConfig struct {
    RapidOperations               *RapidOperationsConfig `json:"rapid_operations"`                // Rapid ops detection
    PrivilegeEscalationIndicators bool                   `json:"privilege_escalation_indicators"` // Privilege escalation
    LateralMovementPatterns       bool                   `json:"lateral_movement_patterns"`       // Lateral movement
    ReconnaissanceIndicators      bool                   `json:"reconnaissance_indicators"`       // Reconnaissance
    // ... other detection flags
}
```

#### SecurityContextConfig
OpenShift-specific security monitoring including SCC violations and pod security standards.

```go
type SecurityContextConfig struct {
    SCCViolations        bool     `json:"scc_violations"`         // SCC violations
    PodSecurityStandards string   `json:"pod_security_standards"` // Pod security level
    PrivilegeAnalysis    bool     `json:"privilege_analysis"`     // Privilege analysis
    CapabilityMonitoring []string `json:"capability_monitoring"`  // Linux capabilities
}
```

#### ComplianceFrameworkConfig
Automated compliance monitoring for regulatory frameworks.

```go
type ComplianceFrameworkConfig struct {
    Standards           []string                   `json:"standards"`            // Compliance standards
    Controls            []string                   `json:"controls"`             // Control requirements
    Reporting           *ComplianceReportingConfig `json:"reporting"`            // Report configuration
    AuditTrail          bool                       `json:"audit_trail"`          // Audit trail generation
    EvidenceCollection  bool                       `json:"evidence_collection"`  // Evidence collection
}
```

**Supported Standards**: SOX, PCI-DSS, GDPR, HIPAA, ISO27001, NIST, CIS, FedRAMP

#### TemporalAnalysisConfig
Advanced time-based pattern analysis for seasonal trends and temporal anomalies.

```go
type TemporalAnalysisConfig struct {
    PatternType          string  `json:"pattern_type"`           // Pattern type
    IntervalDetection    bool    `json:"interval_detection"`     // Interval detection
    AnomalyThreshold     float64 `json:"anomaly_threshold"`      // Anomaly threshold
    SeasonalityDetection bool    `json:"seasonality_detection"`  // Seasonal patterns
    TrendAnalysis        bool    `json:"trend_analysis"`         // Trend analysis
}
```

## Validation Rules

All configuration objects include comprehensive validation rules:

### Field Constraints
- **LogSource**: Must be one of `kube-apiserver`, `openshift-apiserver`, `oauth-server`, `oauth-apiserver`, `node-auditd`
- **Thresholds**: Numeric ranges enforced (e.g., confidence interval: 0.5-0.99)
- **Timeframes**: Predefined values (e.g., `1_hour_ago`, `24_hours_ago`, `7_days_ago`)
- **Algorithms**: Specific algorithm names validated

### Cross-Field Dependencies
- `analysis.statistical_analysis` requires `analysis.type`
- `time_range` and `timeframe` are mutually exclusive
- `multi_source` requires at least one secondary source

### Performance Constraints
- Maximum limit: 1000 results
- Maximum correlation fields: 10
- Pattern length limits for regex validation

## Usage Examples

### Basic Query (Backward Compatible)
```json
{
  "log_source": "kube-apiserver",
  "verb": "delete",
  "resource": "secrets",
  "timeframe": "yesterday",
  "exclude_users": ["system:", "kube-"],
  "limit": 20
}
```

### Intermediate Query with Business Hours Analysis
```json
{
  "log_source": "kube-apiserver",
  "verb": ["create", "update", "patch"],
  "resource": ["roles", "rolebindings"],
  "user_pattern": "^(?!system:).*",
  "business_hours": {
    "outside_only": true,
    "start_hour": 9,
    "end_hour": 17
  },
  "group_by": ["user", "namespace"],
  "limit": 50
}
```

### Advanced Threat Hunting Query
```json
{
  "log_source": "kube-apiserver",
  "analysis": {
    "type": "apt_reconnaissance_detection",
    "kill_chain_phase": "reconnaissance",
    "multi_stage_correlation": true,
    "statistical_analysis": {
      "pattern_deviation_threshold": 2.5,
      "confidence_interval": 0.95
    }
  },
  "multi_source": {
    "primary_source": "kube-apiserver",
    "secondary_sources": ["oauth-server", "node-auditd"],
    "correlation_window": "30_minutes",
    "correlation_fields": ["user", "source_ip"]
  },
  "behavioral_analysis": {
    "anomaly_detection": {
      "algorithm": "isolation_forest",
      "sensitivity": 0.8
    }
  },
  "detection_criteria": {
    "reconnaissance_indicators": true,
    "unusual_api_patterns": true
  },
  "limit": 100
}
```

### Compliance Monitoring Query
```json
{
  "log_source": "kube-apiserver",
  "compliance_framework": {
    "standards": ["SOX", "PCI-DSS"],
    "controls": ["access_logging", "data_protection"],
    "reporting": {
      "format": "detailed",
      "include_evidence": true
    },
    "audit_trail": true
  },
  "limit": 50
}
```

## Backward Compatibility

The enhanced schema maintains full backward compatibility:
- All existing basic and intermediate queries work unchanged
- Advanced fields are optional (`omitempty` tags)
- Existing validation rules preserved
- JSON serialization/deserialization maintains compatibility

## Testing

### Test Coverage
Comprehensive test coverage includes:
- **JSON Serialization**: Marshaling/unmarshaling for all struct types
- **Complex Validation**: Nested object validation and cross-field dependencies
- **Backward Compatibility**: Verification that existing queries work unchanged  
- **Edge Cases**: Boundary conditions, Unicode handling, large datasets
- **Performance**: Large schema objects and high-volume operations
- **Security**: Validation rules and constraint enforcement

### Test Commands

#### Run All Tests
```bash
# Run all tests in the types package
go test ./pkg/types/... -v

# Run tests with coverage report
go test ./pkg/types/... -v -cover

# Run tests with detailed coverage
go test ./pkg/types/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

#### Run Specific Test Categories
```bash
# Test only StringOrArray functionality
go test ./pkg/types/... -run="TestStringOrArray" -v

# Test enhanced StructuredQuery features
go test ./pkg/types/... -run="TestStructuredQuery|TestMultiSource|TestAdvanced|TestBehavioral" -v

# Test model management types
go test ./pkg/types/... -run="TestModel|TestToken" -v

# Test session and context management
go test ./pkg/types/... -run="TestConversation|TestSession" -v

# Test backward compatibility
go test ./pkg/types/... -run="TestBackward" -v
```

#### Performance and Benchmark Tests
```bash
# Run benchmark tests
go test ./pkg/types/... -bench=. -v

# Run specific benchmarks
go test ./pkg/types/... -bench="BenchmarkStringOrArray" -v

# Memory profiling
go test ./pkg/types/... -bench=. -memprofile=mem.prof
go tool pprof mem.prof
```

#### Test Validation and Build
```bash
# Validate package builds correctly
go build ./pkg/types/...

# Run tests with race detection
go test ./pkg/types/... -race -v

# Validate with strict checks
go vet ./pkg/types/...

# Check for common issues
golint ./pkg/types/...
```

### Test Data Examples

#### Basic Query Test
```go
query := &StructuredQuery{
    LogSource: "kube-apiserver",
    Verb:      StringOrArray{value: "delete"},
    Resource:  StringOrArray{value: "secrets"},
    Limit:     20,
}
```

#### Advanced Query Test
```go
query := &StructuredQuery{
    LogSource: "kube-apiserver",
    Analysis: &AdvancedAnalysisConfig{
        Type: "apt_reconnaissance_detection",
        KillChainPhase: "reconnaissance",
    },
    MultiSource: &MultiSourceConfig{
        PrimarySource: "kube-apiserver",
        SecondarySources: []string{"oauth-server"},
    },
}
```

### Test Organization
- `audit_test.go` - StructuredQuery and audit-related types
- `models_test.go` - Model management and configuration types  
- `query_test.go` - Request/response pipeline types
- `session_test.go` - Session and context management types

### Continuous Integration
For CI/CD pipelines:
```bash
# Standard CI test command
go test ./pkg/types/... -v -race -cover

# With timeout for long-running tests
go test ./pkg/types/... -v -timeout=5m

# JSON output for CI processing
go test ./pkg/types/... -json > test-results.json
```

## Migration from Basic Schema

To migrate existing queries to use advanced features:

1. **Add Multi-source Correlation**:
   ```json
   {
     // existing fields...
     "multi_source": {
       "primary_source": "kube-apiserver",
       "secondary_sources": ["oauth-server"],
       "correlation_fields": ["user"]
     }
   }
   ```

2. **Enable Behavioral Analytics**:
   ```json
   {
     // existing fields...
     "behavioral_analysis": {
       "user_profiling": true,
       "anomaly_detection": {
         "algorithm": "isolation_forest"
       }
     }
   }
   ```

3. **Add Compliance Monitoring**:
   ```json
   {
     // existing fields...
     "compliance_framework": {
       "standards": ["SOX", "PCI-DSS"],
       "audit_trail": true
     }
   }
   ```

This enhanced schema enables the system to support enterprise-grade security monitoring, advanced threat hunting, and automated compliance reporting while maintaining full backward compatibility with existing implementations.

## Summary

The `pkg/types` package serves as the foundation for the GenAI-Powered OpenShift Audit Query System, providing:

### ✅ **Complete Type Coverage**
- **Request/Response Pipeline**: Complete API types for natural language processing
- **Model Management**: LLM provider configuration, token tracking, performance metrics
- **Session Management**: Multi-turn conversations with context and reference resolution
- **Enhanced Audit Schema**: Basic to enterprise-grade security monitoring capabilities

### ✅ **Advanced Security Features**
- **Multi-source correlation** across OpenShift audit log sources
- **Threat hunting** with APT detection and MITRE ATT&CK integration
- **Behavioral analytics** with user profiling and anomaly detection
- **Machine learning** integration for predictive security modeling
- **Compliance automation** for SOX, PCI-DSS, GDPR, HIPAA monitoring

### ✅ **Quality Assurance**
- **Comprehensive Testing**: 65.4% test coverage with extensive test suites
- **Backward Compatibility**: 100% compatibility with existing basic queries
- **Performance Optimized**: Efficient JSON serialization and validation
- **Well Documented**: Complete API documentation with examples

### ✅ **Development Ready**
- **Independent Testing**: Comprehensive test commands for isolated development
- **CI/CD Integration**: Pipeline-ready test commands with JSON output
- **Build Validation**: Package builds and validates correctly
- **Type Safety**: Strong typing with comprehensive validation rules

This package enables the transformation from basic audit querying to enterprise-grade security monitoring while maintaining simplicity for basic use cases and providing powerful advanced capabilities for complex security analysis.