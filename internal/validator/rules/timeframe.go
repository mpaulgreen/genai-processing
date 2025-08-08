package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// TimeframeRule implements timeframe validation and limits
type TimeframeRule struct {
	maxDaysBack       int
	defaultLimit      int
	maxLimit          int
	minLimit          int
	allowedTimeframes []string
	enabled           bool
}

// NewTimeframeRule creates a new timeframe validation rule
func NewTimeframeRule(config map[string]interface{}) *TimeframeRule {
	rule := &TimeframeRule{
		enabled: true,
	}

	// Extract configuration values
	if maxDays, ok := config["max_days_back"].(int); ok {
		rule.maxDaysBack = maxDays
	} else {
		rule.maxDaysBack = 90
	}

	if defaultLimit, ok := config["default_limit"].(int); ok {
		rule.defaultLimit = defaultLimit
	} else {
		rule.defaultLimit = 20
	}

	if maxLimit, ok := config["max_limit"].(int); ok {
		rule.maxLimit = maxLimit
	} else {
		rule.maxLimit = 1000
	}

	if minLimit, ok := config["min_limit"].(int); ok {
		rule.minLimit = minLimit
	} else {
		rule.minLimit = 1
	}

	if allowedTimeframes, ok := config["allowed_timeframes"].([]interface{}); ok {
		for _, tf := range allowedTimeframes {
			if str, ok := tf.(string); ok {
				rule.allowedTimeframes = append(rule.allowedTimeframes, str)
			}
		}
	} else {
		// Default allowed timeframes
		rule.allowedTimeframes = []string{
			"today", "yesterday", "1_hour_ago", "2_hours_ago", "3_hours_ago",
			"6_hours_ago", "12_hours_ago", "1_day_ago", "2_days_ago", "3_days_ago",
			"7_days_ago", "14_days_ago", "30_days_ago", "60_days_ago", "90_days_ago",
		}
	}

	return rule
}

// Validate applies timeframe validation to the query
func (t *TimeframeRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "timeframe_validation",
		Severity:        "medium",
		Message:         "Timeframe validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		QuerySnapshot:   query,
	}

	// Validate timeframe string
	if query.Timeframe != "" {
		if !t.isAllowedTimeframe(query.Timeframe) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Timeframe '%s' is not in allowed list", query.Timeframe))
		}

		// Check if timeframe exceeds max days back
		if days := t.extractDaysFromTimeframe(query.Timeframe); days > t.maxDaysBack {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Timeframe '%s' exceeds maximum allowed days back (%d)", query.Timeframe, t.maxDaysBack))
		}
	}

	// Validate custom time range
	if query.TimeRange != nil {
		if err := t.validateTimeRange(query.TimeRange); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}

	// Validate limit only if it's explicitly set (non-zero)
	if query.Limit != 0 {
		if query.Limit > t.maxLimit {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Limit %d exceeds maximum allowed limit of %d", query.Limit, t.maxLimit))
		}
		if query.Limit < t.minLimit {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Limit %d is below minimum allowed limit of %d", query.Limit, t.minLimit))
		}
		if query.Limit < 0 {
			// Negative limits are invalid
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Limit %d is below minimum allowed limit of %d", query.Limit, t.minLimit))
		}
	}

	// Validate business hours configuration
	if query.BusinessHours != nil {
		if err := t.validateBusinessHours(query.BusinessHours); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Timeframe validation failed"
		result.Severity = "medium"
		result.Recommendations = append(result.Recommendations,
			"Use allowed timeframe values from the configuration",
			fmt.Sprintf("Keep timeframes within %d days back", t.maxDaysBack),
			fmt.Sprintf("Use limits between %d and %d", t.minLimit, t.maxLimit),
			"Ensure time ranges are valid and within allowed bounds")
	}

	return result
}

// GetRuleName returns the rule name
func (t *TimeframeRule) GetRuleName() string {
	return "timeframe_validation"
}

// GetRuleDescription returns the rule description
func (t *TimeframeRule) GetRuleDescription() string {
	return "Validates timeframe limits and constraints for audit queries"
}

// IsEnabled indicates if the rule is enabled
func (t *TimeframeRule) IsEnabled() bool {
	return t.enabled
}

// GetSeverity returns the rule severity
func (t *TimeframeRule) GetSeverity() string {
	return "medium"
}

// Helper methods
func (t *TimeframeRule) isAllowedTimeframe(timeframe string) bool {
	for _, allowed := range t.allowedTimeframes {
		if strings.EqualFold(timeframe, allowed) {
			return true
		}
	}
	return false
}

func (t *TimeframeRule) extractDaysFromTimeframe(timeframe string) int {
	// Extract days from timeframe patterns
	patterns := map[string]*regexp.Regexp{
		"days":   regexp.MustCompile(`(\d+)_days?_ago`),
		"weeks":  regexp.MustCompile(`(\d+)_weeks?_ago`),
		"months": regexp.MustCompile(`(\d+)_months?_ago`),
	}

	for unit, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(timeframe); len(matches) > 1 {
			if value, err := strconv.Atoi(matches[1]); err == nil {
				switch unit {
				case "days":
					return value
				case "weeks":
					return value * 7
				case "months":
					return value * 30 // Approximate
				}
			}
		}
	}

	// Handle special cases
	switch strings.ToLower(timeframe) {
	case "today", "yesterday":
		return 1
	case "1_hour_ago", "2_hours_ago", "3_hours_ago", "6_hours_ago", "12_hours_ago":
		return 1 // Less than a day
	default:
		return 0 // Unknown timeframe
	}
}

func (t *TimeframeRule) validateTimeRange(timeRange *types.TimeRange) error {
	// Check if start is before end
	if timeRange.Start.After(timeRange.End) {
		return fmt.Errorf("time range start (%s) is after end (%s)",
			timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339))
	}

	// Check if time range is within max days back
	now := time.Now()
	maxStart := now.AddDate(0, 0, -t.maxDaysBack)

	if timeRange.Start.Before(maxStart) {
		return fmt.Errorf("time range start (%s) is more than %d days in the past",
			timeRange.Start.Format(time.RFC3339), t.maxDaysBack)
	}

	// Check if time range is not in the future
	if timeRange.Start.After(now) || timeRange.End.After(now) {
		return fmt.Errorf("time range cannot be in the future")
	}

	// Check if time range is reasonable (not too long)
	duration := timeRange.End.Sub(timeRange.Start)
	maxDuration := time.Duration(t.maxDaysBack) * 24 * time.Hour

	if duration > maxDuration {
		return fmt.Errorf("time range duration (%v) exceeds maximum allowed duration (%v)",
			duration, maxDuration)
	}

	return nil
}

func (t *TimeframeRule) validateBusinessHours(businessHours *types.BusinessHours) error {
	// Validate hour values
	if businessHours.StartHour < 0 || businessHours.StartHour > 23 {
		return fmt.Errorf("business hours start hour (%d) must be between 0 and 23", businessHours.StartHour)
	}

	if businessHours.EndHour < 0 || businessHours.EndHour > 23 {
		return fmt.Errorf("business hours end hour (%d) must be between 0 and 23", businessHours.EndHour)
	}

	// Validate that start is before end (allowing for overnight shifts)
	if businessHours.StartHour == businessHours.EndHour {
		return fmt.Errorf("business hours start and end hours cannot be the same")
	}

	return nil
}
