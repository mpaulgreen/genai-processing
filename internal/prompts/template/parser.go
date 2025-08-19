package template

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	promptErrors "genai-processing/internal/prompts/errors"
)

// PlaceholderType represents different types of placeholders
type PlaceholderType int

const (
	TypeRequired PlaceholderType = iota
	TypeOptional
	TypeUnknown
)

// Placeholder represents a parsed placeholder with metadata
type Placeholder struct {
	Name     string          `json:"name"`
	Type     PlaceholderType `json:"type"`
	Position int             `json:"position"`
	Length   int             `json:"length"`
}

// TemplateSegment represents a parsed segment of the template
type TemplateSegment struct {
	IsPlaceholder bool        `json:"is_placeholder"`
	Content       string      `json:"content"`
	Placeholder   *Placeholder `json:"placeholder,omitempty"`
}

// ParsedTemplate represents a pre-parsed template structure
type ParsedTemplate struct {
	Original    string             `json:"original"`
	Segments    []TemplateSegment  `json:"segments"`
	Placeholders map[string]*Placeholder `json:"placeholders"`
	IsValid     bool               `json:"is_valid"`
	Errors      []string           `json:"errors,omitempty"`
	ParsedAt    time.Time          `json:"parsed_at"`
	UseCount    int64              `json:"use_count"`
}

// TemplateParser provides template parsing and caching functionality
type TemplateParser struct {
	cache           sync.Map // map[string]*ParsedTemplate
	placeholderRegex *regexp.Regexp
	requiredFields  map[string]bool
	optionalFields  map[string]bool
	maxCacheSize    int
	cacheHits       int64
	cacheMisses     int64
	mu              sync.RWMutex
}

// NewTemplateParser creates a new template parser with default configuration
func NewTemplateParser() *TemplateParser {
	return NewTemplateParserWithConfig(TemplateParserConfig{
		RequiredFields: []string{"system_prompt", "examples", "query"},
		OptionalFields: []string{"timestamp", "session_id", "model_name", "provider"},
		MaxCacheSize:   1000,
		PlaceholderPattern: `\{([a-zA-Z_][a-zA-Z0-9_]*)\}`,
	})
}

// TemplateParserConfig provides configuration for the template parser
type TemplateParserConfig struct {
	RequiredFields     []string `json:"required_fields"`
	OptionalFields     []string `json:"optional_fields"`
	MaxCacheSize       int      `json:"max_cache_size"`
	PlaceholderPattern string   `json:"placeholder_pattern"`
}

// NewTemplateParserWithConfig creates a parser with custom configuration
func NewTemplateParserWithConfig(config TemplateParserConfig) *TemplateParser {
	regex := regexp.MustCompile(config.PlaceholderPattern)
	
	required := make(map[string]bool)
	for _, field := range config.RequiredFields {
		required[field] = true
	}
	
	optional := make(map[string]bool)
	for _, field := range config.OptionalFields {
		optional[field] = true
	}
	
	return &TemplateParser{
		placeholderRegex: regex,
		requiredFields:   required,
		optionalFields:   optional,
		maxCacheSize:     config.MaxCacheSize,
	}
}

// Parse parses a template and returns a ParsedTemplate (cached)
func (p *TemplateParser) Parse(template string) (*ParsedTemplate, error) {
	// Check cache first
	if cached, found := p.cache.Load(template); found {
		p.mu.Lock()
		p.cacheHits++
		p.mu.Unlock()
		
		parsedTemplate := cached.(*ParsedTemplate)
		parsedTemplate.UseCount++
		return parsedTemplate, nil
	}
	
	p.mu.Lock()
	p.cacheMisses++
	p.mu.Unlock()
	
	// Parse the template
	parsed, err := p.parseTemplate(template)
	if err != nil {
		return nil, err
	}
	
	// Store in cache if under limit
	p.storeParsedTemplate(template, parsed)
	
	return parsed, nil
}

// parseTemplate performs the actual template parsing
func (p *TemplateParser) parseTemplate(template string) (*ParsedTemplate, error) {
	parsed := &ParsedTemplate{
		Original:     template,
		Segments:     []TemplateSegment{},
		Placeholders: make(map[string]*Placeholder),
		IsValid:      true,
		ParsedAt:     time.Now(),
		UseCount:     1,
	}
	
	// Handle empty template
	if strings.TrimSpace(template) == "" {
		parsed.Segments = append(parsed.Segments, TemplateSegment{
			IsPlaceholder: false,
			Content:       template,
		})
		return parsed, nil
	}
	
	// Find all placeholder matches
	matches := p.placeholderRegex.FindAllStringSubmatchIndex(template, -1)
	
	lastEnd := 0
	for _, match := range matches {
		// Add text before placeholder
		if match[0] > lastEnd {
			parsed.Segments = append(parsed.Segments, TemplateSegment{
				IsPlaceholder: false,
				Content:       template[lastEnd:match[0]],
			})
		}
		
		// Extract placeholder name
		placeholderName := template[match[2]:match[3]]
		placeholderType := p.classifyPlaceholder(placeholderName)
		
		// Create placeholder
		placeholder := &Placeholder{
			Name:     placeholderName,
			Type:     placeholderType,
			Position: match[0],
			Length:   match[1] - match[0],
		}
		
		// Add placeholder segment
		parsed.Segments = append(parsed.Segments, TemplateSegment{
			IsPlaceholder: true,
			Content:       template[match[0]:match[1]],
			Placeholder:   placeholder,
		})
		
		// Store placeholder
		parsed.Placeholders[placeholderName] = placeholder
		
		lastEnd = match[1]
	}
	
	// Add remaining text
	if lastEnd < len(template) {
		parsed.Segments = append(parsed.Segments, TemplateSegment{
			IsPlaceholder: false,
			Content:       template[lastEnd:],
		})
	}
	
	// Validate template
	p.validateParsedTemplate(parsed)
	
	return parsed, nil
}

// classifyPlaceholder determines the type of a placeholder
func (p *TemplateParser) classifyPlaceholder(name string) PlaceholderType {
	if p.requiredFields[name] {
		return TypeRequired
	}
	if p.optionalFields[name] {
		return TypeOptional
	}
	return TypeUnknown
}

// validateParsedTemplate validates the parsed template structure
func (p *TemplateParser) validateParsedTemplate(parsed *ParsedTemplate) {
	var errors []string
	
	// Check for required fields
	for requiredField := range p.requiredFields {
		if _, found := parsed.Placeholders[requiredField]; !found {
			errors = append(errors, fmt.Sprintf("missing required placeholder: %s", requiredField))
		}
	}
	
	// Check for unknown placeholders
	for name, placeholder := range parsed.Placeholders {
		if placeholder.Type == TypeUnknown {
			errors = append(errors, fmt.Sprintf("unknown placeholder: %s", name))
		}
	}
	
	if len(errors) > 0 {
		parsed.IsValid = false
		parsed.Errors = errors
	}
}

// storeParsedTemplate stores a parsed template in cache with size limits
func (p *TemplateParser) storeParsedTemplate(template string, parsed *ParsedTemplate) {
	// Simple cache eviction based on count
	if p.getCacheSize() >= p.maxCacheSize {
		p.evictLeastUsed()
	}
	
	p.cache.Store(template, parsed)
}

// Render efficiently renders a parsed template with provided values
func (p *TemplateParser) Render(parsed *ParsedTemplate, values map[string]string) (string, error) {
	if !parsed.IsValid {
		return "", fmt.Errorf("cannot render invalid template: %v", parsed.Errors)
	}
	
	// Pre-calculate capacity for efficiency
	capacity := len(parsed.Original)
	for _, value := range values {
		capacity += len(value)
	}
	
	var builder strings.Builder
	builder.Grow(capacity)
	
	// Render segments efficiently
	for _, segment := range parsed.Segments {
		if segment.IsPlaceholder {
			// Replace with value or empty string
			if value, exists := values[segment.Placeholder.Name]; exists {
				builder.WriteString(value)
			}
			// Empty string for missing optional placeholders
		} else {
			builder.WriteString(segment.Content)
		}
	}
	
	return builder.String(), nil
}

// ParseAndRender is a convenience method that parses and renders in one call
func (p *TemplateParser) ParseAndRender(template string, values map[string]string) (string, error) {
	parsed, err := p.Parse(template)
	if err != nil {
		return "", err
	}
	
	return p.Render(parsed, values)
}

// GetStats returns parser statistics
func (p *TemplateParser) GetStats() ParserStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return ParserStats{
		CacheHits:   p.cacheHits,
		CacheMisses: p.cacheMisses,
		CacheSize:   p.getCacheSize(),
		HitRatio:    float64(p.cacheHits) / float64(p.cacheHits+p.cacheMisses),
	}
}

// ParserStats contains parser performance statistics
type ParserStats struct {
	CacheHits   int64   `json:"cache_hits"`
	CacheMisses int64   `json:"cache_misses"`
	CacheSize   int     `json:"cache_size"`
	HitRatio    float64 `json:"hit_ratio"`
}

// getCacheSize returns the current cache size
func (p *TemplateParser) getCacheSize() int {
	count := 0
	p.cache.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// evictLeastUsed removes the least used template from cache
func (p *TemplateParser) evictLeastUsed() {
	var leastUsedKey interface{}
	var leastUsedCount int64 = -1
	
	p.cache.Range(func(key, value interface{}) bool {
		parsed := value.(*ParsedTemplate)
		if leastUsedCount == -1 || parsed.UseCount < leastUsedCount {
			leastUsedKey = key
			leastUsedCount = parsed.UseCount
		}
		return true
	})
	
	if leastUsedKey != nil {
		p.cache.Delete(leastUsedKey)
	}
}

// ClearCache clears the template cache
func (p *TemplateParser) ClearCache() {
	p.cache.Range(func(key, _ interface{}) bool {
		p.cache.Delete(key)
		return true
	})
	
	p.mu.Lock()
	p.cacheHits = 0
	p.cacheMisses = 0
	p.mu.Unlock()
}

// ValidateTemplate validates a template without parsing it fully
func (p *TemplateParser) ValidateTemplate(template string) *promptErrors.TemplateValidationResult {
	result := &promptErrors.TemplateValidationResult{IsValid: true}
	
	if strings.TrimSpace(template) == "" {
		return result
	}
	
	// Quick validation
	matches := p.placeholderRegex.FindAllStringSubmatch(template, -1)
	placeholders := make(map[string]bool)
	
	for _, match := range matches {
		if len(match) > 1 {
			placeholders[match[1]] = true
		}
	}
	
	// Check required fields
	for requiredField := range p.requiredFields {
		if !placeholders[requiredField] {
			result.AddError(promptErrors.ErrorMissingPlaceholder, 
				fmt.Sprintf("missing required placeholder: %s", requiredField), 0, "")
		}
	}
	
	// Check unknown placeholders
	for placeholder := range placeholders {
		if !p.requiredFields[placeholder] && !p.optionalFields[placeholder] {
			result.AddError(promptErrors.ErrorUnknownPlaceholder, 
				fmt.Sprintf("unknown placeholder: %s", placeholder), 0, "")
		}
	}
	
	return result
}