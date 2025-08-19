package template

import (
	"strings"
	"testing"
)

func TestNewTemplateParser(t *testing.T) {
	parser := NewTemplateParser()
	if parser == nil {
		t.Fatal("NewTemplateParser returned nil")
	}
	
	if parser.placeholderRegex == nil {
		t.Error("placeholder regex not initialized")
	}
	
	if len(parser.requiredFields) == 0 {
		t.Error("no required fields configured")
	}
}

func TestTemplateParser_Parse_EmptyTemplate(t *testing.T) {
	parser := NewTemplateParser()
	
	templates := []string{"", "   ", "\n\t  "}
	
	for _, template := range templates {
		parsed, err := parser.Parse(template)
		if err != nil {
			t.Errorf("Error parsing empty template: %v", err)
		}
		
		if !parsed.IsValid {
			t.Error("Empty template should be valid")
		}
		
		if len(parsed.Segments) != 1 {
			t.Errorf("Expected 1 segment for empty template, got %d", len(parsed.Segments))
		}
		
		if parsed.Segments[0].IsPlaceholder {
			t.Error("Empty template segment should not be a placeholder")
		}
	}
}

func TestTemplateParser_Parse_ValidTemplate(t *testing.T) {
	parser := NewTemplateParser()
	template := "{system_prompt} instructions {examples} examples {query} query"
	
	parsed, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Error parsing valid template: %v", err)
	}
	
	if !parsed.IsValid {
		t.Errorf("Valid template should be valid, errors: %v", parsed.Errors)
	}
	
	// Should have 6 segments: text, placeholder, text, placeholder, text, placeholder
	expectedSegments := 6
	if len(parsed.Segments) != expectedSegments {
		t.Errorf("Expected %d segments, got %d", expectedSegments, len(parsed.Segments))
	}
	
	// Check placeholders
	expectedPlaceholders := []string{"system_prompt", "examples", "query"}
	if len(parsed.Placeholders) != len(expectedPlaceholders) {
		t.Errorf("Expected %d placeholders, got %d", len(expectedPlaceholders), len(parsed.Placeholders))
	}
	
	for _, expected := range expectedPlaceholders {
		if _, found := parsed.Placeholders[expected]; !found {
			t.Errorf("Expected placeholder %s not found", expected)
		}
	}
}

func TestTemplateParser_Parse_InvalidTemplate(t *testing.T) {
	parser := NewTemplateParser()
	
	tests := []struct {
		name     string
		template string
		expectValid bool
	}{
		{"missing required", "{system_prompt}{examples}", false},
		{"unknown placeholder", "{system_prompt}{examples}{query}{unknown}", false},
		{"valid with optional", "{system_prompt}{examples}{query}{timestamp}", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.template)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			
			if parsed.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v. Errors: %v", tt.expectValid, parsed.IsValid, parsed.Errors)
			}
		})
	}
}

func TestTemplateParser_Render(t *testing.T) {
	parser := NewTemplateParser()
	template := "System: {system_prompt}\nExamples: {examples}\nQuery: {query}"
	
	parsed, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	values := map[string]string{
		"system_prompt": "You are helpful",
		"examples":      "Example 1\nExample 2",
		"query":         "Test query",
	}
	
	result, err := parser.Render(parsed, values)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	
	expected := "System: You are helpful\nExamples: Example 1\nExample 2\nQuery: Test query"
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestTemplateParser_Render_WithMissingValues(t *testing.T) {
	parser := NewTemplateParser()
	template := "{system_prompt}{examples}{query}{timestamp}{session_id}"
	
	parsed, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	values := map[string]string{
		"system_prompt": "test",
		"examples":      "examples",
		"query":         "query",
		// timestamp and session_id missing - should be replaced with empty strings
	}
	
	result, err := parser.Render(parsed, values)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	
	expected := "testexamplesquery"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestTemplateParser_ParseAndRender(t *testing.T) {
	parser := NewTemplateParser()
	template := "Hello {system_prompt}! Examples: {examples} Query: {query}"
	values := map[string]string{
		"system_prompt": "Assistant",
		"examples":      "",
		"query":         "Test",
	}
	
	result, err := parser.ParseAndRender(template, values)
	if err != nil {
		t.Fatalf("ParseAndRender error: %v", err)
	}
	
	expected := "Hello Assistant! Examples:  Query: Test"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestTemplateParser_Caching(t *testing.T) {
	parser := NewTemplateParser()
	template := "{system_prompt}{examples}{query}"
	
	// First parse - should miss cache
	parsed1, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	stats1 := parser.GetStats()
	if stats1.CacheMisses != 1 || stats1.CacheHits != 0 {
		t.Errorf("Expected 1 miss, 0 hits after first parse, got %d misses, %d hits", 
			stats1.CacheMisses, stats1.CacheHits)
	}
	
	// Second parse - should hit cache
	parsed2, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	stats2 := parser.GetStats()
	if stats2.CacheMisses != 1 || stats2.CacheHits != 1 {
		t.Errorf("Expected 1 miss, 1 hit after second parse, got %d misses, %d hits", 
			stats2.CacheMisses, stats2.CacheHits)
	}
	
	// Should return same template object
	if parsed1 != parsed2 {
		t.Error("Cache should return same template object")
	}
	
	// Use count should increase
	if parsed2.UseCount != 2 {
		t.Errorf("Expected use count 2, got %d", parsed2.UseCount)
	}
}

func TestTemplateParser_CacheEviction(t *testing.T) {
	config := TemplateParserConfig{
		RequiredFields:     []string{"system_prompt"},
		OptionalFields:     []string{},
		MaxCacheSize:       2,
		PlaceholderPattern: `\{([a-zA-Z_][a-zA-Z0-9_]*)\}`,
	}
	parser := NewTemplateParserWithConfig(config)
	
	// Fill cache to capacity
	_, err := parser.Parse("{system_prompt} template 1")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	_, err = parser.Parse("{system_prompt} template 2")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	if parser.getCacheSize() != 2 {
		t.Errorf("Expected cache size 2, got %d", parser.getCacheSize())
	}
	
	// Add third template - should evict least used
	_, err = parser.Parse("{system_prompt} template 3")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	if parser.getCacheSize() != 2 {
		t.Errorf("Expected cache size to remain 2 after eviction, got %d", parser.getCacheSize())
	}
}

func TestTemplateParser_ClearCache(t *testing.T) {
	parser := NewTemplateParser()
	
	// Add some templates to cache
	parser.Parse("{system_prompt}{examples}{query}")
	parser.Parse("{system_prompt} different template {query}")
	
	if parser.getCacheSize() == 0 {
		t.Error("Cache should not be empty before clear")
	}
	
	parser.ClearCache()
	
	if parser.getCacheSize() != 0 {
		t.Errorf("Cache should be empty after clear, got size %d", parser.getCacheSize())
	}
	
	stats := parser.GetStats()
	if stats.CacheHits != 0 || stats.CacheMisses != 0 {
		t.Error("Stats should be reset after cache clear")
	}
}

func TestTemplateParser_ValidateTemplate(t *testing.T) {
	parser := NewTemplateParser()
	
	tests := []struct {
		name        string
		template    string
		expectValid bool
		expectErrors int
	}{
		{"empty template", "", true, 0},
		{"valid template", "{system_prompt}{examples}{query}", true, 0},
		{"missing required", "{system_prompt}{examples}", false, 1},
		{"unknown placeholder", "{system_prompt}{examples}{query}{unknown}", false, 1},
		{"multiple issues", "{system_prompt}{unknown}", false, 3},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ValidateTemplate(tt.template)
			
			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v", tt.expectValid, result.IsValid)
			}
			
			if len(result.Errors) != tt.expectErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestTemplateParser_PlaceholderClassification(t *testing.T) {
	parser := NewTemplateParser()
	
	tests := []struct {
		name         string
		placeholder  string
		expectedType PlaceholderType
	}{
		{"required field", "system_prompt", TypeRequired},
		{"optional field", "timestamp", TypeOptional},
		{"unknown field", "unknown_field", TypeUnknown},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placeholderType := parser.classifyPlaceholder(tt.placeholder)
			if placeholderType != tt.expectedType {
				t.Errorf("Expected type %v, got %v for placeholder %s", 
					tt.expectedType, placeholderType, tt.placeholder)
			}
		})
	}
}

func TestTemplateParser_CustomConfig(t *testing.T) {
	config := TemplateParserConfig{
		RequiredFields:     []string{"custom_prompt"},
		OptionalFields:     []string{"custom_optional"},
		MaxCacheSize:       500,
		PlaceholderPattern: `\{([a-zA-Z_][a-zA-Z0-9_]*)\}`,
	}
	
	parser := NewTemplateParserWithConfig(config)
	
	// Test with custom required field
	template := "{custom_prompt}"
	parsed, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	if !parsed.IsValid {
		t.Errorf("Template with custom required field should be valid, errors: %v", parsed.Errors)
	}
	
	// Test with missing custom required field
	template2 := "{custom_optional}"
	parsed2, err := parser.Parse(template2)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	if parsed2.IsValid {
		t.Error("Template missing custom required field should be invalid")
	}
}

func TestTemplateParser_SegmentStructure(t *testing.T) {
	parser := NewTemplateParser()
	template := "Before {system_prompt} middle {examples} after"
	
	parsed, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	expectedSegments := []struct {
		isPlaceholder bool
		content       string
	}{
		{false, "Before "},
		{true, "{system_prompt}"},
		{false, " middle "},
		{true, "{examples}"},
		{false, " after"},
	}
	
	if len(parsed.Segments) != len(expectedSegments) {
		t.Errorf("Expected %d segments, got %d", len(expectedSegments), len(parsed.Segments))
	}
	
	for i, expected := range expectedSegments {
		if i >= len(parsed.Segments) {
			break
		}
		
		segment := parsed.Segments[i]
		if segment.IsPlaceholder != expected.isPlaceholder {
			t.Errorf("Segment %d: expected IsPlaceholder=%v, got %v", 
				i, expected.isPlaceholder, segment.IsPlaceholder)
		}
		
		if segment.Content != expected.content {
			t.Errorf("Segment %d: expected content %q, got %q", 
				i, expected.content, segment.Content)
		}
	}
}

// Benchmark tests
func BenchmarkTemplateParser_Parse(b *testing.B) {
	parser := NewTemplateParser()
	template := "{system_prompt} instructions {examples} examples {query} query"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(template)
	}
}

func BenchmarkTemplateParser_ParseCached(b *testing.B) {
	parser := NewTemplateParser()
	template := "{system_prompt} instructions {examples} examples {query} query"
	
	// Pre-populate cache
	parser.Parse(template)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(template)
	}
}

func BenchmarkTemplateParser_Render(b *testing.B) {
	parser := NewTemplateParser()
	template := "{system_prompt} instructions {examples} examples {query} query"
	
	parsed, _ := parser.Parse(template)
	values := map[string]string{
		"system_prompt": "You are helpful",
		"examples":      "Example 1\nExample 2",
		"query":         "Test query",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Render(parsed, values)
	}
}

func BenchmarkTemplateParser_ParseAndRender(b *testing.B) {
	parser := NewTemplateParser()
	template := "{system_prompt} instructions {examples} examples {query} query"
	values := map[string]string{
		"system_prompt": "You are helpful",
		"examples":      "Example 1\nExample 2",
		"query":         "Test query",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseAndRender(template, values)
	}
}

func BenchmarkTemplateParser_RenderLarge(b *testing.B) {
	parser := NewTemplateParser()
	template := strings.Repeat("{system_prompt} ", 100) + "{examples} " + strings.Repeat("{query} ", 100)
	
	parsed, _ := parser.Parse(template)
	values := map[string]string{
		"system_prompt": strings.Repeat("System instruction ", 10),
		"examples":      strings.Repeat("Example content ", 20),
		"query":         strings.Repeat("Query content ", 10),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Render(parsed, values)
	}
}