# Parser Package

The parser package is a core component of the GenAI Processing Layer responsible for converting raw LLM responses into structured, normalized, and validated `StructuredQuery` objects. It implements a multi-stage pipeline that handles response extraction, field normalization, schema validation, and fallback recovery.

## Architecture Overview

The parser package follows a layered architecture with clear separation of concerns:

```
internal/parser/
├── extractors/     # Model-specific response parsing 
├── normalizers/    # Field mapping and data normalization
└── recovery/       # Fallback handling and retry logic
```

### Core Pipeline Flow

1. **Extraction**: Model-specific extractors parse raw LLM responses into StructuredQuery objects
2. **Normalization**: Field mappers and normalizers clean and standardize the data
3. **Validation**: Schema validators ensure data integrity and business rule compliance
4. **Recovery**: Fallback handlers provide graceful degradation when parsing fails

## Components

### Extractors (`extractors/`)

Model-specific extractors that handle the nuances of different LLM response formats.

#### Available Extractors

- **ClaudeExtractor** (`claude_extractor.go`): Handles Anthropic Claude responses with support for markdown code blocks and XML wrapping
- **OpenAIExtractor** (`openai_extractor.go`): Processes OpenAI GPT responses including function calls and tool calls  
- **OllamaExtractor** (`ollama_extractor.go`): Manages local LLaMA/Ollama model responses with confidence scoring
- **GenericExtractor** (`generic_extractor.go`): Fallback extractor for basic JSON responses

#### Key Features

- **Format Detection**: Automatically detects JSON within markdown code blocks, XML tags, or raw text
- **Confidence Scoring**: Each extractor provides confidence metrics for parsed responses
- **Model Selection**: Factory pattern for selecting appropriate extractor based on model type
- **Error Handling**: Robust parsing with detailed error messages for debugging

#### ExtractorFactory (`extractor_factory.go`)

Central factory for managing model-specific extractors with thread-safe operations.

**Features:**
- **Thread-Safe Registration**: Concurrent-safe parser registration and retrieval
- **Model Aliases**: Maps model variants (e.g., `gpt` → `openai`, `claude-3` → `claude`)  
- **Generic Fallback**: Automatic fallback to generic extractor for unknown models
- **Delegating Parser**: Creates parsers that route to appropriate extractors at runtime
- **Deterministic Output**: Sorted model type lists for consistent behavior

#### Usage Example

```go
import "genai-processing/internal/parser/extractors"

// Create extractor factory
factory := extractors.NewExtractorFactory()

// Register extractors with aliases
factory.Register("claude", NewClaudeExtractor(), "anthropic", "claude-3")
factory.Register("openai", NewOpenAIExtractor(), "gpt", "gpt-4")
factory.SetGeneric(NewGenericExtractor())

// Create extractor for specific model
extractor, err := factory.CreateExtractor("claude-3-5-sonnet")
if err != nil {
    return fmt.Errorf("failed to get extractor: %w", err)
}

// Parse response
query, err := extractor.ParseResponse(rawResponse, "claude-3-5-sonnet")
if err != nil {
    return fmt.Errorf("failed to parse response: %w", err)
}

// Check confidence
confidence := extractor.GetConfidence()
```

### Normalizers (`normalizers/`)

Components that clean, standardize, and validate parsed data.

#### Field Mapper (`field_mapper.go`)

Maps field aliases to canonical forms and standardizes field values.

**Features:**
- **Log Source Aliases**: Maps variations like `oauth-api-server` → `oauth-server`
- **Verb Mapping**: Converts HTTP verbs like `POST` → `create`
- **Response Status**: Normalizes status codes like `ok` → `200`
- **Case Insensitive**: Handles mixed case input gracefully
- **Performance**: < 1ms per mapping operation

**Example:**
```go
mapper := normalizers.NewFieldMapper()
err := mapper.MapFields(query)  // Modifies query in-place
```

#### JSON Normalizer (`json_normalizer.go`)

Normalizes JSON structure and applies business rules.

**Features:**
- **Default Values**: Sets default log_source and limit values
- **Limit Bounds**: Enforces min/max limits (1-1000)
- **StringOrArray Fields**: Standardizes field formats and trims whitespace
- **Timeframe Keywords**: Normalizes time expressions like `1h` → `1_hour_ago`
- **Time Range Validation**: Ensures chronological order and extends identical times
- **Performance**: < 1ms per normalization operation

**Example:**
```go
normalizer := normalizers.NewJSONNormalizer()
err := normalizer.Normalize(query)  // Modifies query in-place
```

#### Schema Validator (`schema_validator.go`)

Validates structured queries against business rules and constraints.

**Features:**
- **Required Fields**: Ensures log_source is present and valid
- **Range Validation**: Validates limit bounds (1-1000) 
- **Enum Validation**: Checks auth_decision, sort_order, sort_by values
- **Timeframe Validation**: Validates timeframe keywords and formats
- **Time Range Logic**: Ensures end time is after start time
- **Business Hours**: Validates business hour constraints
- **Analysis Types**: Validates analysis type and parameters
- **Performance**: < 100µs per validation operation

**Example:**
```go
validator := normalizers.NewSchemaValidator()
err := validator.ValidateSchema(query)  // Returns validation errors
```

### Recovery (`recovery/`)

Fallback mechanisms for handling parsing failures and providing graceful degradation.

#### Fallback Handler (`fallback_handler.go`)

Creates minimal valid queries when parsing fails using heuristic analysis.

**Features:**
- **Log Source Heuristics**: Detects OAuth/OpenShift keywords for source selection
- **Timeframe Extraction**: Identifies time keywords (today, yesterday, hour)
- **Safe Defaults**: Provides conservative defaults (kube-apiserver, limit 20)
- **Case Insensitive**: Handles mixed case input
- **Fast Performance**: < 100µs per fallback operation
- **Model Agnostic**: Works with any model type

**Example:**
```go
handler := recovery.NewFallbackHandler()
query, err := handler.CreateMinimalQuery(rawResponse, "gpt-4", "show oauth failures today")
```

#### Retry Parser (`retry_parser.go`)

Implements retry logic with exponential backoff for transient failures.

**Features:**
- **Configurable Retries**: Customizable retry attempts and delays
- **Exponential Backoff**: Intelligent retry timing
- **Error Classification**: Differentiates between retryable and permanent errors
- **Circuit Breaker**: Prevents cascading failures

## Testing

### Test Coverage

The parser package has comprehensive test coverage with over 200 test cases:

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **Performance Tests**: Validate performance targets
- **Edge Case Tests**: Handle error conditions and boundary cases
- **Benchmark Tests**: Measure and validate performance

### Running Tests

#### All Parser Tests
```bash
# Run all parser package tests
go test ./internal/parser/...

# Run with verbose output
go test -v ./internal/parser/...

# Run with coverage
go test -cover ./internal/parser/...
```

#### Component-Specific Tests
```bash
# Test extractors only
go test ./internal/parser/extractors/

# Test normalizers only  
go test ./internal/parser/normalizers/

# Test recovery components only
go test ./internal/parser/recovery/
```

#### Individual Test Files
```bash
# Test specific extractor
go test -v ./internal/parser/extractors -run TestClaudeExtractor

# Test field mapper
go test -v ./internal/parser/normalizers -run TestFieldMapper

# Test fallback handler
go test -v ./internal/parser/recovery -run TestFallbackHandler
```

#### Performance Benchmarks
```bash
# Run performance benchmarks
go test -bench=. ./internal/parser/...

# Run specific benchmark
go test -bench=BenchmarkFieldMapper ./internal/parser/normalizers/

# Run benchmark with memory profiling
go test -bench=. -benchmem ./internal/parser/...
```

#### Test with Build Tags
```bash
# Run integration tests (if using build tags)
go test -tags=integration ./internal/parser/...

# Run performance tests only
go test -tags=performance ./internal/parser/...
```

### Performance Targets

The parser package meets strict performance requirements:

| Component | Target | Actual |
|-----------|--------|---------|
| Field Mapper | < 1ms | ~315ns |
| JSON Normalizer | < 1ms | ~261ns |
| Schema Validator | < 100µs | ~9.5ns |
| Fallback Handler | < 100µs | ~249ns |

## Usage Patterns

### Basic Usage
```go
import (
    "genai-processing/internal/parser/extractors"
    "genai-processing/internal/parser/normalizers"
    "genai-processing/internal/parser/recovery"
)

// Parse raw response
extractor, _ := extractors.NewExtractorFactory().GetExtractor("claude-3")
query, err := extractor.ParseResponse(rawResponse, "claude-3")
if err != nil {
    // Fallback to heuristic parsing
    fallback := recovery.NewFallbackHandler()
    query, _ = fallback.CreateMinimalQuery(rawResponse, "claude-3", originalQuery)
}

// Normalize and validate
mapper := normalizers.NewFieldMapper()
normalizer := normalizers.NewJSONNormalizer()
validator := normalizers.NewSchemaValidator()

mapper.MapFields(query)
normalizer.Normalize(query)
err = validator.ValidateSchema(query)
```

### Pipeline Pattern
```go
// Create processing pipeline
pipeline := []func(*types.StructuredQuery) error{
    mapper.MapFields,
    normalizer.Normalize,
    validator.ValidateSchema,
}

// Process query through pipeline
for _, step := range pipeline {
    if err := step(query); err != nil {
        return fmt.Errorf("pipeline step failed: %w", err)
    }
}
```

### Error Handling
```go
// Handle extraction errors
query, err := extractor.ParseResponse(rawResponse, modelType)
if err != nil {
    log.Printf("Primary extraction failed: %v", err)
    
    // Try fallback
    fallback := recovery.NewFallbackHandler()
    query, err = fallback.CreateMinimalQuery(rawResponse, modelType, originalQuery)
    if err != nil {
        return fmt.Errorf("fallback parsing failed: %w", err)
    }
    
    log.Printf("Fallback parsing succeeded with confidence: %f", 0.3)
}
```

## Error Handling

The parser package uses structured error handling with detailed context:

```go
// Validation errors include field-specific details
type ValidationError struct {
    Field   string
    Value   interface{}
    Rule    string
    Message string
}

// Extraction errors preserve original content
type ExtractionError struct {
    ModelType string
    Content   string
    Reason    string
}
```

## Configuration

### Extractor Selection

Extractors are selected automatically based on model type:

- `claude`, `anthropic` → ClaudeExtractor
- `openai`, `gpt` → OpenAIExtractor  
- `ollama`, `llama` → OllamaExtractor
- Others → GenericExtractor

### Validation Rules

Schema validation rules are defined in the validator:

- **Log Sources**: `kube-apiserver`, `oauth-server`, `openshift-apiserver`
- **Auth Decisions**: `allow`, `forbid`, `error`
- **Sort Orders**: `asc`, `desc`
- **Limit Range**: 1-1000
- **Analysis Types**: `multi_namespace_access`, `privilege_escalation`, etc.

## Best Practices

1. **Always Use Fallback**: Implement fallback handling for production resilience
2. **Validate Performance**: Run benchmarks to ensure performance targets
3. **Handle Errors Gracefully**: Provide meaningful error messages and recovery paths
4. **Test Edge Cases**: Include nil checks, empty inputs, and boundary conditions
5. **Monitor Confidence**: Track extractor confidence scores for quality metrics
6. **Log Failures**: Log parsing failures for debugging and improvement

## Dependencies

The parser package depends on:

- `genai-processing/pkg/types`: Core data structures
- `genai-processing/pkg/interfaces`: Interface definitions
- Standard library: `strings`, `encoding/json`, `fmt`, `time`

## Contributing

When adding new extractors or normalizers:

1. Implement the required interfaces
2. Add comprehensive unit tests (>90% coverage)
3. Include performance benchmarks
4. Add error handling and edge cases
5. Update this README with new components
6. Ensure backward compatibility

## Troubleshooting

### Common Issues

**Extraction Failures**
- Check raw response format
- Verify model type mapping
- Enable debug logging for content analysis

**Validation Errors**
- Review field values against validation rules
- Check required fields are present
- Validate data types and ranges

**Performance Issues**
- Run benchmarks to identify bottlenecks
- Check for memory leaks in string processing
- Optimize string operations and allocations

### Debug Commands

```bash
# Run tests with race detection
go test -race ./internal/parser/...

# Profile memory usage
go test -memprofile=mem.prof ./internal/parser/...

# Profile CPU usage  
go test -cpuprofile=cpu.prof ./internal/parser/...

# Verbose test output with timing
go test -v -test.timeout=30s ./internal/parser/...
```