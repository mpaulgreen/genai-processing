# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Context

For every task you perform remember that:
- `./inputs/Audit_Log_Quer_PRD.md` is the PRD
Addtional important input for these tasks are the md files in test/functional folder

## What is the genai-processing app?

**IMPORTANT**: The genai-processing app is specifically the **"GenAI Processing Layer"** component in the overall GenAI-Powered OpenShift Audit Query System architecture (as defined in PRD section 8.1).

**Core Responsibility**: Process natural language audit questions through a simple three-component pipeline:

1. **LLM Engine**: Convert natural language → structured JSON parameters using prompt engineering
   - Input: "Who deleted the customer CRD yesterday?"
   - Output: `{"log_source": "kube-apiserver", "patterns": ["customresourcedefinition", "delete", "customer"], "timeframe": "yesterday", "exclude": ["system:"]}`

2. **Context Manager**: Track conversation history and resolve references (handle follow-ups like "he", "that user", "around that time")

3. **Safety Validator**: Ensure generated queries are safe and reasonable through rule-based validation

**Key Point**: This app is the **translation layer** between natural language input and structured JSON parameters that downstream MCP Servers can understand and convert to safe `oc` commands. It handles the AI intelligence but does NOT execute OpenShift commands directly.

## CRITICAL SAFETY CONSTRAINT

**NEVER MODIFY THE OPENSHIFT CLUSTER**: DO NOT PERFORM any operation that would add, modify, delete, or PATCH a resource on the OpenShift cluster. This system is designed for READ-ONLY audit querying only. All operations must be limited to:
- `oc get` (read operations)
- `oc describe` (read operations) 
- `oc adm node-logs` (read audit logs)
- Other read-only commands only

**ABSOLUTELY FORBIDDEN**: `oc create`, `oc delete`, `oc patch`, `oc apply`, `oc edit`, or any other commands that modify cluster state.

## Project Overview

This is a GenAI-powered audit query processing system built in Go. The system processes natural language audit queries and converts them into structured responses using multiple LLM providers (OpenAI, Claude, Ollama).

## Architecture

The codebase follows a clean architecture pattern with distinct layers:

- **cmd/server**: HTTP server entry point and handlers
- **internal/engine**: Core LLM engine orchestration with provider-agnostic design
- **internal/engine/providers**: LLM provider implementations (OpenAI, Claude, Ollama, Generic)
- **internal/engine/adapters**: Input adapters that format requests for specific models
- **internal/parser**: Response parsing with extractors and normalizers
- **internal/processor**: High-level query processing coordination
- **internal/validator**: Input validation and safety rules
- **internal/config**: Configuration management with YAML support
- **pkg/interfaces**: Core interfaces defining contracts between layers
- **pkg/types**: Shared data types and structures

## Key Components

### LLM Engine Flow
The system uses a provider-adapter pattern where:
1. **InputAdapter** formats queries for specific model requirements
2. **LLMProvider** handles API communication with different LLM services
3. **Extractor** parses model-specific responses into normalized formats
4. **Processor** orchestrates the complete pipeline

### Configuration
Configuration is managed through YAML files in `configs/`:
- `models.yaml`: Provider configurations (OpenAI, Claude, Ollama)
- `prompts.yaml`: System prompts and formatting templates
- `rules.yaml`: Validation rules and safety constraints

## Development Commands

### Build and Test
```bash
# Build the server binary
go build -o server ./cmd/server

# Run all tests
go test -count=1 ./...

# Run tests for specific package
go test ./internal/engine/...
```

### Running the Server
```bash
# Using the provided script (recommended)
./run_server.sh

# Manual execution
set -a; source .env; set +a; ./server
```

### Environment Setup
The server requires these environment variables:
- `OPENAI_API_KEY`: OpenAI API key
- `CLAUDE_API_KEY`: Anthropic Claude API key
- Optional: `OLLAMA_ENDPOINT` for local Ollama instance

### API Testing
```bash
# Health check
curl http://localhost:8080/health

# Query endpoint
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Who deleted the customer CRD yesterday?","session_id":"test"}' | jq .

# Kill server process
lsof -ti:8080 | xargs kill -9
```

## Key Design Patterns

### Provider Factory Pattern
The system uses a factory pattern in `internal/engine/providers/factory.go` to instantiate different LLM providers based on configuration.

### Adapter Pattern
Input adapters in `internal/engine/adapters/` transform generic requests into provider-specific formats while maintaining a consistent interface.

### Parser Extraction Chain
Response parsing uses model-specific extractors that handle the nuances of different LLM response formats before normalizing to a common structure.

## Testing Strategy

Tests are organized by component with:
- Unit tests for individual functions and methods
- Integration tests in `test/integration/` for end-to-end scenarios
- Retry integration tests for resilience testing
- Mock-based testing for external dependencies
- Please always read the test/functional folder to read the basic, intermediate and advanced queries and files related to JSON schema and its validation