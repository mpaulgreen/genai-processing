package processor

import (
	"context"
	"testing"
	"time"

	"genai-processing/internal/parser/extractors"
	"genai-processing/internal/parser/recovery"
	"genai-processing/pkg/types"
)

// TestRetryParserIntegration tests the integration of RetryParser with the processor
func TestRetryParserIntegration(t *testing.T) {
	// Create a simple RetryParser configuration
	retryConfig := &recovery.RetryConfig{
		MaxRetries:          2,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.7,
		EnableReprompting:   false, // Disable for testing
	}

	// Create RetryParser with nil dependencies for testing
	retryParser := recovery.NewRetryParser(retryConfig, nil, nil)

	// Register parsers
	claudeExtractor := extractors.NewClaudeExtractor()
	openaiExtractor := extractors.NewOpenAIExtractor()

	retryParser.RegisterParser(recovery.StrategySpecific, claudeExtractor)
	retryParser.RegisterParser(recovery.StrategyGeneric, openaiExtractor)

	// Test with valid JSON response
	validRawResponse := &types.RawResponse{
		Content: `{"log_source": "kube-apiserver", "verb": "get", "resource": "pods", "limit": 20}`,
	}

	ctx := context.Background()
	result, err := retryParser.ParseWithRetry(ctx, validRawResponse, "claude", "test query", "test-session")

	if err != nil {
		t.Fatalf("Expected no error for valid JSON, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.LogSource != "kube-apiserver" {
		t.Errorf("Expected LogSource 'kube-apiserver', got '%s'", result.LogSource)
	}

	// Test statistics
	stats := retryParser.GetRetryStatistics()
	if stats == nil {
		t.Fatal("Expected statistics, got nil")
	}

	if stats["max_retries"] != 2 {
		t.Errorf("Expected max_retries 2, got %v", stats["max_retries"])
	}

	if stats["confidence_threshold"] != 0.7 {
		t.Errorf("Expected confidence_threshold 0.7, got %v", stats["confidence_threshold"])
	}
}

// TestRetryParserWithProcessorConstructor tests that the processor constructor can create RetryParser
func TestRetryParserWithProcessorConstructor(t *testing.T) {
	// This test verifies that the NewGenAIProcessor constructor can create a processor
	// with RetryParser integration without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewGenAIProcessor panicked: %v", r)
		}
	}()

	// Create processor - this should work with RetryParser integration
	processor := NewGenAIProcessor()

	if processor == nil {
		t.Fatal("Expected processor, got nil")
	}

	if processor.RetryParser == nil {
		t.Fatal("Expected retryParser to be initialized, got nil")
	}

	// Verify RetryParser configuration
	stats := processor.RetryParser.GetRetryStatistics()
	if stats == nil {
		t.Fatal("Expected RetryParser statistics, got nil")
	}

	// Check that parsers are registered
	strategies := stats["registered_strategies"].([]string)
	if len(strategies) == 0 {
		t.Error("Expected registered strategies, got empty list")
	}

	// Verify configuration values
	if stats["max_retries"] != 3 {
		t.Errorf("Expected max_retries 3, got %v", stats["max_retries"])
	}

	if stats["confidence_threshold"] != 0.7 {
		t.Errorf("Expected confidence_threshold 0.7, got %v", stats["confidence_threshold"])
	}

	if stats["enable_reprompting"] != true {
		t.Errorf("Expected enable_reprompting true, got %v", stats["enable_reprompting"])
	}
}

// TestRetryParserErrorHandling tests error handling with RetryParser
func TestRetryParserErrorHandling(t *testing.T) {
	// Create RetryParser with minimal configuration
	retryConfig := &recovery.RetryConfig{
		MaxRetries:          1,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.8,
		EnableReprompting:   false,
	}

	retryParser := recovery.NewRetryParser(retryConfig, nil, nil)

	// Test with invalid JSON response
	invalidRawResponse := &types.RawResponse{
		Content: `invalid json content`,
	}

	ctx := context.Background()
	result, err := retryParser.ParseWithRetry(ctx, invalidRawResponse, "claude", "test query", "test-session")

	// Should return error since no parsers are registered
	if err == nil {
		t.Fatal("Expected error for invalid JSON with no parsers, got nil")
	}

	if result != nil {
		t.Fatal("Expected nil result for invalid JSON, got result")
	}

	// Test error classification
	if !retryParser.IsRecoverableError(err) {
		t.Error("Expected error to be recoverable")
	}
}

// TestRetryParserFallbackQuery tests fallback query creation
func TestRetryParserFallbackQuery(t *testing.T) {
	retryParser := recovery.NewRetryParser(nil, nil, nil)

	// Test with different content types
	testCases := []struct {
		content        string
		expectedSource string
		expectedTime   string
	}{
		{
			content:        "oauth server logs",
			expectedSource: "oauth-server",
			expectedTime:   "",
		},
		{
			content:        "logs from today",
			expectedSource: "kube-apiserver",
			expectedTime:   "today",
		},
		{
			content:        "yesterday's logs",
			expectedSource: "kube-apiserver",
			expectedTime:   "yesterday",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.content, func(t *testing.T) {
			raw := &types.RawResponse{Content: tc.content}
			result := retryParser.CreateFallbackQuery(raw, "test-model")

			if result.LogSource != tc.expectedSource {
				t.Errorf("Expected LogSource %s, got %s", tc.expectedSource, result.LogSource)
			}

			if result.Timeframe != tc.expectedTime {
				t.Errorf("Expected Timeframe %s, got %s", tc.expectedTime, result.Timeframe)
			}

			if result.Limit != 20 {
				t.Errorf("Expected Limit 20, got %d", result.Limit)
			}
		})
	}
}
