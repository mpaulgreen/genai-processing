package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
