# Configuration Management Package

This package provides comprehensive configuration management for the genai-processing application, supporting multiple configuration sources, validation, and environment-specific overrides.

## Overview

The config package handles all configuration aspects of the genai-processing application including:

- **Server Configuration**: HTTP server settings, timeouts, and limits
- **Models Configuration**: LLM provider settings for Claude, OpenAI, Ollama, and custom models
- **Prompts Configuration**: System prompts, examples, formats, and validation rules
- **Rules Configuration**: Safety rules, sanitization, query limits, and business constraints
- **Context Configuration**: Context manager settings for session management and persistence

## Architecture

```
internal/config/
├── config.go          # Core configuration structures and validation
├── loader.go          # Configuration loading and environment overrides
├── config_test.go     # Configuration structure tests
├── loader_test.go     # Configuration loading tests
└── README.md          # This documentation
```

## Configuration Sources

The configuration system uses a **layered approach** with the following precedence:

1. **Default Values** (lowest priority)
2. **YAML Configuration Files** (configs/*.yaml)
3. **Environment Variables** (highest priority)

### Configuration Files

| File | Purpose | Required |
|------|---------|----------|
| `configs/models.yaml` | LLM provider configurations | Optional |
| `configs/prompts.yaml` | System prompts and examples | Optional |
| `configs/rules.yaml` | Safety and validation rules | Optional |
| `configs/context.yaml` | Context manager settings | Optional |

If configuration files don't exist, the system uses sensible defaults.

## Core Components

### AppConfig Structure

```go
type AppConfig struct {
    Server  ServerConfig  // HTTP server configuration
    Models  ModelsConfig  // LLM provider configurations
    Prompts PromptsConfig // Prompt templates and validation
    Rules   RulesConfig   // Safety and validation rules
    Context ContextConfig // Context manager configuration
}
```

### Configuration Loader

```go
// Create a loader for the configs directory
loader := NewLoader("./configs")

// Load configuration with defaults, files, and environment overrides
config, err := loader.LoadConfig()

// Load from a specific file
config, err := loader.LoadConfigFromFile("./configs/custom.yaml")

// Validate configuration
result := config.Validate()
```

## Configuration Details

### Server Configuration

HTTP server settings and security constraints:

```yaml
server:
  port: "8080"                    # Server port
  host: "0.0.0.0"                # Bind address
  read_timeout: "30s"            # Request read timeout
  write_timeout: "30s"           # Response write timeout
  idle_timeout: "60s"            # Keep-alive timeout
  shutdown_timeout: "10s"        # Graceful shutdown timeout
  max_request_size: 1048576      # Maximum request size in bytes
```

**Environment Overrides:**
- `SERVER_PORT` - Server port
- `SERVER_HOST` - Bind address
- `SERVER_READ_TIMEOUT` - Read timeout
- `SERVER_WRITE_TIMEOUT` - Write timeout
- `SERVER_IDLE_TIMEOUT` - Idle timeout
- `SERVER_SHUTDOWN_TIMEOUT` - Shutdown timeout
- `SERVER_MAX_REQUEST_SIZE` - Maximum request size

### Models Configuration

LLM provider settings supporting multiple providers:

```yaml
models:
  default_provider: "claude"
  providers:
    claude:
      provider: "anthropic"
      endpoint: "https://api.anthropic.com/v1/messages"
      api_key: "${CLAUDE_API_KEY:-placeholder-key}"
      model_name: "claude-3-5-sonnet-20241022"
      max_tokens: 4000
      temperature: 0.1
      timeout: "60s"
      retry_attempts: 3
      retry_delay: "1s"
      input_adapter: "claude_input_adapter"
      output_parser: "claude_extractor"
      prompt_formatter: "claude_formatter"
    
    openai:
      provider: "openai"
      endpoint: "https://api.openai.com/v1/chat/completions"
      api_key: "${OPENAI_API_KEY:-placeholder-key}"
      model_name: "gpt-4"
      max_tokens: 2000
      temperature: 0.1
      timeout: "30s"
      retry_attempts: 2
      retry_delay: "500ms"
      input_adapter: "openai_input_adapter"
      output_parser: "openai_extractor"
      prompt_formatter: "openai_formatter"
```

**Environment Overrides:**
- `DEFAULT_PROVIDER` - Default provider name
- `CLAUDE_API_KEY` - Claude API key
- `OPENAI_API_KEY` - OpenAI API key
- `CLAUDE_MODEL_NAME` - Claude model name
- `OPENAI_MODEL_NAME` - OpenAI model name
- `CLAUDE_MAX_TOKENS` - Claude max tokens
- `OPENAI_MAX_TOKENS` - OpenAI max tokens

### Rules Configuration

Comprehensive safety and validation rules:

```yaml
rules:
  safety_rules:
    allowed_log_sources:
      - "kube-apiserver"
      - "openshift-apiserver"
      - "oauth-server"
    allowed_verbs:
      - "get"
      - "list"
      - "create"
      - "update"
      - "patch"
      - "delete"
    forbidden_patterns:
      - "rm -rf"
      - "delete --all"
      - "system:admin"
    timeframe_limits:
      max_days_back: 90
      default_limit: 20
      max_limit: 1000
      min_limit: 1
      allowed_timeframes:
        - "today"
        - "yesterday"
        - "1_hour_ago"
        - "7_days_ago"
    required_fields:
      - "log_source"
  
  sanitization:
    max_query_length: 10000
    max_pattern_length: 500
    valid_regex_pattern: "^[a-zA-Z0-9\\-_\\*\\.\\?\\+\\[\\]\\{\\}\\(\\)\\|\\\\/\\s]+$"
    valid_ip_pattern: "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
    forbidden_chars:
      - "<"
      - ">"
      - "&"
```

### Context Configuration

Context manager settings for session management:

```yaml
context:
  cleanup_interval: "5m"           # Session cleanup frequency
  session_timeout: "24h"           # Session expiration time
  max_sessions: 10000              # Maximum concurrent sessions
  max_memory_mb: 100               # Memory limit in MB
  enable_persistence: true         # Enable session persistence
  persistence_path: "./sessions"   # Persistence directory
  persistence_format: "json"       # Storage format (json/gob)
  persistence_interval: "30s"      # Persistence frequency
  enable_async_persistence: true   # Enable async persistence
```

**Environment Overrides:**
- `CONTEXT_CLEANUP_INTERVAL` - Cleanup frequency
- `CONTEXT_SESSION_TIMEOUT` - Session timeout
- `CONTEXT_MAX_SESSIONS` - Maximum sessions
- `CONTEXT_MAX_MEMORY_MB` - Memory limit
- `CONTEXT_ENABLE_PERSISTENCE` - Enable persistence
- `CONTEXT_PERSISTENCE_PATH` - Persistence path
- `CONTEXT_PERSISTENCE_FORMAT` - Storage format
- `CONTEXT_PERSISTENCE_INTERVAL` - Persistence interval
- `CONTEXT_ENABLE_ASYNC_PERSISTENCE` - Enable async persistence

## Usage Examples

### Basic Configuration Loading

```go
package main

import (
    "log"
    "genai-processing/internal/config"
)

func main() {
    // Load configuration from ./configs directory
    loader := config.NewLoader("./configs")
    cfg, err := loader.LoadConfig()
    if err != nil {
        log.Fatal("Failed to load configuration:", err)
    }
    
    // Validate configuration
    result := cfg.Validate()
    if !result.Valid {
        log.Fatal("Invalid configuration:", result.Errors)
    }
    
    // Use configuration
    log.Printf("Server will run on port %s", cfg.Server.Port)
    log.Printf("Default LLM provider: %s", cfg.Models.DefaultProvider)
}
```

### Context Manager Integration

```go
import (
    "genai-processing/internal/config"
    "genai-processing/internal/context"
)

func setupContextManager(cfg *config.AppConfig) *context.ContextManager {
    // Convert configuration for backward compatibility
    contextConfig := cfg.Context.ToContextManagerConfig()
    
    // Create context manager with unified configuration
    return context.NewContextManagerWithConfig(contextConfig)
}
```

### Saving Configuration

```go
// Save individual configuration sections
loader := config.NewLoader("./configs")

// Save models configuration
err := loader.SaveModelsConfig(cfg)

// Save prompts configuration  
err := loader.SavePromptsConfig(cfg)

// Save rules configuration
err := loader.SaveRulesConfig(cfg)

// Save context configuration
err := loader.SaveContextConfig(cfg)

// Save complete configuration to a file
err := loader.SaveConfig(cfg, "./configs/complete.yaml")
```

## Testing Commands

### Run All Tests

```bash
# Run all configuration tests
go test ./internal/config -v

# Run with race detection
go test ./internal/config -v -race

# Run with coverage
go test ./internal/config -v -cover

# Run specific test patterns
go test ./internal/config -v -run TestRulesConfig
go test ./internal/config -v -run TestContextConfig
go test ./internal/config -v -run TestLoader
```

### Test Configuration Loading

```bash
# Test with environment variables
CONTEXT_MAX_SESSIONS=5000 CLAUDE_API_KEY=test go test ./internal/config -v -run TestEnvironmentOverrides

# Test configuration validation
go test ./internal/config -v -run TestValidate

# Test configuration file loading
go test ./internal/config -v -run TestLoadConfig
```

### Example Test Commands

```bash
# Test rules configuration loading and validation
go test ./internal/config -v -run TestLoadConfig_WithRulesFile

# Test context configuration with environment overrides
go test ./internal/config -v -run TestLoadConfig_ContextEnvironmentOverrides

# Test configuration file paths and management
go test ./internal/config -v -run TestGetConfigFilePaths

# Test configuration saving functionality
go test ./internal/config -v -run TestSave

# Test backward compatibility with context manager
go test ./internal/config -v -run TestToContextManagerConfig
```

## Configuration Validation

The configuration system provides comprehensive validation:

### Validation Features

- **Required Fields**: Ensures critical configuration is present
- **Type Validation**: Validates data types and ranges
- **Cross-Field Validation**: Ensures consistent configuration across fields
- **Business Rule Validation**: Enforces application-specific constraints
- **Environment Variable Validation**: Validates environment overrides

### Validation Results

```go
result := config.Validate()
if !result.Valid {
    for _, err := range result.Errors {
        log.Printf("Configuration error: %s", err)
    }
    for _, warning := range result.Warnings {
        log.Printf("Configuration warning: %s", warning)
    }
}
```

## Default Values

The system provides production-ready defaults for all configuration:

- **Server**: Runs on port 8080 with secure timeouts
- **Models**: Claude as default provider with fallback configurations
- **Rules**: Comprehensive safety rules for OpenShift audit queries
- **Context**: Balanced session management with 24h timeouts
- **Prompts**: OpenShift-specific system prompts and examples

## Error Handling

The configuration system provides detailed error reporting:

```go
// Configuration loading errors
config, err := loader.LoadConfig()
if err != nil {
    // Handle file reading, parsing, or validation errors
}

// Validation errors
result := config.Validate()
if !result.Valid {
    // Handle specific validation failures
    for _, err := range result.Errors {
        // Process each validation error
    }
}
```

## Best Practices

1. **Environment Variables**: Use environment variables for secrets and deployment-specific settings
2. **Configuration Files**: Use YAML files for complex structured configuration
3. **Validation**: Always validate configuration before using it
4. **Defaults**: Rely on sensible defaults for development environments
5. **Testing**: Test configuration loading and validation thoroughly
6. **Security**: Never commit API keys or secrets to configuration files

## Troubleshooting

### Common Issues

1. **Configuration File Not Found**
   - Solution: Files are optional - defaults will be used
   - Check: Verify file paths and permissions

2. **Validation Failures**
   - Solution: Check validation error messages
   - Common: Missing required fields, invalid ranges

3. **Environment Variable Issues**
   - Solution: Check variable names and formats
   - Example: Duration format should be "30s", "5m", "1h"

4. **API Key Issues**
   - Solution: Use environment variable placeholders
   - Format: `"${API_KEY:-fallback-value}"`

### Debug Commands

```bash
# Test configuration loading with verbose output
go test ./internal/config -v -run TestLoadConfig_DefaultConfig

# Validate specific configuration sections
go test ./internal/config -v -run TestValidate

# Check environment variable processing
go test ./internal/config -v -run TestEnvironmentOverrides
```

This configuration system provides robust, flexible, and secure configuration management for the genai-processing application with comprehensive testing and validation capabilities.