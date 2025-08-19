package types

import (
	"encoding/json"
	"testing"
)

func TestNewStringOrArray(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "string value",
			input:    "test",
			expected: "test",
		},
		{
			name:     "string array value",
			input:    []string{"test1", "test2"},
			expected: []string{"test1", "test2"},
		},
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewStringOrArray(tt.input)
			if tt.expected == nil {
				if result.GetValue() != nil {
					t.Errorf("NewStringOrArray() = %v, want nil", result.GetValue())
				}
			} else if str, ok := tt.expected.(string); ok {
				if result.GetString() != str {
					t.Errorf("NewStringOrArray() = %v, want %v", result.GetString(), str)
				}
			} else if arr, ok := tt.expected.([]string); ok {
				resultArr := result.GetArray()
				if len(resultArr) != len(arr) {
					t.Errorf("NewStringOrArray() array length = %v, want %v", len(resultArr), len(arr))
				} else {
					for i, v := range resultArr {
						if v != arr[i] {
							t.Errorf("NewStringOrArray() array[%d] = %v, want %v", i, v, arr[i])
						}
					}
				}
			}
		})
	}
}

func TestStringOrArray_IsString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "string value",
			input:    "test",
			expected: true,
		},
		{
			name:     "string array value",
			input:    []string{"test1", "test2"},
			expected: false,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &StringOrArray{value: tt.input}
			if result := sa.IsString(); result != tt.expected {
				t.Errorf("IsString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStringOrArray_IsArray(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "string value",
			input:    "test",
			expected: false,
		},
		{
			name:     "string array value",
			input:    []string{"test1", "test2"},
			expected: true,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &StringOrArray{value: tt.input}
			if result := sa.IsArray(); result != tt.expected {
				t.Errorf("IsArray() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStringOrArray_GetString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string value",
			input:    "test",
			expected: "test",
		},
		{
			name:     "string array value",
			input:    []string{"test1", "test2"},
			expected: "",
		},
		{
			name:     "nil value",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &StringOrArray{value: tt.input}
			if result := sa.GetString(); result != tt.expected {
				t.Errorf("GetString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStringOrArray_GetArray(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "string value",
			input:    "test",
			expected: nil,
		},
		{
			name:     "string array value",
			input:    []string{"test1", "test2"},
			expected: []string{"test1", "test2"},
		},
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &StringOrArray{value: tt.input}
			result := sa.GetArray()
			if len(result) != len(tt.expected) {
				t.Errorf("GetArray() length = %v, want %v", len(result), len(tt.expected))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("GetArray()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestStringOrArray_GetValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "string value",
			input:    "test",
			expected: "test",
		},
		{
			name:     "string array value",
			input:    []string{"test1", "test2"},
			expected: []string{"test1", "test2"},
		},
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &StringOrArray{value: tt.input}
			result := sa.GetValue()
			if tt.expected == nil {
				if result != nil {
					t.Errorf("GetValue() = %v, want nil", result)
				}
			} else if str, ok := tt.expected.(string); ok {
				if resultStr, ok := result.(string); !ok || resultStr != str {
					t.Errorf("GetValue() = %v, want %v", result, str)
				}
			} else if arr, ok := tt.expected.([]string); ok {
				if resultArr, ok := result.([]string); !ok || len(resultArr) != len(arr) {
					t.Errorf("GetValue() = %v, want %v", result, arr)
				} else {
					for i, v := range resultArr {
						if v != arr[i] {
							t.Errorf("GetValue() array[%d] = %v, want %v", i, v, arr[i])
						}
					}
				}
			}
		})
	}
}

func TestStringOrArray_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "string value",
			input:    "test",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "string array value",
			input:    []string{"test1", "test2"},
			expected: false,
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: true,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: true,
		},
		{
			name:     "other type",
			input:    123,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &StringOrArray{value: tt.input}
			if result := sa.IsEmpty(); result != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStringOrArray_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string value",
			input:    "test",
			expected: `"test"`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "string array value",
			input:    []string{"test1", "test2"},
			expected: `["test1","test2"]`,
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: `[]`,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &StringOrArray{value: tt.input}
			result, err := json.Marshal(sa)
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("MarshalJSON() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

func TestStringOrArray_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "string value",
			input:    `"test"`,
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    `""`,
			expected: "",
			wantErr:  false,
		},
		{
			name:     "string array value",
			input:    `["test1","test2"]`,
			expected: []string{"test1", "test2"},
			wantErr:  false,
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "nil value",
			input:    `null`,
			expected: "",
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			input:    `invalid`,
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "number value",
			input:    `123`,
			expected: float64(123),
			wantErr:  false,
		},
		{
			name:     "boolean value",
			input:    `true`,
			expected: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &StringOrArray{}
			err := sa.UnmarshalJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.expected == nil {
					if sa.GetValue() != nil {
						t.Errorf("UnmarshalJSON() = %v, want nil", sa.GetValue())
					}
				} else if str, ok := tt.expected.(string); ok {
					if sa.GetString() != str {
						t.Errorf("UnmarshalJSON() = %v, want %v", sa.GetString(), str)
					}
				} else if arr, ok := tt.expected.([]string); ok {
					resultArr := sa.GetArray()
					if len(resultArr) != len(arr) {
						t.Errorf("UnmarshalJSON() array length = %v, want %v", len(resultArr), len(arr))
					} else {
						for i, v := range resultArr {
							if v != arr[i] {
								t.Errorf("UnmarshalJSON() array[%d] = %v, want %v", i, v, arr[i])
							}
						}
					}
				} else {
					// For other types like numbers, booleans
					if sa.GetValue() != tt.expected {
						t.Errorf("UnmarshalJSON() = %v, want %v", sa.GetValue(), tt.expected)
					}
				}
			}
		})
	}
}

func TestStringOrArray_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "string value",
			input: "test",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "string array value",
			input: []string{"test1", "test2"},
		},
		{
			name:  "empty array",
			input: []string{},
		},
		{
			name:  "nil value",
			input: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &StringOrArray{value: tt.input}

			// Marshal
			data, err := json.Marshal(original)
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			// Unmarshal
			var result StringOrArray
			err = json.Unmarshal(data, &result)
			if err != nil {
				t.Errorf("UnmarshalJSON() error = %v", err)
				return
			}

			// Compare - handle special case where nil becomes empty string
			if tt.input == nil {
				// When nil is marshaled and unmarshaled, it becomes an empty string
				if !result.IsString() || result.GetString() != "" {
					t.Errorf("Round trip failed for nil: expected empty string, got %v", result.GetValue())
				}
			} else if original.IsString() {
				if !result.IsString() || original.GetString() != result.GetString() {
					t.Errorf("Round trip failed: GetString() mismatch: original = %v, result = %v", original.GetString(), result.GetString())
				}
			} else if original.IsArray() {
				if !result.IsArray() {
					t.Errorf("Round trip failed: expected array, got %v", result.GetValue())
				} else {
					origArr := original.GetArray()
					resultArr := result.GetArray()
					if len(origArr) != len(resultArr) {
						t.Errorf("Round trip failed: array length mismatch: original = %d, result = %d", len(origArr), len(resultArr))
					} else {
						for i, v := range origArr {
							if v != resultArr[i] {
								t.Errorf("Round trip failed: array[%d] mismatch: original = %v, result = %v", i, v, resultArr[i])
							}
						}
					}
				}
			}
		})
	}
}

func TestStringOrArray_EdgeCases(t *testing.T) {
	t.Run("unicode string", func(t *testing.T) {
		sa := NewStringOrArray("测试")
		if !sa.IsString() {
			t.Error("Expected IsString() to return true for unicode string")
		}
		if sa.GetString() != "测试" {
			t.Errorf("Expected GetString() to return '测试', got '%s'", sa.GetString())
		}
	})

	t.Run("array with unicode strings", func(t *testing.T) {
		sa := NewStringOrArray([]string{"测试1", "test2", "测试3"})
		if !sa.IsArray() {
			t.Error("Expected IsArray() to return true for array with unicode strings")
		}
		expected := []string{"测试1", "test2", "测试3"}
		result := sa.GetArray()
		if len(result) != len(expected) {
			t.Errorf("Expected array length %d, got %d", len(expected), len(result))
		}
		for i, v := range result {
			if v != expected[i] {
				t.Errorf("Expected array[%d] = '%s', got '%s'", i, expected[i], v)
			}
		}
	})

	t.Run("large array", func(t *testing.T) {
		largeArray := make([]string, 1000)
		for i := range largeArray {
			largeArray[i] = "item"
		}
		sa := NewStringOrArray(largeArray)
		if !sa.IsArray() {
			t.Error("Expected IsArray() to return true for large array")
		}
		if len(sa.GetArray()) != 1000 {
			t.Errorf("Expected array length 1000, got %d", len(sa.GetArray()))
		}
	})

	t.Run("special characters in string", func(t *testing.T) {
		specialStr := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		sa := NewStringOrArray(specialStr)
		if sa.GetString() != specialStr {
			t.Errorf("Expected GetString() to return special characters, got '%s'", sa.GetString())
		}
	})
}

// Benchmarks

func BenchmarkStringOrArray_MarshalJSON_String(b *testing.B) {
	sa := NewStringOrArray("test string")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(sa)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringOrArray_MarshalJSON_Array(b *testing.B) {
	sa := NewStringOrArray([]string{"test1", "test2", "test3", "test4", "test5"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(sa)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringOrArray_UnmarshalJSON_String(b *testing.B) {
	data := []byte(`"test string"`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sa StringOrArray
		err := json.Unmarshal(data, &sa)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringOrArray_UnmarshalJSON_Array(b *testing.B) {
	data := []byte(`["test1","test2","test3","test4","test5"]`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sa StringOrArray
		err := json.Unmarshal(data, &sa)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringOrArray_IsString(b *testing.B) {
	sa := NewStringOrArray("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sa.IsString()
	}
}

func BenchmarkStringOrArray_IsArray(b *testing.B) {
	sa := NewStringOrArray([]string{"test1", "test2"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sa.IsArray()
	}
}

func BenchmarkStringOrArray_GetString(b *testing.B) {
	sa := NewStringOrArray("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sa.GetString()
	}
}

func BenchmarkStringOrArray_GetArray(b *testing.B) {
	sa := NewStringOrArray([]string{"test1", "test2"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sa.GetArray()
	}
}

func BenchmarkStringOrArray_IsEmpty(b *testing.B) {
	sa := NewStringOrArray("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sa.IsEmpty()
	}
}

func BenchmarkNewStringOrArray(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStringOrArray("test")
	}
}

// Tests for enhanced StructuredQuery with advanced security monitoring fields

func TestStructuredQuery_JSONSerialization_Basic(t *testing.T) {
	tests := []struct {
		name  string
		query *StructuredQuery
	}{
		{
			name: "basic query with enhanced log source",
			query: &StructuredQuery{
				LogSource: "node-auditd",
				Verb:      StringOrArray{value: "get"},
				Resource:  StringOrArray{value: "pods"},
				Limit:     20,
			},
		},
		{
			name: "query with correlation fields",
			query: &StructuredQuery{
				LogSource:         "kube-apiserver",
				CorrelationFields: []string{"user", "source_ip", "timestamp"},
				Limit:             50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.query)
			if err != nil {
				t.Errorf("JSON marshal failed: %v", err)
				return
			}

			// Unmarshal back
			var unmarshaled StructuredQuery
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("JSON unmarshal failed: %v", err)
				return
			}

			// Verify key fields
			if unmarshaled.LogSource != tt.query.LogSource {
				t.Errorf("LogSource mismatch: got %v, want %v", unmarshaled.LogSource, tt.query.LogSource)
			}

			if tt.query.CorrelationFields != nil {
				if len(unmarshaled.CorrelationFields) != len(tt.query.CorrelationFields) {
					t.Errorf("CorrelationFields length mismatch: got %d, want %d", 
						len(unmarshaled.CorrelationFields), len(tt.query.CorrelationFields))
				}
			}
		})
	}
}

func TestMultiSourceConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		config *MultiSourceConfig
	}{
		{
			name: "complete multi-source config",
			config: &MultiSourceConfig{
				PrimarySource:     "kube-apiserver",
				SecondarySources:  []string{"oauth-server", "node-auditd"},
				CorrelationWindow: "30_minutes",
				CorrelationFields: []string{"user", "source_ip"},
				JoinType:          "inner",
			},
		},
		{
			name: "minimal multi-source config",
			config: &MultiSourceConfig{
				PrimarySource: "openshift-apiserver",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test as part of StructuredQuery
			query := &StructuredQuery{
				LogSource:   "kube-apiserver",
				MultiSource: tt.config,
				Limit:       20,
			}

			// Marshal to JSON
			data, err := json.Marshal(query)
			if err != nil {
				t.Errorf("JSON marshal failed: %v", err)
				return
			}

			// Unmarshal back
			var unmarshaled StructuredQuery
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("JSON unmarshal failed: %v", err)
				return
			}

			// Verify multi-source config
			if unmarshaled.MultiSource == nil {
				t.Error("MultiSource config was lost during serialization")
				return
			}

			if unmarshaled.MultiSource.PrimarySource != tt.config.PrimarySource {
				t.Errorf("PrimarySource mismatch: got %v, want %v", 
					unmarshaled.MultiSource.PrimarySource, tt.config.PrimarySource)
			}

			if tt.config.SecondarySources != nil {
				if len(unmarshaled.MultiSource.SecondarySources) != len(tt.config.SecondarySources) {
					t.Errorf("SecondarySources length mismatch: got %d, want %d", 
						len(unmarshaled.MultiSource.SecondarySources), len(tt.config.SecondarySources))
				}
			}
		})
	}
}

func TestAdvancedAnalysisConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		config *AdvancedAnalysisConfig
	}{
		{
			name: "APT reconnaissance analysis",
			config: &AdvancedAnalysisConfig{
				Type:                  "apt_reconnaissance_detection",
				KillChainPhase:        "reconnaissance",
				MultiStageCorrelation: true,
				StatisticalAnalysis: &StatisticalAnalysisConfig{
					PatternDeviationThreshold: 2.5,
					ConfidenceInterval:        0.95,
					SampleSizeMinimum:         100,
					BaselineWindow:            "30_days",
				},
				Threshold:  5,
				TimeWindow: "15_minutes",
			},
		},
		{
			name: "basic analysis type",
			config: &AdvancedAnalysisConfig{
				Type:      "anomaly_detection",
				Threshold: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test as part of StructuredQuery
			query := &StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis:  tt.config,
				Limit:     20,
			}

			// Marshal to JSON
			data, err := json.Marshal(query)
			if err != nil {
				t.Errorf("JSON marshal failed: %v", err)
				return
			}

			// Unmarshal back
			var unmarshaled StructuredQuery
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("JSON unmarshal failed: %v", err)
				return
			}

			// Verify analysis config
			if unmarshaled.Analysis == nil {
				t.Error("Analysis config was lost during serialization")
				return
			}

			if unmarshaled.Analysis.Type != tt.config.Type {
				t.Errorf("Analysis type mismatch: got %v, want %v", 
					unmarshaled.Analysis.Type, tt.config.Type)
			}

			if tt.config.StatisticalAnalysis != nil {
				if unmarshaled.Analysis.StatisticalAnalysis == nil {
					t.Error("StatisticalAnalysis was lost during serialization")
					return
				}

				if unmarshaled.Analysis.StatisticalAnalysis.PatternDeviationThreshold != 
					tt.config.StatisticalAnalysis.PatternDeviationThreshold {
					t.Errorf("PatternDeviationThreshold mismatch: got %v, want %v",
						unmarshaled.Analysis.StatisticalAnalysis.PatternDeviationThreshold,
						tt.config.StatisticalAnalysis.PatternDeviationThreshold)
				}
			}
		})
	}
}

func TestBehavioralAnalysisConfig_JSONSerialization(t *testing.T) {
	config := &BehavioralAnalysisConfig{
		UserProfiling:      true,
		BaselineComparison: true,
		RiskScoring: &RiskScoringConfig{
			Enabled:   true,
			Algorithm: "weighted_sum",
			RiskFactors: []string{"privilege_level", "resource_sensitivity", "timing_anomaly"},
			WeightingScheme: map[string]float64{
				"privilege_level":      0.4,
				"resource_sensitivity": 0.3,
				"timing_anomaly":       0.3,
			},
		},
		AnomalyDetection: &AnomalyDetectionConfig{
			Algorithm:     "isolation_forest",
			Contamination: 0.1,
			Sensitivity:   0.8,
		},
		BaselineWindow: "30_days",
		LearningPeriod: "7_days",
	}

	// Test as part of StructuredQuery
	query := &StructuredQuery{
		LogSource:          "kube-apiserver",
		BehavioralAnalysis: config,
		Limit:              20,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify behavioral analysis config
	if unmarshaled.BehavioralAnalysis == nil {
		t.Error("BehavioralAnalysis config was lost during serialization")
		return
	}

	if !unmarshaled.BehavioralAnalysis.UserProfiling {
		t.Error("UserProfiling should be true")
	}

	if unmarshaled.BehavioralAnalysis.RiskScoring == nil {
		t.Error("RiskScoring config was lost during serialization")
		return
	}

	if unmarshaled.BehavioralAnalysis.RiskScoring.Algorithm != "weighted_sum" {
		t.Errorf("RiskScoring algorithm mismatch: got %v, want weighted_sum", 
			unmarshaled.BehavioralAnalysis.RiskScoring.Algorithm)
	}
}

func TestThreatIntelligenceConfig_JSONSerialization(t *testing.T) {
	config := &ThreatIntelligenceConfig{
		IOCCorrelation:        true,
		AttackPatternMatching: true,
		ThreatActorAttribution: true,
		FeedSources:           []string{"mitre_att&ck", "custom_feeds"},
		ConfidenceThreshold:   0.7,
		TTPAnalysis:           true,
	}

	// Test as part of StructuredQuery
	query := &StructuredQuery{
		LogSource:         "kube-apiserver",
		ThreatIntelligence: config,
		Limit:             20,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify threat intelligence config
	if unmarshaled.ThreatIntelligence == nil {
		t.Error("ThreatIntelligence config was lost during serialization")
		return
	}

	if !unmarshaled.ThreatIntelligence.IOCCorrelation {
		t.Error("IOCCorrelation should be true")
	}

	if len(unmarshaled.ThreatIntelligence.FeedSources) != 2 {
		t.Errorf("FeedSources length mismatch: got %d, want 2", 
			len(unmarshaled.ThreatIntelligence.FeedSources))
	}
}

func TestMachineLearningConfig_JSONSerialization(t *testing.T) {
	config := &MachineLearningConfig{
		ModelType: "anomaly_detection",
		FeatureEngineering: &FeatureEngineeringConfig{
			TemporalFeatures:   true,
			BehavioralFeatures: true,
			NetworkFeatures:    true,
			SequentialFeatures: false,
		},
		TrainingWindow:      "30_days",
		PredictionThreshold: 0.8,
		ModelParameters: map[string]interface{}{
			"n_estimators": 100,
			"max_depth":    10,
		},
		ValidationMethod: "cross_validation",
	}

	// Test as part of StructuredQuery
	query := &StructuredQuery{
		LogSource:       "kube-apiserver",
		MachineLearning: config,
		Limit:           20,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify machine learning config
	if unmarshaled.MachineLearning == nil {
		t.Error("MachineLearning config was lost during serialization")
		return
	}

	if unmarshaled.MachineLearning.ModelType != "anomaly_detection" {
		t.Errorf("ModelType mismatch: got %v, want anomaly_detection", 
			unmarshaled.MachineLearning.ModelType)
	}

	if unmarshaled.MachineLearning.FeatureEngineering == nil {
		t.Error("FeatureEngineering config was lost during serialization")
		return
	}

	if !unmarshaled.MachineLearning.FeatureEngineering.TemporalFeatures {
		t.Error("TemporalFeatures should be true")
	}
}

func TestDetectionCriteriaConfig_JSONSerialization(t *testing.T) {
	config := &DetectionCriteriaConfig{
		RapidOperations: &RapidOperationsConfig{
			Threshold:  10,
			TimeWindow: "1_minute",
		},
		PrivilegeEscalationIndicators: true,
		LateralMovementPatterns:       true,
		DataAccessAnomalies:           true,
		ReconnaissanceIndicators:      true,
		UnusualAPIPatterns:            true,
		PersistenceMechanisms:         false,
		DefenseEvasion:                false,
	}

	// Test as part of StructuredQuery
	query := &StructuredQuery{
		LogSource:         "kube-apiserver",
		DetectionCriteria: config,
		Limit:             20,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify detection criteria config
	if unmarshaled.DetectionCriteria == nil {
		t.Error("DetectionCriteria config was lost during serialization")
		return
	}

	if unmarshaled.DetectionCriteria.RapidOperations == nil {
		t.Error("RapidOperations config was lost during serialization")
		return
	}

	if unmarshaled.DetectionCriteria.RapidOperations.Threshold != 10 {
		t.Errorf("RapidOperations threshold mismatch: got %d, want 10", 
			unmarshaled.DetectionCriteria.RapidOperations.Threshold)
	}

	if !unmarshaled.DetectionCriteria.PrivilegeEscalationIndicators {
		t.Error("PrivilegeEscalationIndicators should be true")
	}
}

func TestSecurityContextConfig_JSONSerialization(t *testing.T) {
	config := &SecurityContextConfig{
		SCCViolations:        true,
		PodSecurityStandards: "restricted",
		PrivilegeAnalysis:    true,
		CapabilityMonitoring: []string{"SYS_ADMIN", "NET_ADMIN"},
		HostAccessMonitoring: true,
		SELinuxViolations:    false,
	}

	// Test as part of StructuredQuery
	query := &StructuredQuery{
		LogSource:       "kube-apiserver",
		SecurityContext: config,
		Limit:           20,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify security context config
	if unmarshaled.SecurityContext == nil {
		t.Error("SecurityContext config was lost during serialization")
		return
	}

	if !unmarshaled.SecurityContext.SCCViolations {
		t.Error("SCCViolations should be true")
	}

	if unmarshaled.SecurityContext.PodSecurityStandards != "restricted" {
		t.Errorf("PodSecurityStandards mismatch: got %v, want restricted", 
			unmarshaled.SecurityContext.PodSecurityStandards)
	}

	if len(unmarshaled.SecurityContext.CapabilityMonitoring) != 2 {
		t.Errorf("CapabilityMonitoring length mismatch: got %d, want 2", 
			len(unmarshaled.SecurityContext.CapabilityMonitoring))
	}
}

func TestComplianceFrameworkConfig_JSONSerialization(t *testing.T) {
	config := &ComplianceFrameworkConfig{
		Standards: []string{"SOX", "PCI-DSS", "GDPR", "HIPAA"},
		Controls:  []string{"access_logging", "data_protection", "audit_trail"},
		Reporting: &ComplianceReportingConfig{
			Format:           "detailed",
			IncludeEvidence:  true,
			RetentionPeriod:  "7_years",
			DigitalSignature: true,
		},
		AuditTrail:         true,
		ViolationThreshold: 5,
		EvidenceCollection: true,
	}

	// Test as part of StructuredQuery
	query := &StructuredQuery{
		LogSource:           "kube-apiserver",
		ComplianceFramework: config,
		Limit:               20,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify compliance framework config
	if unmarshaled.ComplianceFramework == nil {
		t.Error("ComplianceFramework config was lost during serialization")
		return
	}

	if len(unmarshaled.ComplianceFramework.Standards) != 4 {
		t.Errorf("Standards length mismatch: got %d, want 4", 
			len(unmarshaled.ComplianceFramework.Standards))
	}

	if unmarshaled.ComplianceFramework.Reporting == nil {
		t.Error("Reporting config was lost during serialization")
		return
	}

	if unmarshaled.ComplianceFramework.Reporting.Format != "detailed" {
		t.Errorf("Reporting format mismatch: got %v, want detailed", 
			unmarshaled.ComplianceFramework.Reporting.Format)
	}
}

func TestTemporalAnalysisConfig_JSONSerialization(t *testing.T) {
	config := &TemporalAnalysisConfig{
		PatternType:          "periodic",
		IntervalDetection:    true,
		AnomalyThreshold:     2.0,
		BaselineWindow:       "30_days",
		SeasonalityDetection: true,
		TrendAnalysis:        true,
		CorrelationWindow:    "1_hour",
	}

	// Test as part of StructuredQuery
	query := &StructuredQuery{
		LogSource:        "kube-apiserver",
		TemporalAnalysis: config,
		Limit:            20,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify temporal analysis config
	if unmarshaled.TemporalAnalysis == nil {
		t.Error("TemporalAnalysis config was lost during serialization")
		return
	}

	if unmarshaled.TemporalAnalysis.PatternType != "periodic" {
		t.Errorf("PatternType mismatch: got %v, want periodic", 
			unmarshaled.TemporalAnalysis.PatternType)
	}

	if !unmarshaled.TemporalAnalysis.IntervalDetection {
		t.Error("IntervalDetection should be true")
	}

	if unmarshaled.TemporalAnalysis.AnomalyThreshold != 2.0 {
		t.Errorf("AnomalyThreshold mismatch: got %v, want 2.0", 
			unmarshaled.TemporalAnalysis.AnomalyThreshold)
	}
}

func TestStructuredQuery_ComplexAdvancedQuery(t *testing.T) {
	// Test a complex query with multiple advanced features like those from the functional tests
	query := &StructuredQuery{
		LogSource: "kube-apiserver",
		Analysis: &AdvancedAnalysisConfig{
			Type:                  "apt_reconnaissance_detection",
			KillChainPhase:        "reconnaissance",
			MultiStageCorrelation: true,
			StatisticalAnalysis: &StatisticalAnalysisConfig{
				PatternDeviationThreshold: 2.5,
				ConfidenceInterval:        0.95,
			},
		},
		MultiSource: &MultiSourceConfig{
			PrimarySource:     "kube-apiserver",
			SecondarySources:  []string{"oauth-server", "node-auditd"},
			CorrelationWindow: "30_minutes",
			CorrelationFields: []string{"user", "source_ip"},
		},
		BehavioralAnalysis: &BehavioralAnalysisConfig{
			AnomalyDetection: &AnomalyDetectionConfig{
				Algorithm:   "isolation_forest",
				Sensitivity: 0.8,
			},
		},
		DetectionCriteria: &DetectionCriteriaConfig{
			ReconnaissanceIndicators: true,
			UnusualAPIPatterns:       true,
		},
		Limit: 100,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify all components are preserved
	if unmarshaled.Analysis == nil {
		t.Error("Analysis config was lost")
	}
	if unmarshaled.MultiSource == nil {
		t.Error("MultiSource config was lost")
	}
	if unmarshaled.BehavioralAnalysis == nil {
		t.Error("BehavioralAnalysis config was lost")
	}
	if unmarshaled.DetectionCriteria == nil {
		t.Error("DetectionCriteria config was lost")
	}

	// Verify specific nested values
	if unmarshaled.Analysis.Type != "apt_reconnaissance_detection" {
		t.Errorf("Analysis type mismatch: got %v, want apt_reconnaissance_detection", 
			unmarshaled.Analysis.Type)
	}

	if len(unmarshaled.MultiSource.SecondarySources) != 2 {
		t.Errorf("MultiSource secondary sources length mismatch: got %d, want 2", 
			len(unmarshaled.MultiSource.SecondarySources))
	}
}

func TestStructuredQuery_BackwardCompatibility(t *testing.T) {
	// Test that existing basic queries still work
	query := &StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      StringOrArray{value: "delete"},
		Resource:  StringOrArray{value: "secrets"},
		Timeframe: "yesterday",
		ExcludeUsers: []string{"system:", "kube-"},
		Limit:     20,
	}

	// Marshal to JSON
	data, err := json.Marshal(query)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
		return
	}

	// Unmarshal back
	var unmarshaled StructuredQuery
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
		return
	}

	// Verify all basic fields are preserved
	if unmarshaled.LogSource != "kube-apiserver" {
		t.Errorf("LogSource mismatch: got %v, want kube-apiserver", unmarshaled.LogSource)
	}

	if unmarshaled.Verb.GetString() != "delete" {
		t.Errorf("Verb mismatch: got %v, want delete", unmarshaled.Verb.GetString())
	}

	if unmarshaled.Resource.GetString() != "secrets" {
		t.Errorf("Resource mismatch: got %v, want secrets", unmarshaled.Resource.GetString())
	}

	if len(unmarshaled.ExcludeUsers) != 2 {
		t.Errorf("ExcludeUsers length mismatch: got %d, want 2", len(unmarshaled.ExcludeUsers))
	}

	// Verify advanced fields are nil (not set)
	if unmarshaled.Analysis != nil {
		t.Error("Analysis should be nil for basic query")
	}
	if unmarshaled.MultiSource != nil {
		t.Error("MultiSource should be nil for basic query")
	}
	if unmarshaled.BehavioralAnalysis != nil {
		t.Error("BehavioralAnalysis should be nil for basic query")
	}
}
