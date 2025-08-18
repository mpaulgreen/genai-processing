# Enhanced JSON Schema for OpenShift Audit Query System

## Overview

This document defines the comprehensive JSON schema for the GenAI-Powered OpenShift Audit Query System, supporting all query complexity levels from basic filtering to advanced threat hunting and behavioral analytics.

## Schema Design Principles

1. **Progressive Complexity**: Basic queries use simple fields, advanced queries leverage nested objects
2. **Backward Compatibility**: All existing queries remain valid
3. **Type Safety**: Strong typing with validation constraints
4. **Extensibility**: Schema supports future enhancements
5. **Performance**: Efficient serialization and parsing

## Complete JSON Schema Definition

### Root Schema Structure

```json
{
  "type": "object",
  "required": ["log_source"],
  "properties": {
    // Core required fields
    "log_source": { ... },
    
    // Basic filtering fields
    "verb": { ... },
    "resource": { ... },
    "namespace": { ... },
    "user": { ... },
    "timeframe": { ... },
    "limit": { ... },
    
    // Advanced filtering fields
    "exclude_users": { ... },
    "user_pattern": { ... },
    "namespace_pattern": { ... },
    "resource_name_pattern": { ... },
    "request_uri_pattern": { ... },
    
    // Response and status fields
    "response_status": { ... },
    "auth_decision": { ... },
    "authorization_reason_pattern": { ... },
    "response_message_pattern": { ... },
    
    // Network and source fields
    "source_ip": { ... },
    "correlation_fields": { ... },
    
    // Resource-specific fields
    "subresource": { ... },
    "include_changes": { ... },
    "missing_annotation": { ... },
    "exclude_resources": { ... },
    
    // Grouping and sorting
    "group_by": { ... },
    "sort_by": { ... },
    "sort_order": { ... },
    
    // Time-based analysis
    "time_range": { ... },
    "business_hours": { ... },
    "temporal_analysis": { ... },
    
    // Advanced analysis configuration
    "analysis": { ... },
    "multi_source": { ... },
    "behavioral_analysis": { ... },
    "threat_intelligence": { ... },
    "machine_learning": { ... },
    
    // Detection and security
    "detection_criteria": { ... },
    "security_context": { ... },
    "compliance_framework": { ... }
  }
}
```

## Core Fields (Required/Basic)

### log_source
**Type**: string (required)  
**Description**: Source of audit logs  
**Valid Values**: `kube-apiserver`, `openshift-apiserver`, `oauth-server`, `oauth-apiserver`, `node-auditd`

```json
"log_source": "kube-apiserver"
```

### limit  
**Type**: integer  
**Description**: Maximum number of results to return  
**Range**: 1-1000  
**Default**: 20

```json
"limit": 50
```

## Basic Filtering Fields

### verb
**Type**: StringOrArray  
**Description**: HTTP verbs to filter on  
**Examples**: Single verb or array of verbs

```json
"verb": "create"
// OR
"verb": ["create", "update", "patch", "delete"]
```

### resource
**Type**: StringOrArray  
**Description**: Kubernetes resource types to filter on  

```json
"resource": "pods"
// OR  
"resource": ["secrets", "configmaps", "serviceaccounts"]
```

### namespace
**Type**: StringOrArray  
**Description**: Specific namespace(s) to filter on

```json
"namespace": "default"
// OR
"namespace": ["production", "staging"]
```

### user
**Type**: StringOrArray  
**Description**: Specific user(s) to filter on

```json
"user": "john.doe@example.com"
// OR
"user": ["admin@company.com", "operator@company.com"]
```

### timeframe
**Type**: string  
**Description**: Predefined time periods for filtering  
**Valid Values**: `today`, `yesterday`, `1_hour_ago`, `6_hours_ago`, `24_hours_ago`, `7_days_ago`, `30_days_ago`, `last_week`, `last_month`

```json
"timeframe": "24_hours_ago"
```

## Advanced Filtering Fields

### exclude_users
**Type**: array of strings  
**Description**: User patterns to exclude from results

```json
"exclude_users": ["system:", "kube-", "openshift-"]
```

### user_pattern
**Type**: string (regex)  
**Description**: Regular expression pattern for user matching

```json
"user_pattern": "^admin@.*\\.company\\.com$"
```

### namespace_pattern
**Type**: string (regex)  
**Description**: Regular expression pattern for namespace matching

```json
"namespace_pattern": "^(prod-.*|staging-.*)$"
```

### resource_name_pattern
**Type**: string (regex)  
**Description**: Regular expression pattern for resource name matching

```json
"resource_name_pattern": "secret-.*-credentials"
```

### request_uri_pattern
**Type**: string (regex)  
**Description**: Pattern for matching request URIs

```json
"request_uri_pattern": "/api/v1/namespaces/.*/secrets/.*"
```

## Response and Status Fields

### response_status
**Type**: StringOrArray  
**Description**: HTTP response status codes to filter on

```json
"response_status": "403"
// OR
"response_status": ["401", "403", "500"]
// OR (range syntax)
"response_status": ">=400"
```

### auth_decision
**Type**: string  
**Description**: Authentication decision filter  
**Valid Values**: `allow`, `error`, `forbid`

```json
"auth_decision": "forbid"
```

### authorization_reason_pattern
**Type**: string (regex)  
**Description**: Pattern for authorization reason matching

```json
"authorization_reason_pattern": "cluster-admin|admin"
```

### response_message_pattern
**Type**: string (regex)  
**Description**: Pattern for response message matching

```json
"response_message_pattern": "(?i)(unauthorized|forbidden|denied)"
```

## Network and Source Fields

### source_ip
**Type**: StringOrArray  
**Description**: Source IP address filtering

```json
"source_ip": "192.168.1.100"
// OR
"source_ip": ["10.0.0.0/8", "172.16.0.0/12"]
```

### correlation_fields
**Type**: array of strings  
**Description**: Fields to correlate across log entries

```json
"correlation_fields": ["user", "source_ip", "timing", "resource"]
```

## Resource-Specific Fields

### subresource
**Type**: string  
**Description**: Kubernetes subresource (exec, scale, status, etc.)

```json
"subresource": "exec"
```

### include_changes
**Type**: boolean  
**Description**: Include before/after object comparisons

```json
"include_changes": true
```

### missing_annotation
**Type**: string  
**Description**: Annotation that should be missing from the event

```json
"missing_annotation": "admission.k8s.io/audit"
```

### exclude_resources
**Type**: array of strings  
**Description**: Resource patterns to exclude from results

```json
"exclude_resources": ["events", "endpoints", "endpointslices"]
```

## Grouping and Sorting

### group_by
**Type**: StringOrArray  
**Description**: Fields to group results by

```json
"group_by": "username"
// OR
"group_by": ["namespace", "resource", "verb"]
```

### sort_by
**Type**: string  
**Description**: Field to sort results by  
**Valid Values**: `timestamp`, `user`, `resource`, `count`, `namespace`

```json
"sort_by": "timestamp"
```

### sort_order
**Type**: string  
**Description**: Sort direction  
**Valid Values**: `asc`, `desc`

```json
"sort_order": "desc"
```

## Time-Based Analysis

### time_range
**Type**: object  
**Description**: Custom time range with specific start and end timestamps

```json
"time_range": {
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-01T23:59:59Z",
  "timezone": "UTC"
}
```

### business_hours
**Type**: object  
**Description**: Business hours filtering configuration

```json
"business_hours": {
  "outside_only": true,
  "start_hour": 9,
  "end_hour": 17,
  "timezone": "EST",
  "include_weekends": false
}
```

### temporal_analysis
**Type**: object  
**Description**: Advanced time-based pattern analysis

```json
"temporal_analysis": {
  "pattern_type": "periodic",
  "interval_detection": true,
  "anomaly_threshold": 2.0,
  "baseline_window": "30_days",
  "seasonality_detection": true
}
```

## Advanced Analysis Configuration

### analysis
**Type**: object  
**Description**: Advanced analysis configuration for complex security investigations

```json
"analysis": {
  "type": "apt_reconnaissance_detection",
  "kill_chain_phase": "reconnaissance",
  "multi_stage_correlation": true,
  "statistical_analysis": {
    "pattern_deviation_threshold": 2.5,
    "confidence_interval": 0.95,
    "sample_size_minimum": 100
  },
  "threshold": 5,
  "time_window": "15_minutes"
}
```

**Analysis Types:**
- `multi_namespace_access`
- `excessive_reads`
- `privilege_escalation`
- `anomaly_detection`
- `correlation`
- `apt_reconnaissance_detection`
- `lateral_movement_detection`
- `data_exfiltration_detection`
- `persistence_mechanism_detection`
- `defense_evasion_detection`
- `credential_harvesting_detection`
- `supply_chain_attack_detection`
- `living_off_the_land_detection`
- `c2_communication_detection`
- `compliance_violation_detection`

**Kill Chain Phases:**
- `reconnaissance`
- `initial_access`
- `execution`
- `persistence`
- `privilege_escalation`
- `defense_evasion`
- `credential_access`
- `discovery`
- `lateral_movement`
- `collection`
- `command_and_control`
- `exfiltration`
- `impact`

### multi_source
**Type**: object  
**Description**: Multi-source correlation configuration

```json
"multi_source": {
  "primary_source": "kube-apiserver",
  "secondary_sources": ["oauth-server", "node-auditd"],
  "correlation_window": "30_minutes",
  "correlation_fields": ["user", "source_ip", "timestamp"],
  "join_type": "inner"
}
```

### behavioral_analysis
**Type**: object  
**Description**: User and system behavior analytics

```json
"behavioral_analysis": {
  "user_profiling": true,
  "baseline_comparison": true,
  "risk_scoring": {
    "enabled": true,
    "algorithm": "weighted_sum",
    "risk_factors": ["privilege_level", "resource_sensitivity", "timing_anomaly"]
  },
  "anomaly_detection": {
    "algorithm": "isolation_forest",
    "contamination": 0.1,
    "sensitivity": 0.8
  }
}
```

### threat_intelligence
**Type**: object  
**Description**: Threat intelligence correlation and analysis

```json
"threat_intelligence": {
  "ioc_correlation": true,
  "attack_pattern_matching": true,
  "threat_actor_attribution": true,
  "feed_sources": ["mitre_att&ck", "custom_feeds"],
  "confidence_threshold": 0.7
}
```

### machine_learning
**Type**: object  
**Description**: Machine learning analysis parameters

```json
"machine_learning": {
  "model_type": "anomaly_detection",
  "feature_engineering": {
    "temporal_features": true,
    "behavioral_features": true,
    "network_features": true
  },
  "training_window": "30_days",
  "prediction_threshold": 0.8
}
```

## Detection and Security

### detection_criteria
**Type**: object  
**Description**: Specific detection criteria for security analysis

```json
"detection_criteria": {
  "rapid_operations": {
    "threshold": 10,
    "time_window": "1_minute"
  },
  "privilege_escalation_indicators": true,
  "lateral_movement_patterns": true,
  "data_access_anomalies": true
}
```

### security_context
**Type**: object  
**Description**: Security context and constraint analysis

```json
"security_context": {
  "scc_violations": true,
  "pod_security_standards": "restricted",
  "privilege_analysis": true,
  "capability_monitoring": ["SYS_ADMIN", "NET_ADMIN"]
}
```

### compliance_framework
**Type**: object  
**Description**: Compliance framework monitoring

```json
"compliance_framework": {
  "standards": ["SOX", "PCI-DSS", "GDPR", "HIPAA"],
  "controls": ["access_logging", "data_protection", "audit_trail"],
  "reporting": {
    "format": "detailed",
    "include_evidence": true
  }
}
```

## Type Definitions

### StringOrArray
**Description**: Flexible type that accepts either a single string or an array of strings

```typescript
type StringOrArray = string | string[]
```

### TimeWindow
**Description**: Predefined time window values

```typescript
type TimeWindow = "1_minute" | "5_minutes" | "15_minutes" | "30_minutes" | 
                  "1_hour" | "6_hours" | "12_hours" | "24_hours" | 
                  "7_days" | "30_days"
```

### LogSource
**Description**: Available audit log sources

```typescript
type LogSource = "kube-apiserver" | "openshift-apiserver" | 
                 "oauth-server" | "oauth-apiserver" | "node-auditd"
```

## Example Queries by Complexity Level

### Basic Query Example
```json
{
  "log_source": "kube-apiserver",
  "verb": "delete",
  "resource": "secrets",
  "timeframe": "24_hours_ago",
  "exclude_users": ["system:", "kube-"],
  "limit": 20
}
```

### Intermediate Query Example
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
  "sort_by": "count",
  "sort_order": "desc",
  "limit": 50
}
```

### Advanced Query Example
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

## Schema Validation Rules

### Field Dependencies
- `time_range` requires both `start` and `end`
- `business_hours` requires `start_hour` and `end_hour`
- `analysis.statistical_analysis` requires `analysis.type`
- `multi_source` requires at least one secondary source

### Mutual Exclusions
- `timeframe` and `time_range` are mutually exclusive
- Basic filtering and advanced analysis may have different validation rules

### Value Constraints
- `limit`: 1 ≤ limit ≤ 1000
- `business_hours.start_hour`: 0 ≤ hour ≤ 23
- `confidence_interval`: 0.0 ≤ confidence ≤ 1.0
- `threshold`: threshold ≥ 1

## Migration Strategy

### Phase 1: Core Schema Extension
1. Add new fields to `StructuredQuery` struct
2. Update validation logic
3. Maintain backward compatibility

### Phase 2: Advanced Features
1. Implement complex object validation
2. Add machine learning parameter support
3. Integrate threat intelligence correlation

### Phase 3: Optimization
1. Performance optimization for large schemas
2. Schema versioning implementation
3. Dynamic validation rules

## Conclusion

This enhanced JSON schema provides comprehensive support for all query complexity levels while maintaining backward compatibility and performance. The schema enables enterprise-grade security monitoring, threat hunting, and compliance automation capabilities required for modern OpenShift environments.