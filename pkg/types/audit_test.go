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
