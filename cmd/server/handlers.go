package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"genai-processing/internal/processor"
	"genai-processing/pkg/types"
)

// QueryHandler handles POST /query requests for natural language audit query processing
func QueryHandler(genaiProcessor *processor.GenAIProcessor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Set response headers
		w.Header().Set("Content-Type", "application/json")

		// Log incoming request
		log.Printf("[QueryHandler] Received %s request from %s", r.Method, r.RemoteAddr)

		// Validate HTTP method
		if r.Method != http.MethodPost {
			log.Printf("[QueryHandler] Invalid method: %s", r.Method)
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only POST method is supported")
			return
		}

		// Parse request body
		var req types.ProcessingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[QueryHandler] Failed to decode request body: %v", err)
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request format", "Failed to parse JSON request body")
			return
		}

		// Basic input validation
		if err := validateProcessingRequest(&req); err != nil {
			log.Printf("[QueryHandler] Request validation failed: %v", err)
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request", err.Error())
			return
		}

		// Log request details
		log.Printf("[QueryHandler] Processing query: %q, SessionID: %s", req.Query, req.SessionID)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Extract user ID from Authorization header (e.g., "Bearer <token>")
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if userID := extractUserIDFromAuthHeader(authHeader); userID != "" {
				ctx = context.WithValue(ctx, types.ContextKeyUserID, userID)
			}
		}

		// DEMO mode: return a deterministic example response for known inputs
		if os.Getenv("DEMO_MODE") == "true" {
			if demoResp, ok := buildDemoResponse(&req); ok {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(demoResp)
				return
			}
		}

		// Process the query using GenAIProcessor
		response, err := genaiProcessor.ProcessQuery(ctx, &req)
		if err != nil {
			log.Printf("[QueryHandler] Processing failed: %v", err)
			writeErrorResponse(w, http.StatusInternalServerError, "Processing error", "Failed to process query")
			return
		}

		// Check if processing resulted in an error response
		if response.Error != "" {
			log.Printf("[QueryHandler] Processing returned error: %s", response.Error)
			writeErrorResponse(w, http.StatusBadRequest, "Processing error", response.Error)
			return
		}

		// Log successful processing
		processingTime := time.Since(startTime)
		log.Printf("[QueryHandler] Query processed successfully in %v", processingTime)

		// Write successful response
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("[QueryHandler] Failed to encode response: %v", err)
			// Response already started, can't change status code
			return
		}

		// Log response details
		log.Printf("[QueryHandler] Response sent successfully, confidence: %.2f", response.Confidence)
	}
}

// buildDemoResponse returns a deterministic 200 OK response that exactly matches the OpenAPI example
// when DEMO_MODE is enabled. For unrecognized queries, returns (nil, false).
func buildDemoResponse(req *types.ProcessingRequest) (map[string]interface{}, bool) {
	if req == nil {
		return nil, false
	}
	// Recognized canonical demo query
	if strings.EqualFold(strings.TrimSpace(req.Query), "Who deleted the customer CRD yesterday?") {
		// Choose timestamps: fixed for exact match with docs if DEMO_FIXED_TIMESTAMPS=true; else use now
		ts := "2025-01-01T00:00:00Z"
		if os.Getenv("DEMO_FIXED_TIMESTAMPS") != "true" {
			ts = time.Now().UTC().Format(time.RFC3339)
		}

		structured := map[string]interface{}{
			"log_source":            "kube-apiserver",
			"verb":                  "delete",
			"resource":              "customresourcedefinitions",
			"namespace":             nil,
			"user":                  nil,
			"timeframe":             "yesterday",
			"limit":                 20,
			"response_status":       nil,
			"exclude_users":         []string{"system:"},
			"resource_name_pattern": "customer",
			"source_ip":             nil,
			"group_by":              nil,
		}

		// Build rule results entries helper
		mkRule := func(name, severity, message string) map[string]interface{} {
			return map[string]interface{}{
				"is_valid":       true,
				"rule_name":      name,
				"severity":       severity,
				"message":        message,
				"timestamp":      "",
				"query_snapshot": structured,
			}
		}

		validation := map[string]interface{}{
			"is_valid":  true,
			"rule_name": "comprehensive_safety_validation",
			"severity":  "info",
			"message":   "Query validation completed successfully",
			"details": map[string]interface{}{
				"rule_results": map[string]interface{}{
					"patterns":        mkRule("forbidden_patterns_validation", "critical", "Forbidden patterns validation passed"),
					"required_fields": mkRule("required_fields_validation", "critical", "Required fields validation passed"),
					"sanitization":    mkRule("sanitization_validation", "high", "Input sanitization validation passed"),
					"timeframe":       mkRule("timeframe_validation", "medium", "Timeframe validation passed"),
					"whitelist":       mkRule("whitelist_validation", "critical", "Whitelist validation passed"),
				},
				"total_rules_applied":  5,
				"validation_timestamp": ts,
			},
			"timestamp":      ts,
			"query_snapshot": structured,
		}

		return map[string]interface{}{
			"structured_query": structured,
			"confidence":       0.7,
			"validation_info":  validation,
		}, true
	}
	return nil, false
}

// extractUserIDFromAuthHeader is a placeholder for extracting user identity from Authorization header.
// In production, replace with proper JWT parsing and validation. For now, supports a simple scheme:
// Authorization: Bearer user:<user-id>
func extractUserIDFromAuthHeader(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	token := strings.TrimSpace(parts[1])
	// Very basic demo format: user:<id>
	if strings.HasPrefix(token, "user:") {
		return strings.TrimSpace(strings.TrimPrefix(token, "user:"))
	}
	return ""
}

// HealthHandler handles GET /health requests for health checks
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set response headers
		w.Header().Set("Content-Type", "application/json")

		// Log health check request
		log.Printf("[HealthHandler] Health check request from %s", r.RemoteAddr)

		// Validate HTTP method
		if r.Method != http.MethodGet {
			log.Printf("[HealthHandler] Invalid method: %s", r.Method)
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET method is supported")
			return
		}

		// Create health response
		healthResponse := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"service":   "genai-audit-query-processor",
			"version":   "1.0.0",
		}

		// Write response
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
			log.Printf("[HealthHandler] Failed to encode health response: %v", err)
			return
		}

		log.Printf("[HealthHandler] Health check completed successfully")
	}
}

// validateProcessingRequest performs basic validation on the processing request
func validateProcessingRequest(req *types.ProcessingRequest) error {
	// Check if query is provided and not empty
	if req.Query == "" {
		return fmt.Errorf("query is required and cannot be empty")
	}

	// Check query length (reasonable limits)
	if len(req.Query) > 1000 {
		return fmt.Errorf("query too long, maximum 1000 characters allowed")
	}

	// Check if session ID is provided
	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	// Validate session ID format (basic check)
	if len(req.SessionID) > 100 {
		return fmt.Errorf("session_id too long, maximum 100 characters allowed")
	}

	// Validate model type if provided
	if req.ModelType != "" {
		// Add model type validation if needed
		// For now, just check length
		if len(req.ModelType) > 50 {
			return fmt.Errorf("model_type too long, maximum 50 characters allowed")
		}
	}

	return nil
}

// writeErrorResponse writes a standardized error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorType, message string) {
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"type":    errorType,
			"message": message,
			"code":    statusCode,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// setupRoutes configures the HTTP routes for the server
func setupRoutes(genaiProcessor *processor.GenAIProcessor) *http.ServeMux {
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/query", QueryHandler(genaiProcessor))
	mux.HandleFunc("/health", HealthHandler())
	mux.HandleFunc("/openapi.json", OpenAPIHandler())
	mux.HandleFunc("/docs", DocsHandler())
	mux.HandleFunc("/redoc", RedocHandler())

	// Add logging middleware
	return mux
}

// loggingMiddleware adds request logging to all handlers
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("[HTTP] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Call next handler
		next.ServeHTTP(w, r)

		// Log response time
		duration := time.Since(start)
		log.Printf("[HTTP] %s %s completed in %v", r.Method, r.URL.Path, duration)
	})
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// OpenAPIHandler serves the OpenAPI specification for the API
func OpenAPIHandler() http.HandlerFunc {
	// Expanded OpenAPI 3.0 spec with schemas, examples, and headers
	const spec = `{
  "openapi": "3.0.1",
  "info": {
    "title": "GenAI Audit Query API",
    "version": "1.0.0",
    "description": "API for processing natural language audit queries into structured queries."
  },
  "servers": [
    { "url": "http://localhost:8080", "description": "Local server" }
  ],
  "tags": [
    { "name": "Health", "description": "Service health" },
    { "name": "Query", "description": "Audit query processing" }
  ],
  "paths": {
    "/health": {
      "get": {
        "tags": ["Health"],
        "operationId": "getHealth",
        "summary": "Health check",
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "status": { "type": "string" },
                    "timestamp": { "type": "string", "format": "date-time" },
                    "service": { "type": "string" },
                    "version": { "type": "string" }
                  }
                },
                "examples": {
                  "ok": {
                    "value": { "status": "healthy", "timestamp": "2025-01-01T00:00:00Z", "service": "genai-audit-query-processor", "version": "1.0.0" }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/query": {
      "post": {
        "tags": ["Query"],
        "operationId": "processQuery",
        "summary": "Process a natural language audit query",
        "description": "Optional Authorization header supports demo format: Bearer user:<user-id>.",
        "parameters": [
          {
            "name": "Authorization",
            "in": "header",
            "required": false,
            "schema": { "type": "string" },
            "description": "Bearer token. Demo format supported: Bearer user:<user-id>."
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/ProcessingRequest" },
              "examples": {
                "example1": {
                  "value": { "query": "Who deleted the customer CRD yesterday?", "session_id": "test" }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Query processed successfully",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ProcessingResponse" },
                "examples": {
                  "success": {
                    "value": {
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
                            "patterns": {
                              "is_valid": true,
                              "rule_name": "forbidden_patterns_validation",
                              "severity": "critical",
                              "message": "Forbidden patterns validation passed",
                              "timestamp": "",
                              "query_snapshot": {
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
                              }
                            },
                            "required_fields": {
                              "is_valid": true,
                              "rule_name": "required_fields_validation",
                              "severity": "critical",
                              "message": "Required fields validation passed",
                              "timestamp": "",
                              "query_snapshot": {
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
                              }
                            },
                            "sanitization": {
                              "is_valid": true,
                              "rule_name": "sanitization_validation",
                              "severity": "high",
                              "message": "Input sanitization validation passed",
                              "timestamp": "",
                              "query_snapshot": {
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
                              }
                            },
                            "timeframe": {
                              "is_valid": true,
                              "rule_name": "timeframe_validation",
                              "severity": "medium",
                              "message": "Timeframe validation passed",
                              "timestamp": "",
                              "query_snapshot": {
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
                              }
                            },
                            "whitelist": {
                              "is_valid": true,
                              "rule_name": "whitelist_validation",
                              "severity": "critical",
                              "message": "Whitelist validation passed",
                              "timestamp": "",
                              "query_snapshot": {
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
                              }
                            }
                          },
                          "total_rules_applied": 5,
                          "validation_timestamp": "2025-01-01T00:00:00Z"
                        },
                        "timestamp": "2025-01-01T00:00:00Z",
                        "query_snapshot": {
                          "log_source": "kube-apiserver",
                          "verb": "delete",
                          "resource": "customresourcedefinitions",
                          "timeframe": "yesterday",
                          "limit": 20,
                          "exclude_users": ["system:"],
                          "resource_name_pattern": "customer"
                        }
                      }
                    }
                  }
                }
              }
            }
          },
          "400": {
            "description": "Invalid request",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ErrorResponse" },
                "examples": {
                  "badRequest": {
                    "value": { "error": { "type": "Invalid request", "message": "query is required and cannot be empty", "code": 400 }, "timestamp": "2025-01-01T00:00:00Z" }
                  }
                }
              }
            }
          },
          "405": { "description": "Method not allowed" },
          "500": {
            "description": "Processing error",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ErrorResponse" }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
        "description": "Demo only: Bearer user:<user-id> accepted; no real auth/verification."
      }
    },
    "schemas": {
      "ProcessingRequest": {
        "type": "object",
        "required": ["query", "session_id"],
        "properties": {
          "query": { "type": "string", "description": "Natural language query (<= 1000 chars)" },
          "session_id": { "type": "string", "description": "Session identifier (<= 100 chars)" },
          "model_type": { "type": "string", "description": "Preferred model type", "nullable": true }
        }
      },
      "StructuredQuery": {
        "type": "object",
        "properties": {
          "log_source": { "type": "string", "nullable": true, "description": "Source of audit logs" },
          "verb": { "type": "string", "nullable": true, "description": "Operation verb", "enum": ["get","list","create","update","patch","delete","watch"] },
          "resource": { "type": "string", "nullable": true },
          "namespace": { "type": "string", "nullable": true },
          "user": { "type": "string", "nullable": true },
          "timeframe": { "type": "string", "nullable": true },
          "limit": { "type": "integer", "nullable": true, "default": 20, "minimum": 1 },
          "response_status": { "type": "string", "nullable": true },
          "exclude_users": { "type": "array", "items": { "type": "string" }, "nullable": true, "description": "Users to exclude (prefix-friendly)" },
          "resource_name_pattern": { "type": "string", "nullable": true, "description": "Substring/pattern of resource name" },
          "source_ip": { "type": "string", "nullable": true },
          "group_by": { "type": "string", "nullable": true }
        }
      },
      "ValidationRuleResult": {
        "type": "object",
        "properties": {
          "is_valid": { "type": "boolean" },
          "rule_name": { "type": "string" },
          "severity": { "type": "string" },
          "message": { "type": "string" },
          "timestamp": { "type": "string", "format": "date-time" },
          "query_snapshot": { "$ref": "#/components/schemas/StructuredQuery" }
        }
      },
      "ValidationDetails": {
        "type": "object",
        "properties": {
          "rule_results": {
            "type": "object",
            "additionalProperties": { "$ref": "#/components/schemas/ValidationRuleResult" }
          },
          "total_rules_applied": { "type": "integer" },
          "validation_timestamp": { "type": "string", "format": "date-time" }
        }
      },
      "ValidationInfo": {
        "type": "object",
        "properties": {
          "is_valid": { "type": "boolean" },
          "rule_name": { "type": "string" },
          "severity": { "type": "string" },
          "message": { "type": "string" },
          "details": { "$ref": "#/components/schemas/ValidationDetails" },
          "timestamp": { "type": "string", "format": "date-time" },
          "query_snapshot": { "$ref": "#/components/schemas/StructuredQuery" }
        }
      },
      "ProcessingResponse": {
        "type": "object",
        "properties": {
          "structured_query": { "$ref": "#/components/schemas/StructuredQuery" },
          "confidence": { "type": "number", "format": "float" },
          "validation_info": { "$ref": "#/components/schemas/ValidationInfo" },
          "error": { "type": "string" },
          "timestamp": { "type": "string", "format": "date-time" },
          "query_snapshot": { "$ref": "#/components/schemas/StructuredQuery" }
        }
      },
      "ErrorResponse": {
        "type": "object",
        "properties": {
          "error": {
            "type": "object",
            "properties": {
              "type": { "type": "string" },
              "message": { "type": "string" },
              "code": { "type": "integer" }
            }
          },
          "timestamp": { "type": "string", "format": "date-time" }
        }
      }
    }
  }
}`

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(spec))
	}
}

// DocsHandler serves a Swagger UI that points to the OpenAPI spec
func DocsHandler() http.HandlerFunc {
	const html = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>GenAI Audit Query API Docs</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css" />
    <style>body { margin: 0; padding: 0; } #swagger-ui { width: 100%; }</style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js" crossorigin></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: '/openapi.json',
        dom_id: '#swagger-ui',
        presets: [SwaggerUIBundle.presets.apis],
        layout: 'BaseLayout'
      });
    </script>
  </body>
</html>`

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}
}

// RedocHandler serves an alternative ReDoc documentation page
func RedocHandler() http.HandlerFunc {
	const html = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>GenAI Audit Query API - ReDoc</title>
    <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Nunito:300,400,700|Roboto:300,400,700" />
    <style> body { margin: 0; padding: 0; } </style>
  </head>
  <body>
    <redoc spec-url='/openapi.json'></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
  </body>
</html>`
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}
}
