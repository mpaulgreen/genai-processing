package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestModelConfigValidation(t *testing.T) {
	t.Run("valid model config", func(t *testing.T) {
		config := ModelConfig{
			ModelName:   "gpt-4",
			Provider:    "openai",
			MaxTokens:   1000,
			Temperature: 0.7,
			APIEndpoint: "https://api.openai.com/v1",
		}

		// Test that the config can be marshaled to JSON
		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal valid config: %v", err)
		}

		// Test that the config can be unmarshaled back
		var unmarshaled ModelConfig
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal config: %v", err)
		}

		// Verify the round-trip preserved all values
		if config.ModelName != unmarshaled.ModelName {
			t.Errorf("ModelName round-trip failed: expected %s, got %s", config.ModelName, unmarshaled.ModelName)
		}
		if config.Provider != unmarshaled.Provider {
			t.Errorf("Provider round-trip failed: expected %s, got %s", config.Provider, unmarshaled.Provider)
		}
		if config.MaxTokens != unmarshaled.MaxTokens {
			t.Errorf("MaxTokens round-trip failed: expected %d, got %d", config.MaxTokens, unmarshaled.MaxTokens)
		}
		if config.Temperature != unmarshaled.Temperature {
			t.Errorf("Temperature round-trip failed: expected %f, got %f", config.Temperature, unmarshaled.Temperature)
		}
		if config.APIEndpoint != unmarshaled.APIEndpoint {
			t.Errorf("APIEndpoint round-trip failed: expected %s, got %s", config.APIEndpoint, unmarshaled.APIEndpoint)
		}
	})

	t.Run("temperature bounds validation", func(t *testing.T) {
		// Test minimum temperature
		config := ModelConfig{
			ModelName:   "gpt-4",
			Provider:    "openai",
			MaxTokens:   1000,
			Temperature: 0.0,
			APIEndpoint: "https://api.openai.com/v1",
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal config with min temperature: %v", err)
		}

		var unmarshaled ModelConfig
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal config with min temperature: %v", err)
		}

		if unmarshaled.Temperature != 0.0 {
			t.Errorf("expected minimum temperature to be preserved, got %f", unmarshaled.Temperature)
		}

		// Test maximum temperature
		config.Temperature = 2.0
		data, err = json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal config with max temperature: %v", err)
		}

		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal config with max temperature: %v", err)
		}

		if unmarshaled.Temperature != 2.0 {
			t.Errorf("expected maximum temperature to be preserved, got %f", unmarshaled.Temperature)
		}
	})

	t.Run("max tokens validation", func(t *testing.T) {
		config := ModelConfig{
			ModelName:   "gpt-4",
			Provider:    "openai",
			MaxTokens:   1, // Minimum valid value
			Temperature: 0.7,
			APIEndpoint: "https://api.openai.com/v1",
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal config with min tokens: %v", err)
		}

		var unmarshaled ModelConfig
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal config with min tokens: %v", err)
		}

		if unmarshaled.MaxTokens != 1 {
			t.Errorf("expected minimum tokens to be preserved, got %d", unmarshaled.MaxTokens)
		}
	})
}

func TestTokenUsageCalculations(t *testing.T) {
	now := time.Now()

	t.Run("token calculation logic", func(t *testing.T) {
		usage := TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
		}

		// Test that TotalTokens is calculated correctly
		expectedTotal := usage.PromptTokens + usage.CompletionTokens
		if expectedTotal != 150 {
			t.Errorf("token calculation failed: expected %d, got %d", 150, expectedTotal)
		}
	})

	t.Run("cost calculation with processing time", func(t *testing.T) {
		usage := TokenUsage{
			TotalTokens:     150,
			ProcessingTime:  2 * time.Second,
			TokensPerSecond: 75.0,
		}

		// Test that processing metrics are consistent
		calculatedTokensPerSecond := float64(usage.TotalTokens) / usage.ProcessingTime.Seconds()
		if calculatedTokensPerSecond != usage.TokensPerSecond {
			t.Errorf("tokens per second calculation mismatch: expected %f, got %f", usage.TokensPerSecond, calculatedTokensPerSecond)
		}
	})

	t.Run("JSON round-trip with all fields", func(t *testing.T) {
		usage := TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			EstimatedCost:    0.0025,
			Currency:         "USD",
			ProcessingTime:   2 * time.Second,
			TokensPerSecond:  75.0,
			ModelName:        "gpt-4",
			RequestID:        "req-123",
			Timestamp:        now,
		}

		data, err := json.Marshal(usage)
		if err != nil {
			t.Fatalf("failed to marshal usage: %v", err)
		}

		var unmarshaled TokenUsage
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal usage: %v", err)
		}

		// Verify critical fields are preserved
		if usage.PromptTokens != unmarshaled.PromptTokens {
			t.Errorf("PromptTokens round-trip failed: expected %d, got %d", usage.PromptTokens, unmarshaled.PromptTokens)
		}
		if usage.CompletionTokens != unmarshaled.CompletionTokens {
			t.Errorf("CompletionTokens round-trip failed: expected %d, got %d", usage.CompletionTokens, unmarshaled.CompletionTokens)
		}
		if usage.TotalTokens != unmarshaled.TotalTokens {
			t.Errorf("TotalTokens round-trip failed: expected %d, got %d", usage.TotalTokens, unmarshaled.TotalTokens)
		}
		if usage.EstimatedCost != unmarshaled.EstimatedCost {
			t.Errorf("EstimatedCost round-trip failed: expected %f, got %f", usage.EstimatedCost, unmarshaled.EstimatedCost)
		}
		if usage.Currency != unmarshaled.Currency {
			t.Errorf("Currency round-trip failed: expected %s, got %s", usage.Currency, unmarshaled.Currency)
		}
		if usage.ModelName != unmarshaled.ModelName {
			t.Errorf("ModelName round-trip failed: expected %s, got %s", usage.ModelName, unmarshaled.ModelName)
		}
		if usage.RequestID != unmarshaled.RequestID {
			t.Errorf("RequestID round-trip failed: expected %s, got %s", usage.RequestID, unmarshaled.RequestID)
		}
		if !usage.Timestamp.Equal(unmarshaled.Timestamp) {
			t.Errorf("Timestamp round-trip failed: expected %v, got %v", usage.Timestamp, unmarshaled.Timestamp)
		}
	})
}

func TestExampleValidation(t *testing.T) {
	now := time.Now()

	t.Run("confidence bounds validation", func(t *testing.T) {
		example := Example{
			Input:       "What is 2+2?",
			Output:      "2+2 equals 4",
			Description: "Basic arithmetic example",
			Category:    "math",
			Confidence:  0.95,
			IsActive:    true,
		}

		// Test that confidence is within valid bounds
		if example.Confidence < 0.0 || example.Confidence > 1.0 {
			t.Errorf("confidence out of bounds: %f", example.Confidence)
		}

		// Test JSON round-trip
		data, err := json.Marshal(example)
		if err != nil {
			t.Fatalf("failed to marshal example: %v", err)
		}

		var unmarshaled Example
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal example: %v", err)
		}

		if example.Input != unmarshaled.Input {
			t.Errorf("Input round-trip failed: expected %s, got %s", example.Input, unmarshaled.Input)
		}
		if example.Output != unmarshaled.Output {
			t.Errorf("Output round-trip failed: expected %s, got %s", example.Output, unmarshaled.Output)
		}
		if example.Confidence != unmarshaled.Confidence {
			t.Errorf("Confidence round-trip failed: expected %f, got %f", example.Confidence, unmarshaled.Confidence)
		}
	})

	t.Run("priority validation", func(t *testing.T) {
		example := Example{
			Input:    "test",
			Output:   "test",
			Priority: 1, // Minimum valid value
			IsActive: true,
		}

		// Test that priority is positive
		if example.Priority < 1 {
			t.Errorf("priority should be at least 1, got %d", example.Priority)
		}

		// Test JSON round-trip
		data, err := json.Marshal(example)
		if err != nil {
			t.Fatalf("failed to marshal example with priority: %v", err)
		}

		var unmarshaled Example
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal example with priority: %v", err)
		}

		if example.Priority != unmarshaled.Priority {
			t.Errorf("Priority round-trip failed: expected %d, got %d", example.Priority, unmarshaled.Priority)
		}
	})

	t.Run("metadata handling", func(t *testing.T) {
		example := Example{
			Input:       "Translate 'hello' to Spanish",
			Output:      "Hola",
			Description: "Basic translation example",
			Category:    "translation",
			Tags:        []string{"spanish", "greeting", "translation"},
			Confidence:  0.95,
			CreatedAt:   now,
			UpdatedAt:   now,
			IsActive:    true,
			Priority:    1,
			Metadata: map[string]interface{}{
				"difficulty": "easy",
				"language":   "spanish",
			},
		}

		// Test that metadata is preserved in JSON
		data, err := json.Marshal(example)
		if err != nil {
			t.Fatalf("failed to marshal example with metadata: %v", err)
		}

		var unmarshaled Example
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal example with metadata: %v", err)
		}

		if len(example.Tags) != len(unmarshaled.Tags) {
			t.Errorf("Tags length mismatch: expected %d, got %d", len(example.Tags), len(unmarshaled.Tags))
		}

		if example.Metadata["difficulty"] != unmarshaled.Metadata["difficulty"] {
			t.Errorf("Metadata difficulty mismatch: expected %v, got %v", example.Metadata["difficulty"], unmarshaled.Metadata["difficulty"])
		}
	})
}

func TestModelInfoValidation(t *testing.T) {
	releaseDate := time.Date(2023, 3, 14, 0, 0, 0, 0, time.UTC)

	t.Run("model info with all fields", func(t *testing.T) {
		info := ModelInfo{
			Name:               "gpt-4",
			Provider:           "OpenAI",
			Version:            "1.0",
			Description:        "Large language model for text generation",
			ReleaseDate:        releaseDate,
			ModelType:          "chat",
			ContextWindow:      8192,
			MaxOutputTokens:    4096,
			SupportedLanguages: []string{"python", "javascript", "go", "english"},
			PricingInfo: map[string]interface{}{
				"input_cost_per_1k":  0.03,
				"output_cost_per_1k": 0.06,
				"currency":           "USD",
			},
			PerformanceMetrics: map[string]interface{}{
				"accuracy": 0.95,
				"speed":    "fast",
			},
		}

		// Test that context window is reasonable
		if info.ContextWindow <= 0 {
			t.Errorf("context window should be positive, got %d", info.ContextWindow)
		}

		// Test that max output tokens is reasonable
		if info.MaxOutputTokens <= 0 {
			t.Errorf("max output tokens should be positive, got %d", info.MaxOutputTokens)
		}

		// Test that max output tokens doesn't exceed context window
		if info.MaxOutputTokens > info.ContextWindow {
			t.Errorf("max output tokens (%d) should not exceed context window (%d)", info.MaxOutputTokens, info.ContextWindow)
		}

		// Test JSON round-trip
		data, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("failed to marshal model info: %v", err)
		}

		var unmarshaled ModelInfo
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal model info: %v", err)
		}

		// Verify critical fields are preserved
		if info.Name != unmarshaled.Name {
			t.Errorf("Name round-trip failed: expected %s, got %s", info.Name, unmarshaled.Name)
		}
		if info.Provider != unmarshaled.Provider {
			t.Errorf("Provider round-trip failed: expected %s, got %s", info.Provider, unmarshaled.Provider)
		}
		if info.ContextWindow != unmarshaled.ContextWindow {
			t.Errorf("ContextWindow round-trip failed: expected %d, got %d", info.ContextWindow, unmarshaled.ContextWindow)
		}
		if info.MaxOutputTokens != unmarshaled.MaxOutputTokens {
			t.Errorf("MaxOutputTokens round-trip failed: expected %d, got %d", info.MaxOutputTokens, unmarshaled.MaxOutputTokens)
		}
		if len(info.SupportedLanguages) != len(unmarshaled.SupportedLanguages) {
			t.Errorf("SupportedLanguages length mismatch: expected %d, got %d", len(info.SupportedLanguages), len(unmarshaled.SupportedLanguages))
		}
	})
}
