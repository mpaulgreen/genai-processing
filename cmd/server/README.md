### GenAI Audit Query API – Response Structure Guide

This document explains the API response structures in detail and maps them to the corresponding code paths in this repository.

### Endpoints

- POST `/query`: Process a natural language audit query into a structured query.
- GET `/health`: Health check for the service.
- GET `/openapi.json`: OpenAPI 3.0 spec describing this API.
- GET `/docs`: Swagger UI for interactive documentation.
- GET `/redoc`: ReDoc for interactive documentation.

All the above routes are registered in `cmd/server/handlers.go` within `setupRoutes` and served by the HTTP server in `cmd/server/main.go`.

### Request: ProcessingRequest

Defined in `pkg/types/query.go` as `ProcessingRequest`.

- `query` (string, required): Natural language query to parse.
- `session_id` (string, required): Session identifier (conversation/context correlation).
- `model_type` (string, optional): Preferred model type/provider hint.

Validation rules enforced by `validateProcessingRequest` in `cmd/server/handlers.go`:

- `query` must be non-empty and ≤ 1000 characters.
- `session_id` must be non-empty and ≤ 100 characters.
- `model_type` (if provided) ≤ 50 characters.

### 200 OK Response: ProcessingResponse

Two sources describe the shape:

1) The Go type `ProcessingResponse` in `pkg/types/query.go` sets the top-level shape used by the processor.
2) The OpenAPI spec served by `OpenAPIHandler` in `cmd/server/handlers.go` documents the schema and example payloads.

The live 200 OK response contains:

- `structured_query` (object): The normalized, structured representation produced from the NL query. Typical fields:
  - `log_source` (string | null)
  - `verb` (string | null)
  - `resource` (string | null)
  - `namespace` (string | null)
  - `user` (string | null)
  - `timeframe` (string | null)
  - `limit` (number | null)
  - `response_status` (string | null)
  - `exclude_users` (array[string] | null)
  - `resource_name_pattern` (string | null)
  - `source_ip` (string | null)
  - `group_by` (string | null)

- `confidence` (number): Confidence score (0.0–1.0) of the structured parse.

- `validation_info` (object): Summarizes safety/validation results for the structured query.
  - `is_valid` (boolean)
  - `rule_name` (string)
  - `severity` (string)
  - `message` (string)
  - `timestamp` (RFC3339 string)
  - `query_snapshot` (object): A copy of the `structured_query` at validation time.
  - `details` (object)
    - `total_rules_applied` (number)
    - `validation_timestamp` (RFC3339 string)
    - `rule_results` (object): Per-rule results; keys are rule identifiers. In the current implementation and examples, these include:
      - `patterns`
      - `required_fields`
      - `sanitization`
      - `timeframe`
      - `whitelist`

Each rule entry under `rule_results` has the same shape:

- `is_valid` (boolean)
- `rule_name` (string)
- `severity` (string)
- `message` (string)
- `timestamp` (RFC3339 string, may be empty in examples)
- `query_snapshot` (object): Another copy of `structured_query` at the rule evaluation point.

The example payload for 200 OK returned by the docs (`/openapi.json`, shown in Swagger UI/ReDoc) matches the live 200 OK response returned by the server for the canonical demo query:

```json
{
  "structured_query": {
    "log_source": "kube-apiserver",
    "verb": "delete",
    "resource": "customresourcedefinitions",
    "namespace": null,
    "user": null,
    "timeframe": "yesterday",
    "limit": 20,
    "response_status": null,
    "exclude_users": ["system:"],
    "resource_name_pattern": "customer",
    "source_ip": null,
    "group_by": null
  },
  "confidence": 0.7,
  "validation_info": {
    "is_valid": true,
    "rule_name": "comprehensive_safety_validation",
    "severity": "info",
    "message": "Query validation completed successfully",
    "details": {
      "rule_results": {
        "patterns": { "is_valid": true, "rule_name": "forbidden_patterns_validation", "severity": "critical", "message": "Forbidden patterns validation passed", "timestamp": "", "query_snapshot": { /* structured_query copy */ } },
        "required_fields": { "is_valid": true, "rule_name": "required_fields_validation", "severity": "critical", "message": "Required fields validation passed", "timestamp": "", "query_snapshot": { /* structured_query copy */ } },
        "sanitization": { "is_valid": true, "rule_name": "sanitization_validation", "severity": "high", "message": "Input sanitization validation passed", "timestamp": "", "query_snapshot": { /* structured_query copy */ } },
        "timeframe": { "is_valid": true, "rule_name": "timeframe_validation", "severity": "medium", "message": "Timeframe validation passed", "timestamp": "", "query_snapshot": { /* structured_query copy */ } },
        "whitelist": { "is_valid": true, "rule_name": "whitelist_validation", "severity": "critical", "message": "Whitelist validation passed", "timestamp": "", "query_snapshot": { /* structured_query copy */ } }
      },
      "total_rules_applied": 5,
      "validation_timestamp": "2025-01-01T00:00:00Z"
    },
    "timestamp": "2025-01-01T00:00:00Z",
    "query_snapshot": { /* structured_query copy */ }
  }
}
```

Notes:

- Field order may differ between examples and live responses; JSON object order is not semantically significant.
- The OpenAPI schemas for `StructuredQuery`, `ValidationInfo`, `ValidationDetails`, and `ValidationRuleResult` are embedded in `OpenAPIHandler` in `cmd/server/handlers.go`.

### Error Responses

Errors are standardized by `writeErrorResponse` in `cmd/server/handlers.go` and documented as `ErrorResponse` in the OpenAPI spec.

Shape:

- `error` (object)
  - `type` (string)
  - `message` (string)
  - `code` (number)
- `timestamp` (RFC3339 string)

### Demo Mode for Doc/Response Parity

To guarantee that the Swagger/ReDoc 200 OK example is identical to the live 200 OK response for a canonical demo query, the server supports a demo mode implemented in `QueryHandler` via `buildDemoResponse` (both in `cmd/server/handlers.go`).

Environment flags:

- `DEMO_MODE=true`: Enables deterministic example responses for recognized demo inputs.
- `DEMO_FIXED_TIMESTAMPS=true`: Forces fixed example timestamps (`2025-01-01T00:00:00Z`) for exact parity with the docs.

Canonical demo input recognized by `buildDemoResponse`:

- `query`: "Who deleted the customer CRD yesterday?"
- `session_id`: any non-empty string

### How the Docs Are Served

The OpenAPI JSON is served by `OpenAPIHandler` in `cmd/server/handlers.go`. This handler returns a static, embedded JSON string that includes:

- OpenAPI info, servers, tags
- Paths for `/health` and `/query`
- Components schemas for request/response types
- A 200 OK example for `/query` that mirrors `buildDemoResponse` when demo mode is on

Interactive UIs:

- `/docs` serves Swagger UI (via CDN) pointing at `/openapi.json`.
- `/redoc` serves ReDoc (via CDN) pointing at `/openapi.json`.

### How to Build and Run

Preferred local flow (matches your instructions):

```bash
go build -o server ./cmd/server
set -a; source .env; set +a
export DEMO_MODE=true DEMO_FIXED_TIMESTAMPS=true
./server
```

Then open:

- Swagger UI: http://localhost:8080/docs
- ReDoc: http://localhost:8080/redoc
- Spec: http://localhost:8080/openapi.json

### Code Map

- Request/Response types: `pkg/types/query.go`
- HTTP handlers and middleware: `cmd/server/handlers.go`
  - `QueryHandler` – POST `/query`
  - `HealthHandler` – GET `/health`
  - `OpenAPIHandler` – GET `/openapi.json`
  - `DocsHandler` – GET `/docs`
  - `RedocHandler` – GET `/redoc`
  - `buildDemoResponse` – deterministic 200 example when demo mode is enabled
- Server setup and logging: `cmd/server/main.go`


