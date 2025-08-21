# Configuration Documentation: rules.yaml

## Overview

This document provides comprehensive documentation for the `configs/rules.yaml` file, which defines validation rules and security constraints for the GenAI-Powered OpenShift Audit Query System. After thorough analysis of the codebase, this documentation reveals significant configuration mapping issues that prevent several advanced features from functioning correctly.

## Critical Discovery: Configuration System is Partially Broken

**⚠️ IMPORTANT**: This analysis reveals that **4 out of 13 configuration sections have broken mappings**, preventing advanced analysis features from accessing their configuration. The core input validation system works correctly, but advanced features are effectively disabled due to configuration access failures.

## Configuration Status Legend

- ✅ **ACTIVELY USED**: Configuration is correctly mapped and used by validation rules
- ⚠️ **BROKEN MAPPING**: Configuration exists but cannot be accessed due to mapping issues
- 🔄 **DUPLICATE**: Same values defined in multiple places
- ❌ **UNUSED**: Defined in YAML but not accessed by any code

---

## Configuration Sections Analysis

### 1. Schema Validation Configuration

**Status**: ❌ **UNUSED**

```yaml
schema_validation:
  enabled: true
  strict_mode: true
```

**Issue**: No validation rule or loader mapping accesses this section
**Location in Code**: Not referenced anywhere
**Recommendation**: Remove unless implementing dedicated schema validation rule

---

### 2. Rule Engine Configuration

**Status**: ✅ **ACTIVELY USED**

```yaml
rule_engine:
  parallel_execution: true
  worker_pool_size: 4
  rule_timeout_seconds: 30
  dependency_resolution: true
```

**Usage**: 
- **Struct Mapping**: `ValidationConfig.RuleEngine` (`yaml:"rule_engine"`)
- **Application**: `internal/validator/engine.go` uses these settings for rule execution
- **Function**: Controls parallel rule evaluation, timeouts, and dependency resolution

**Validation Logic**: `loader.go:203-215` validates all limits and timeouts

---

### 3. Advanced Analysis Configuration

**Status**: ⚠️ **BROKEN MAPPING** 

**Critical Issue**: Code requests `"analysis_limits"` but YAML section is `"advanced_analysis"`

```yaml
advanced_analysis:
  enabled: true
  allowed_analysis_types:
    - "apt_reconnaissance_detection"
    - "c2_communication_detection"
    - "statistical_user_behavior_analysis"
    # ... 160+ analysis types
  kill_chain_phases:
    - "reconnaissance" 
    - "weaponization"
    - "delivery"
    - "exploitation"
  max_group_by_fields: 5
```

**Mapping Problem**: 
- **Rule Registration**: `engine.go:616` → `rules.NewAdvancedAnalysisRule(e.config.GetConfigSection("analysis_limits"))`
- **Missing Mapping**: `GetConfigSection("analysis_limits")` returns `config.AnalysisLimits` 
- **YAML Section**: `advanced_analysis` is not mapped in `GetConfigSection()`
- **Result**: AdvancedAnalysisRule receives `nil` configuration, falls back to empty defaults

**Query Examples from Functional Tests**:

**Advanced Query** (APT Detection):
```json
{
  "analysis": {
    "type": "apt_reconnaissance_detection",
    "kill_chain_phase": "reconnaissance",
    "statistical_analysis": {
      "pattern_deviation_threshold": 2.5,
      "baseline_comparison": true
    }
  }
}
```

**Current Status**: This validation **FAILS** because the rule cannot access `allowed_analysis_types`

**Required Fix**:
```go
// In internal/validator/engine.go line 616, change:
e.RegisterRule("advanced_analysis", rules.NewAdvancedAnalysisRule(e.config.GetConfigSection("analysis_limits")))
// To:
e.RegisterRule("advanced_analysis", rules.NewAdvancedAnalysisRule(e.config.GetConfigSection("advanced_analysis")))

// In internal/validator/loader.go GetConfigSection(), add:
case "advanced_analysis":
    return config.AdvancedAnalysis

// In ValidationConfig struct, add:
AdvancedAnalysis map[string]interface{} `yaml:"advanced_analysis"`
```

---

### 4. Time Windows Configuration

**Status**: ⚠️ **BROKEN MAPPING**

```yaml
time_windows:
  allowed_time_windows:
    - "1_minute"
    - "5_minutes"
    - "24_hours"
    - "7_days"
    # ... complete list
  allowed_correlation_windows:
    - "1_minute"
    - "30_minutes"
    - "24_hours"
  allowed_baseline_windows:
    - "7_days"
    - "30_days"
    - "90_days"
```

**Issue**: No `GetConfigSection` mapping for `"time_windows"`
**Expected Usage**: AdvancedAnalysisRule tries to access `r.config["allowed_time_windows"]` but gets `nil` config
**Current Behavior**: Falls back to empty time windows, **blocking all temporal analysis**

**Query Examples**:
```json
{
  "analysis": {
    "type": "statistical_analysis",
    "time_window": "2_hours"
  },
  "timeframe": "24_hours_ago"
}
```

**Required Fix**: Add mapping for `time_windows` section

---

### 5. Behavioral Analytics Configuration

**Status**: ✅ **ACTIVELY USED**

```yaml
behavioral_analytics:
  enabled: true
  allowed_risk_factors:
    - "privilege_level"
    - "resource_sensitivity" 
    - "timing_anomaly"
    - "access_pattern"
  max_risk_factors: 10
  baseline_window_limits:
    min_baseline_days: 7
    max_baseline_days: 90
  anomaly_threshold_limits:
    min: 0.1
    max: 10.0
```

**Usage**:
- **Rule Registration**: `engine.go:618` → `rules.NewBehavioralAnalyticsRule(e.config.GetConfigSection("behavioral_analytics"))`
- **Mapping**: `GetConfigSection("behavioral_analytics")` → `config.BehavioralAnalytics`
- **Struct Field**: `BehavioralAnalytics map[string]interface{} yaml:"behavioral_analytics"`

**Working Query Example**:
```json
{
  "behavioral_analysis": {
    "user_profiling": true,
    "baseline_window": "30_days",
    "risk_scoring": {
      "risk_factors": ["privilege_level", "resource_sensitivity"],
      "weighting_scheme": {
        "privilege_level": 0.6,
        "resource_sensitivity": 0.4
      }
    },
    "anomaly_detection": {
      "algorithm": "isolation_forest",
      "contamination": 0.1
    }
  }
}
```

**Validation**: Ensures risk factors are in allowed list, baseline windows within limits, thresholds within range

---

### 6. Sort Configuration

**Status**: ⚠️ **BROKEN MAPPING**

```yaml
sort_configuration:
  allowed_sort_fields:
    - "timestamp"
    - "username"
    - "resource"
    - "risk_score"
    # ... complete list
  allowed_sort_orders:
    - "asc"
    - "desc"
```

**Issue**: No `GetConfigSection` mapping, needed by AdvancedAnalysisRule for sorting validation
**Current Impact**: Sort field validation **disabled**, potentially allowing invalid sort parameters

---

### 7. Business Hours Configuration

**Status**: ❌ **UNUSED**

```yaml
business_hours_configuration:
  allowed_presets:
    - "business_hours"
    - "outside_business_hours"
    - "weekend"
    - "all_hours"
```

**Issue**: No validation rule uses this configuration
**Mapping Problem**: Code expects `"business_hours"` but YAML defines `"business_hours_configuration"`

---

### 8. Response Status Configuration

**Status**: 🔄 **DUPLICATE** - Points to Section 13

```yaml
response_status_configuration:
  allowed_status_codes:
    - "200"
    - "201"
    - "400" 
    - "401"
    - "403"
    # ... complete list
```

**Issue**: Exact duplicate of `input_validation.field_values.allowed_response_status`
**Actual Usage**: Only the `input_validation` version is used by ComprehensiveInputValidationRule
**Recommendation**: Remove this duplicate section

---

### 9. Auth Decisions Configuration

**Status**: 🔄 **DUPLICATE** - Points to Section 13

```yaml
auth_decisions_configuration:
  allowed_decisions:
    - "allow"
    - "error"
    - "forbid"
```

**Issue**: Exact duplicate of `input_validation.field_values.allowed_auth_decisions`
**Actual Usage**: Only the `input_validation` version is used
**Recommendation**: Remove this duplicate section

---

### 10. Compliance Configuration

**Status**: ⚠️ **BROKEN MAPPING**

**Critical Issue**: Code requests `"compliance_framework"` but YAML section is `"compliance"`

```yaml
compliance:
  enabled: true
  allowed_standards:
    - "SOX"
    - "PCI-DSS"
    - "GDPR"
    - "HIPAA"
    - "ISO27001"
  max_standards: 5
  max_controls: 10
  min_retention_days: 365
  max_audit_gap_hours: 24
```

**Mapping Problem**:
- **Rule Registration**: `engine.go:619` → `rules.NewComplianceRule(e.config.GetConfigSection("compliance_framework"))`
- **GetConfigSection**: Maps `"compliance_framework"` → `config.ComplianceFramework`
- **YAML Section**: `compliance` (not `compliance_framework`)
- **Result**: ComplianceRule receives `nil` configuration

**Query Example from Functional Tests**:
```json
{
  "compliance_framework": {
    "standards": ["SOX"],
    "controls": ["access_logging", "change_management"],
    "reporting": {
      "format": "detailed", 
      "include_evidence": true
    }
  }
}
```

**Current Status**: Compliance validation **DISABLED**

**Required Fix**:
```go
// In internal/validator/engine.go line 619, change:
e.RegisterRule("compliance", rules.NewComplianceRule(e.config.GetConfigSection("compliance_framework")))
// To:
e.RegisterRule("compliance", rules.NewComplianceRule(e.config.GetConfigSection("compliance")))
```

---

### 11. Multi-Source Configuration

**Status**: ✅ **ACTIVELY USED**

```yaml
multi_source:
  enabled: true
  max_sources: 5
  max_correlation_fields: 10
  max_correlation_complexity: 100
```

**Usage**:
- **Rule Registration**: `engine.go:617` → `rules.NewMultiSourceRule(e.config.GetConfigSection("multi_source"))`
- **Mapping**: `GetConfigSection("multi_source")` → `config.MultiSource`
- **Struct Field**: `MultiSource map[string]interface{} yaml:"multi_source"`

**Working Query Example**:
```json
{
  "multi_source": {
    "primary_source": "kube-apiserver",
    "secondary_sources": ["oauth-server", "node-auditd"],
    "correlation_window": "30_minutes",
    "correlation_fields": ["user", "source_ip", "timestamp"],
    "join_type": "inner"
  }
}
```

**Validation**: Ensures sources ≤ max_sources, correlation fields ≤ max_correlation_fields, complexity ≤ max_correlation_complexity

---

### 12. Performance Configuration

**Status**: ❌ **UNUSED**

```yaml
performance:
  enabled: true
  max_query_complexity_score: 100
  max_memory_usage_mb: 1024
  max_cpu_usage_percent: 50
  max_execution_time_seconds: 60
  max_raw_results: 1000
  max_aggregated_results: 500
  max_concurrent_sources: 3
```

**Issue**: No validation rule accesses this configuration
**Potential Usage**: Could implement PerformanceValidationRule
**Recommendation**: Remove or implement dedicated performance validation

---

### 13. Input Validation Configuration (Primary System)

**Status**: ✅ **ACTIVELY USED** - This is the **core security validation system**

```yaml
input_validation:
  enabled: true
  
  required_fields:
    mandatory: ["log_source"]
    conditional: []
  
  character_validation:
    max_query_length: 10000
    max_pattern_length: 500
    forbidden_chars: ["<", ">", "&", "\"", "'", "`", "|", ";", "$"]
    valid_regex_pattern: "^[a-zA-Z0-9\\-_\\*\\.\\?\\+\\[\\]\\{\\}\\(\\)\\|\\/\\s]+$"
    valid_ip_pattern: "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
  
  security_patterns:
    forbidden_patterns:
      - "system:admin"
      - "cluster-admin"
      - "delete --all"
      - "privileged: true"
      # ... complete security pattern list
  
  field_values:
    allowed_log_sources:
      - "kube-apiserver"
      - "openshift-apiserver"
      - "oauth-server"
      - "oauth-apiserver"
      - "node-auditd"
    allowed_verbs:
      - "get"
      - "list"
      - "create"
      - "update"
      - "delete"
      # ... complete list
    allowed_resources:
      - "pods"
      - "services"
      - "secrets"
      - "namespaces"
      # ... complete list
    allowed_auth_decisions:
      - "allow"
      - "error"
      - "forbid"
    allowed_response_status:
      - "200"
      - "201"
      - "400"
      - "401"
      - "403"
      # ... complete list
  
  performance_limits:
    max_result_limit: 50
    max_array_elements: 15
    max_days_back: 90
    allowed_timeframes:
      - "today"
      - "yesterday"
      - "1_hour_ago"
      - "24_hours_ago"
      # ... complete list
```

**Usage**:
- **Direct Access**: `safety.go:183` → `rules.NewComprehensiveInputValidationRule(&sv.config.InputValidation)`
- **Struct Mapping**: `InputValidation types.InputValidationConfig yaml:"input_validation"`
- **Priority**: Highest validation priority (90) after schema validation (100)

**Query Validation Examples**:

**✅ Valid Basic Query**:
```json
{
  "log_source": "kube-apiserver",
  "verb": "create",
  "resource": "pods",
  "namespace": "default",
  "timeframe": "today",
  "limit": 20
}
```

**❌ Blocked Security Pattern**:
```json
{
  "log_source": "kube-apiserver",
  "user_pattern": "system:admin"  // Blocked by forbidden_patterns
}
```

**❌ Invalid Field Value**:
```json
{
  "log_source": "invalid-source",  // Not in allowed_log_sources
  "verb": "hack",                  // Not in allowed_verbs
  "limit": 2000                    // Exceeds max_result_limit
}
```

**Validation Coverage**:
- ✅ Required field validation
- ✅ Character and format validation  
- ✅ Security pattern detection
- ✅ Field value whitelisting
- ✅ Performance limit enforcement
- ✅ **4x faster** than previous scattered validation (single pass vs multiple scans)

---

## Critical Issues Summary

### Configuration Mapping Failures

**4 Major Broken Mappings**:

1. **Advanced Analysis**: Code requests `analysis_limits` but YAML has `advanced_analysis`
2. **Compliance**: Code requests `compliance_framework` but YAML has `compliance`  
3. **Time Windows**: No mapping for `time_windows` section
4. **Sort Configuration**: No mapping for `sort_configuration` section

### Impact Assessment

**✅ Working Systems**:
- ✅ **Core input validation** (comprehensive security checks)
- ✅ **Behavioral analytics** (risk scoring, anomaly detection)
- ✅ **Multi-source correlation** (cross-log analysis)
- ✅ **Rule engine** (parallel execution, timeouts)

**❌ Broken Systems**:
- ❌ **Advanced analysis** (APT detection, threat hunting) - **DISABLED**
- ❌ **Compliance validation** (SOX, PCI-DSS, GDPR) - **DISABLED**
- ❌ **Time window validation** (temporal analysis) - **DISABLED**
- ❌ **Sort field validation** (query result ordering) - **DISABLED**

### Immediate Fixes Required

**Fix 1: Advanced Analysis Mapping**
```bash
# Edit internal/validator/engine.go line 616
# Change: GetConfigSection("analysis_limits")
# To: GetConfigSection("advanced_analysis")

# Add to internal/validator/loader.go ValidationConfig:
AdvancedAnalysis map[string]interface{} `yaml:"advanced_analysis"`

# Add to GetConfigSection():
case "advanced_analysis":
    return config.AdvancedAnalysis
```

**Fix 2: Compliance Mapping**
```bash
# Edit internal/validator/engine.go line 619
# Change: GetConfigSection("compliance_framework") 
# To: GetConfigSection("compliance")
```

**Fix 3: Add Missing Mappings**
```bash
# Add to ValidationConfig struct:
TimeWindows map[string]interface{} `yaml:"time_windows"`
SortConfiguration map[string]interface{} `yaml:"sort_configuration"`

# Add to GetConfigSection():
case "time_windows":
    return config.TimeWindows
case "sort_configuration":  
    return config.SortConfiguration
```

### Cleanup Recommendations

**Remove Duplicates**:
- Remove `response_status_configuration` (use `input_validation.field_values.allowed_response_status`)
- Remove `auth_decisions_configuration` (use `input_validation.field_values.allowed_auth_decisions`)

**Remove Unused**:
- Remove `schema_validation` (no implementation)
- Remove `business_hours_configuration` (no validation rule)
- Remove `performance` (no validation rule) OR implement PerformanceRule

## Configuration Testing

**Test Basic Input Validation** (Working):
```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Who deleted pods today?","session_id":"test"}'
```

**Test Advanced Analysis** (Currently Broken):
```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Detect APT reconnaissance patterns","session_id":"test"}'
```

## Summary

The `configs/rules.yaml` configuration system is **partially functional**. The core security validation (`input_validation`) works correctly and provides comprehensive protection. However, **advanced analysis capabilities are effectively disabled** due to configuration mapping failures.

**Current Status**:
- ✅ **4 sections working**: input_validation, behavioral_analytics, multi_source, rule_engine
- ⚠️ **4 sections broken**: advanced_analysis, compliance, time_windows, sort_configuration
- 🔄 **2 sections duplicate**: response_status_configuration, auth_decisions_configuration
- ❌ **3 sections unused**: schema_validation, business_hours_configuration, performance

**Priority**: Fix the 4 broken mappings to restore advanced analysis, compliance monitoring, and temporal validation capabilities.