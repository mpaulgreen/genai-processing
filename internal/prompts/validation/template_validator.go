package validation

import (
	"regexp"
	"strings"
	"unicode"

	promptErrors "genai-processing/internal/prompts/errors"
)

// PlaceholderConfig defines configuration for template placeholders
type PlaceholderConfig struct {
	Required []string `json:"required"`
	Optional []string `json:"optional"`
	Pattern  string   `json:"pattern"`
}

// TemplateValidator provides comprehensive template validation
type TemplateValidator struct {
	placeholderConfig PlaceholderConfig
	placeholderRegex  *regexp.Regexp
}

// NewTemplateValidator creates a new template validator with default configuration
func NewTemplateValidator() *TemplateValidator {
	return NewTemplateValidatorWithConfig(DefaultPlaceholderConfig())
}

// NewTemplateValidatorWithConfig creates a validator with custom configuration
func NewTemplateValidatorWithConfig(config PlaceholderConfig) *TemplateValidator {
	// Compile regex for placeholder pattern
	pattern := config.Pattern
	if pattern == "" {
		pattern = `\{([a-zA-Z_][a-zA-Z0-9_]*)\}`
	}
	
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// Fallback to default pattern if custom pattern is invalid
		regex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	}
	
	return &TemplateValidator{
		placeholderConfig: config,
		placeholderRegex:  regex,
	}
}

// DefaultPlaceholderConfig returns the default placeholder configuration
func DefaultPlaceholderConfig() PlaceholderConfig {
	return PlaceholderConfig{
		Required: []string{"system_prompt", "examples", "query"},
		Optional: []string{"timestamp", "session_id", "model_name", "provider"},
		Pattern:  `\{([a-zA-Z_][a-zA-Z0-9_]*)\}`,
	}
}

// ValidateTemplate performs comprehensive template validation
func (v *TemplateValidator) ValidateTemplate(template string) *promptErrors.TemplateValidationResult {
	result := &promptErrors.TemplateValidationResult{
		IsValid:      true,
		Placeholders: []string{},
	}
	
	// Skip validation for empty templates (they use fallback formatting)
	if strings.TrimSpace(template) == "" {
		return result
	}
	
	// 1. Basic syntax validation
	v.validateSyntax(template, result)
	
	// 2. Brace balance validation
	v.validateBraceBalance(template, result)
	
	// 3. Placeholder validation
	placeholders := v.extractPlaceholders(template)
	result.Placeholders = placeholders
	v.validatePlaceholders(placeholders, result)
	
	// 4. Structure validation
	v.validateStructure(template, result)
	
	return result
}

// validateSyntax checks for basic syntax issues
func (v *TemplateValidator) validateSyntax(template string, result *promptErrors.TemplateValidationResult) {
	// Check for null bytes or other problematic characters
	for i, r := range template {
		if r == 0 {
			result.AddError(promptErrors.ErrorInvalidSyntax, "Template contains null bytes", i, string(template[max(0, i-5):min(len(template), i+5)]))
		}
		
		// Check for problematic unicode characters that might cause issues
		if r == unicode.ReplacementChar {
			result.AddError(promptErrors.ErrorInvalidSyntax, "Template contains invalid unicode characters", i, string(template[max(0, i-5):min(len(template), i+5)]))
		}
	}
}

// validateBraceBalance ensures all braces are properly balanced
func (v *TemplateValidator) validateBraceBalance(template string, result *promptErrors.TemplateValidationResult) {
	var stack []int
	escaped := false
	
	for i, r := range template {
		if escaped {
			escaped = false
			continue
		}
		
		if r == '\\' {
			escaped = true
			continue
		}
		
		switch r {
		case '{':
			stack = append(stack, i)
		case '}':
			if len(stack) == 0 {
				result.AddError(promptErrors.ErrorUnbalancedBraces, "Closing brace without matching opening brace", i, string(template[max(0, i-5):min(len(template), i+5)]))
			} else {
				stack = stack[:len(stack)-1]
			}
		}
	}
	
	// Check for unclosed braces
	for _, pos := range stack {
		result.AddError(promptErrors.ErrorUnbalancedBraces, "Unclosed opening brace", pos, string(template[max(0, pos-5):min(len(template), pos+5)]))
	}
}

// extractPlaceholders finds all placeholders in the template
func (v *TemplateValidator) extractPlaceholders(template string) []string {
	matches := v.placeholderRegex.FindAllStringSubmatch(template, -1)
	placeholders := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	
	for _, match := range matches {
		if len(match) > 1 {
			placeholder := match[1]
			if !seen[placeholder] {
				placeholders = append(placeholders, placeholder)
				seen[placeholder] = true
			}
		}
	}
	
	return placeholders
}

// validatePlaceholders checks placeholder validity and requirements
func (v *TemplateValidator) validatePlaceholders(placeholders []string, result *promptErrors.TemplateValidationResult) {
	validPlaceholders := make(map[string]bool)
	
	// Build map of valid placeholders
	for _, p := range v.placeholderConfig.Required {
		validPlaceholders[p] = true
	}
	for _, p := range v.placeholderConfig.Optional {
		validPlaceholders[p] = true
	}
	
	// Check for unknown placeholders
	for _, placeholder := range placeholders {
		if !validPlaceholders[placeholder] {
			allValid := append(v.placeholderConfig.Required, v.placeholderConfig.Optional...)
			result.AddError(promptErrors.ErrorUnknownPlaceholder, "", 0, "")
			if len(result.Errors) > 0 {
				lastError := result.Errors[len(result.Errors)-1]
				lastError.Message = "Unknown placeholder '" + placeholder + "'"
				lastError.Suggestions = append([]string{
					"Valid placeholders: " + strings.Join(allValid, ", "),
				}, allValid...)
			}
		}
	}
	
	// Check for missing required placeholders
	foundRequired := make(map[string]bool)
	for _, placeholder := range placeholders {
		foundRequired[placeholder] = true
	}
	
	for _, required := range v.placeholderConfig.Required {
		if !foundRequired[required] {
			result.AddError(promptErrors.ErrorMissingPlaceholder, "Required placeholder '"+required+"' is missing", 0, "")
			result.AddSuggestion("Add {" + required + "} to your template")
		}
	}
}

// validateStructure checks for template structure issues
func (v *TemplateValidator) validateStructure(template string, result *promptErrors.TemplateValidationResult) {
	// Check for potential infinite recursion in placeholders
	v.checkCircularReferences(template, result)
	
	// Check for malformed placeholder syntax
	v.validatePlaceholderSyntax(template, result)
	
	// Warn about potential issues
	v.addStructureWarnings(template, result)
}

// checkCircularReferences detects potential circular references
func (v *TemplateValidator) checkCircularReferences(template string, result *promptErrors.TemplateValidationResult) {
	// For now, just check for obviously problematic patterns
	// This could be expanded to detect more complex circular references
	
	matches := v.placeholderRegex.FindAllStringSubmatch(template, -1)
	for _, match := range matches {
		if len(match) > 1 {
			placeholder := match[1]
			// Check if placeholder name appears in a way that might suggest recursion
			if strings.Contains(template, "{"+placeholder+"_"+placeholder+"}") {
				result.AddError(promptErrors.ErrorCircularReference, "Potential circular reference detected with placeholder '"+placeholder+"'", 0, "")
			}
		}
	}
}

// validatePlaceholderSyntax checks for malformed placeholder syntax
func (v *TemplateValidator) validatePlaceholderSyntax(template string, result *promptErrors.TemplateValidationResult) {
	// Find all potential placeholder-like patterns
	bracePattern := regexp.MustCompile(`\{[^}]*\}`)
	matches := bracePattern.FindAllStringIndex(template, -1)
	
	for _, match := range matches {
		start, end := match[0], match[1]
		placeholder := template[start:end]
		
		// Check if it matches our valid placeholder pattern
		if !v.placeholderRegex.MatchString(placeholder) {
			// Extract the content between braces
			content := placeholder[1 : len(placeholder)-1]
			position := start
			
			// Provide specific error messages for common issues
			if content == "" {
				result.AddError(promptErrors.ErrorInvalidPlaceholder, "Empty placeholder", position, placeholder)
				result.AddSuggestion("Remove empty braces or add a valid placeholder name")
			} else if strings.Contains(content, " ") {
				result.AddError(promptErrors.ErrorInvalidPlaceholder, "Placeholder names cannot contain spaces", position, placeholder)
				result.AddSuggestion("Use underscores instead of spaces: " + strings.ReplaceAll(content, " ", "_"))
			} else if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(content) {
				result.AddError(promptErrors.ErrorInvalidPlaceholder, "Invalid placeholder name '"+content+"'", position, placeholder)
				result.AddSuggestion("Use only letters, numbers, and underscores. Must start with a letter or underscore")
			}
		}
	}
}

// addStructureWarnings adds warnings for potential issues
func (v *TemplateValidator) addStructureWarnings(template string, result *promptErrors.TemplateValidationResult) {
	// Warn about very long templates
	if len(template) > 10000 {
		result.AddWarning(promptErrors.ErrorInvalidStructure, "Template is very long (>10k characters)", 0, "")
	}
	
	// Warn about templates with many placeholders
	placeholders := v.extractPlaceholders(template)
	if len(placeholders) > 20 {
		result.AddWarning(promptErrors.ErrorInvalidStructure, "Template has many placeholders (>20)", 0, "")
	}
	
	// Warn about potentially problematic patterns
	if strings.Contains(template, "{{") || strings.Contains(template, "}}") {
		result.AddWarning(promptErrors.ErrorInvalidSyntax, "Double braces detected - this might cause issues", 0, "")
	}
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// QuickValidate performs a fast validation check for common issues
func (v *TemplateValidator) QuickValidate(template string) error {
	if strings.TrimSpace(template) == "" {
		return nil // Empty templates are valid (use fallback)
	}
	
	// Quick brace balance check
	openBraces := strings.Count(template, "{")
	closeBraces := strings.Count(template, "}")
	if openBraces != closeBraces {
		return promptErrors.NewUnbalancedBracesError(0, "")
	}
	
	// Quick placeholder check
	placeholders := v.extractPlaceholders(template)
	foundRequired := make(map[string]bool)
	for _, p := range placeholders {
		foundRequired[p] = true
	}
	
	for _, required := range v.placeholderConfig.Required {
		if !foundRequired[required] {
			return promptErrors.NewMissingPlaceholderError(required)
		}
	}
	
	return nil
}

// GetSupportedPlaceholders returns all supported placeholders
func (v *TemplateValidator) GetSupportedPlaceholders() []string {
	return append(v.placeholderConfig.Required, v.placeholderConfig.Optional...)
}