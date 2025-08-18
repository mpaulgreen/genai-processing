# Schema Validation Rules for OpenShift Audit Query System

## Overview

This document defines comprehensive validation rules for the enhanced JSON schema, ensuring data integrity, security, and proper query execution across all complexity levels from basic filtering to advanced threat hunting.

## Validation Architecture

### Validation Layers
1. **Syntax Validation**: JSON structure and type validation
2. **Semantic Validation**: Business logic and field relationship validation  
3. **Security Validation**: Security constraints and access control validation
4. **Performance Validation**: Query performance and resource usage validation

### Validation Severity Levels
- **ERROR**: Prevents query execution, must be fixed
- **WARNING**: Query can execute but may have issues
- **INFO**: Informational messages for optimization

## Core Field Validation Rules

### log_source (Required)
**Type**: string  
**Validation Rules**:
- ✅ **REQUIRED**: Must be present in all queries
- ✅ **ENUM**: Must be one of: `kube-apiserver`, `openshift-apiserver`, `oauth-server`, `oauth-apiserver`, `node-auditd`
- ✅ **NOT_EMPTY**: Cannot be empty string or whitespace

```json
// Valid
"log_source": "kube-apiserver"

// Invalid
"log_source": ""  // ERROR: log_source cannot be empty
"log_source": "invalid-source"  // ERROR: invalid log source
```

### limit
**Type**: integer  
**Validation Rules**:
- ✅ **RANGE**: 1 ≤ limit ≤ 1000
- ✅ **DEFAULT**: If not specified, defaults to 20
- ⚠️ **PERFORMANCE**: Values > 500 generate WARNING

```json
// Valid
"limit": 50

// Invalid  
"limit": 0     // ERROR: limit must be at least 1
"limit": 1500  // ERROR: limit cannot exceed 1000
"limit": 800   // WARNING: large limit may impact performance
```

## Basic Filtering Field Validation

### verb
**Type**: StringOrArray  
**Validation Rules**:
- ✅ **ENUM**: Must be valid HTTP verbs: `get`, `list`, `create`, `update`, `patch`, `delete`, `watch`
- ✅ **ARRAY_SIZE**: If array, maximum 10 elements
- ✅ **NO_DUPLICATES**: Array elements must be unique

```json
// Valid
"verb": "create"
"verb": ["create", "update", "delete"]

// Invalid
"verb": "invalid"  // ERROR: invalid verb
"verb": ["create", "create"]  // ERROR: duplicate verbs not allowed
"verb": ["get", "list", "create", "update", "patch", "delete", "watch", "connect", "proxy", "redirect", "bind"]  // ERROR: too many verbs
```

### resource
**Type**: StringOrArray  
**Validation Rules**:
- ✅ **KUBERNETES_RESOURCE**: Must be valid Kubernetes resource type
- ✅ **CASE_SENSITIVE**: Exact case matching required
- ✅ **ARRAY_SIZE**: If array, maximum 20 elements

```json
// Valid
"resource": "pods"
"resource": ["secrets", "configmaps", "serviceaccounts"]

// Invalid
"resource": "Pods"  // ERROR: case mismatch, should be "pods"
"resource": "invalid-resource"  // WARNING: unknown resource type
```

### namespace
**Type**: StringOrArray  
**Validation Rules**:
- ✅ **DNS_LABEL**: Must be valid DNS label format
- ✅ **LENGTH**: 1-63 characters
- ✅ **PATTERN**: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`

```json
// Valid
"namespace": "default"
"namespace": ["production", "staging-env"]

// Invalid
"namespace": "Invalid_Namespace"  // ERROR: invalid characters
"namespace": "a"  // Valid but WARNING: very short namespace name
```

### user
**Type**: StringOrArray  
**Validation Rules**:
- ✅ **EMAIL_FORMAT**: If contains '@', must be valid email format
- ✅ **SYSTEM_USER**: System users should start with "system:"
- ✅ **LENGTH**: 1-256 characters

```json
// Valid
"user": "john.doe@company.com"
"user": "system:serviceaccount:default:my-sa"

// Invalid
"user": "invalid@email"  // ERROR: invalid email format
"user": ""  // ERROR: user cannot be empty
```

### timeframe
**Type**: string  
**Validation Rules**:
- ✅ **ENUM**: Must be one of predefined values
- ✅ **MUTUALLY_EXCLUSIVE**: Cannot be used with `time_range`

**Valid Values**: `today`, `yesterday`, `1_hour_ago`, `6_hours_ago`, `12_hours_ago`, `24_hours_ago`, `7_days_ago`, `30_days_ago`, `last_week`, `last_month`

```json
// Valid
"timeframe": "24_hours_ago"

// Invalid
"timeframe": "2_hours_ago"  // ERROR: not a valid timeframe
// Cannot have both:
{
  "timeframe": "today",
  "time_range": { ... }  // ERROR: mutually exclusive
}
```

## Advanced Filtering Field Validation

### exclude_users
**Type**: array of strings  
**Validation Rules**:
- ✅ **ARRAY_SIZE**: Maximum 50 elements
- ✅ **NO_EMPTY**: No empty strings in array
- ✅ **PATTERN_VALIDATION**: Each element must be valid user pattern

```json
// Valid
"exclude_users": ["system:", "kube-", "openshift-"]

// Invalid
"exclude_users": [""]  // ERROR: empty string not allowed
"exclude_users": [...50+ elements...]  // ERROR: too many exclude patterns
```

### user_pattern, namespace_pattern, resource_name_pattern
**Type**: string (regex)  
**Validation Rules**:
- ✅ **REGEX_SYNTAX**: Must be valid regular expression
- ✅ **COMPLEXITY**: Regex complexity score < 100
- ✅ **SAFETY**: No catastrophic backtracking patterns

```json
// Valid
"user_pattern": "^admin@.*\\.company\\.com$"

// Invalid
"user_pattern": "[unclosed"  // ERROR: invalid regex syntax
"user_pattern": "(.+)+$"  // ERROR: catastrophic backtracking risk
```

## Response and Status Field Validation

### response_status
**Type**: StringOrArray  
**Validation Rules**:
- ✅ **HTTP_STATUS**: Must be valid HTTP status codes (100-599)
- ✅ **RANGE_SYNTAX**: Supports range syntax (>=400, <500)
- ✅ **LOGICAL_RANGE**: Range values must be logical

```json
// Valid
"response_status": "403"
"response_status": ["401", "403", "500"]
"response_status": ">=400"

// Invalid
"response_status": "999"  // ERROR: invalid HTTP status code
"response_status": ">=700"  // ERROR: status code out of range
```

### auth_decision
**Type**: string  
**Validation Rules**:
- ✅ **ENUM**: Must be one of: `allow`, `error`, `forbid`
- ✅ **LOG_SOURCE_COMPATIBILITY**: Must be compatible with log source

```json
// Valid
"auth_decision": "forbid"

// Invalid
"auth_decision": "denied"  // ERROR: invalid auth decision
// Invalid combination:
{
  "log_source": "node-auditd",
  "auth_decision": "forbid"  // ERROR: auth_decision not applicable to node-auditd
}
```

## Network and Source Field Validation

### source_ip
**Type**: StringOrArray  
**Validation Rules**:
- ✅ **IP_FORMAT**: Must be valid IPv4/IPv6 address or CIDR
- ✅ **PRIVATE_RANGES**: Validate against known private IP ranges
- ✅ **CIDR_NOTATION**: Validate CIDR block format

```json
// Valid
"source_ip": "192.168.1.100"
"source_ip": ["10.0.0.0/8", "172.16.0.0/12"]
"source_ip": "2001:db8::1"

// Invalid
"source_ip": "999.999.999.999"  // ERROR: invalid IP address
"source_ip": "10.0.0.0/40"  // ERROR: invalid CIDR notation
```

## Time-Based Analysis Validation

### time_range
**Type**: object  
**Validation Rules**:
- ✅ **REQUIRED_FIELDS**: Must have both `start` and `end`
- ✅ **TIMESTAMP_FORMAT**: Must be valid ISO 8601 format
- ✅ **LOGICAL_ORDER**: start < end
- ✅ **DURATION_LIMITS**: Maximum duration of 90 days
- ✅ **MUTUALLY_EXCLUSIVE**: Cannot be used with `timeframe`

```json
// Valid
"time_range": {
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-02T00:00:00Z"
}

// Invalid
"time_range": {
  "start": "2024-01-02T00:00:00Z",
  "end": "2024-01-01T00:00:00Z"  // ERROR: end before start
}
"time_range": {
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-04-01T00:00:00Z"  // ERROR: duration exceeds 90 days
}
```

### business_hours
**Type**: object  
**Validation Rules**:
- ✅ **HOUR_RANGE**: start_hour and end_hour must be 0-23
- ✅ **LOGICAL_HOURS**: start_hour < end_hour (unless spanning midnight)
- ✅ **TIMEZONE**: Must be valid timezone identifier

```json
// Valid
"business_hours": {
  "outside_only": true,
  "start_hour": 9,
  "end_hour": 17,
  "timezone": "EST"
}

// Invalid
"business_hours": {
  "start_hour": 25,  // ERROR: hour out of range
  "end_hour": 17
}
```

## Advanced Analysis Validation

### analysis
**Type**: object  
**Validation Rules**:
- ✅ **TYPE_REQUIRED**: `type` field is required
- ✅ **TYPE_ENUM**: Must be valid analysis type
- ✅ **FIELD_DEPENDENCIES**: Required fields based on analysis type
- ✅ **STATISTICAL_RANGES**: Statistical parameters within valid ranges

**Analysis Type Validation**:

```json
// Valid
"analysis": {
  "type": "apt_reconnaissance_detection",
  "kill_chain_phase": "reconnaissance",  // Required for APT analysis
  "statistical_analysis": {
    "pattern_deviation_threshold": 2.5,  // 0.1 ≤ threshold ≤ 10.0
    "confidence_interval": 0.95  // 0.5 ≤ confidence ≤ 0.99
  }
}

// Invalid
"analysis": {
  "type": "invalid_analysis"  // ERROR: invalid analysis type
}
"analysis": {
  "type": "apt_reconnaissance_detection"
  // ERROR: missing required kill_chain_phase
}
"analysis": {
  "type": "anomaly_detection",
  "statistical_analysis": {
    "confidence_interval": 1.5  // ERROR: confidence > 1.0
  }
}
```

### multi_source
**Type**: object  
**Validation Rules**:
- ✅ **SOURCE_VALIDATION**: All sources must be valid log sources
- ✅ **UNIQUE_SOURCES**: No duplicate sources
- ✅ **CORRELATION_FIELDS**: Must be valid correlatable fields
- ✅ **WINDOW_FORMAT**: correlation_window must be valid time duration

```json
// Valid
"multi_source": {
  "primary_source": "kube-apiserver",
  "secondary_sources": ["oauth-server", "node-auditd"],
  "correlation_window": "30_minutes",
  "correlation_fields": ["user", "source_ip"]
}

// Invalid
"multi_source": {
  "primary_source": "kube-apiserver",
  "secondary_sources": ["kube-apiserver"]  // ERROR: primary cannot be in secondary
}
```

## Security and Detection Validation

### detection_criteria
**Type**: object  
**Validation Rules**:
- ✅ **THRESHOLD_RANGES**: All thresholds must be positive integers
- ✅ **TIME_WINDOW_FORMAT**: Time windows must be valid durations
- ✅ **LOGICAL_CRITERIA**: Detection criteria must be logically consistent

```json
// Valid
"detection_criteria": {
  "rapid_operations": {
    "threshold": 10,
    "time_window": "1_minute"
  }
}

// Invalid
"detection_criteria": {
  "rapid_operations": {
    "threshold": -5  // ERROR: threshold must be positive
  }
}
```

### compliance_framework
**Type**: object  
**Validation Rules**:
- ✅ **STANDARD_ENUM**: Standards must be from valid list
- ✅ **CONTROL_MAPPING**: Controls must be valid for specified standards

```json
// Valid
"compliance_framework": {
  "standards": ["SOX", "PCI-DSS", "GDPR"],
  "controls": ["access_logging", "data_protection"]
}

// Invalid
"compliance_framework": {
  "standards": ["INVALID_STANDARD"]  // ERROR: unknown compliance standard
}
```

## Cross-Field Validation Rules

### Field Dependencies
1. **analysis.kill_chain_phase** requires **analysis.type** to be APT-related
2. **behavioral_analysis.risk_scoring** requires **behavioral_analysis.user_profiling** = true
3. **machine_learning.feature_engineering** requires **machine_learning.model_type**
4. **threat_intelligence.ioc_correlation** requires **threat_intelligence.feed_sources**

### Mutually Exclusive Fields
1. **timeframe** ⊕ **time_range** (exactly one)
2. **user** ⊕ **user_pattern** (prefer pattern for complex queries)
3. **namespace** ⊕ **namespace_pattern** (prefer pattern for complex queries)

### Log Source Compatibility Matrix

| Field | kube-apiserver | openshift-apiserver | oauth-server | oauth-apiserver | node-auditd |
|-------|----------------|---------------------|--------------|-----------------|-------------|
| verb | ✅ | ✅ | ✅ | ✅ | ❌ |
| resource | ✅ | ✅ | ❌ | ✅ | ❌ |
| auth_decision | ❌ | ❌ | ✅ | ✅ | ❌ |
| subresource | ✅ | ✅ | ❌ | ❌ | ❌ |

## Performance Validation Rules

### Query Complexity Scoring
Assign complexity points based on:
- Basic fields: 1 point each
- Pattern matching: 3 points each
- Multi-source correlation: 5 points
- Advanced analysis: 10 points
- Machine learning: 15 points

**Thresholds**:
- < 20 points: **Low complexity** (fast execution)
- 20-50 points: **Medium complexity** (moderate execution time)
- > 50 points: **High complexity** (WARNING: may be slow)

### Resource Usage Validation
- **Memory**: Estimated based on limit and analysis complexity
- **CPU**: Estimated based on analysis type and data volume
- **Network**: Estimated based on multi-source requirements

```json
// Performance estimation example
{
  "limit": 1000,  // +5 points
  "analysis": {
    "type": "machine_learning"  // +15 points
  },
  "multi_source": { ... }  // +5 points
  // Total: 25 points = Medium complexity
}
```

## Error Message Standards

### Error Format
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable error description",
    "field": "field_name",
    "details": {
      "expected": "expected_value_or_format",
      "actual": "actual_value_received",
      "suggestion": "how_to_fix"
    }
  }
}
```

### Error Code Categories
- **FIELD_REQUIRED**: Missing required field
- **FIELD_TYPE**: Wrong field type
- **FIELD_FORMAT**: Invalid field format
- **FIELD_RANGE**: Value out of allowed range
- **FIELD_ENUM**: Value not in allowed enumeration
- **FIELD_DEPENDENCY**: Missing dependent field
- **FIELD_CONFLICT**: Conflicting field values
- **PERFORMANCE_WARNING**: Performance concern

### Example Error Messages
```json
{
  "error": {
    "code": "FIELD_REQUIRED",
    "message": "log_source is required for all queries",
    "field": "log_source",
    "details": {
      "suggestion": "Add log_source field with value: kube-apiserver, openshift-apiserver, oauth-server, oauth-apiserver, or node-auditd"
    }
  }
}
```

```json
{
  "error": {
    "code": "FIELD_RANGE",
    "message": "limit value exceeds maximum allowed",
    "field": "limit",
    "details": {
      "expected": "1-1000",
      "actual": "1500",
      "suggestion": "Reduce limit to 1000 or less"
    }
  }
}
```

## Validation Implementation Guidelines

### Validation Order
1. **JSON Structure**: Parse and basic type validation
2. **Required Fields**: Check for mandatory fields
3. **Field Types**: Validate individual field types
4. **Field Values**: Validate field value constraints  
5. **Cross-Field**: Validate field relationships
6. **Business Logic**: Validate business rule compliance
7. **Performance**: Estimate and warn about performance implications

### Validation Performance
- **Fast Path**: Basic queries (< 1ms validation time)
- **Standard Path**: Intermediate queries (< 5ms validation time)  
- **Complex Path**: Advanced queries (< 20ms validation time)

### Error Handling Strategy
- **Fail Fast**: Stop on first ERROR-level validation failure
- **Collect Warnings**: Continue validation and collect all WARNINGs
- **Provide Context**: Include helpful error messages and suggestions
- **Security Safe**: Never expose internal system details in errors

## Testing and Quality Assurance

### Validation Test Coverage
- ✅ **Positive Tests**: Valid queries pass validation
- ✅ **Negative Tests**: Invalid queries fail with correct errors
- ✅ **Edge Cases**: Boundary conditions and corner cases
- ✅ **Performance Tests**: Validation time within limits
- ✅ **Security Tests**: Malicious input handling

### Validation Metrics
- **Validation Success Rate**: % of queries that pass validation
- **Average Validation Time**: Time to validate typical query
- **Error Classification**: Distribution of error types
- **Performance Impact**: Validation overhead on query processing

## Conclusion

These comprehensive validation rules ensure that the OpenShift Audit Query System maintains data integrity, security, and performance while supporting the full spectrum of query complexity from basic filtering to advanced threat hunting. The validation system provides clear feedback to users and prevents potential security issues or performance problems.