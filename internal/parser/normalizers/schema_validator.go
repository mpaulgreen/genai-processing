package normalizers

import (
	"fmt"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// SchemaValidator implements interfaces.SchemaValidator for structural checks.
type SchemaValidator struct{}

func NewSchemaValidator() interfaces.SchemaValidator { return &SchemaValidator{} }

// ValidateSchema enforces basic schema constraints on StructuredQuery.
func (v *SchemaValidator) ValidateSchema(q *types.StructuredQuery) error {
	if q == nil {
		return fmt.Errorf("schema: query is nil")
	}

	// Required: LogSource (already defaulted by normalizer, but ensure)
	if strings.TrimSpace(q.LogSource) == "" {
		return fmt.Errorf("schema: log_source is required")
	}

	// Limit range
	if q.Limit < 0 || q.Limit > 1000 {
		return fmt.Errorf("schema: limit out of range (0..1000)")
	}

	// Timeframe: allow empty or known normalized values
	switch strings.ToLower(strings.TrimSpace(q.Timeframe)) {
	case "", "today", "yesterday", "1_hour_ago":
	default:
		// do not hard fail; but flag unusual entries as error for schema layer
		return fmt.Errorf("schema: unsupported timeframe '%s'", q.Timeframe)
	}

	// If TimeRange provided, ensure Start <= End
	if q.TimeRange != nil {
		if q.TimeRange.End.Before(q.TimeRange.Start) {
			return fmt.Errorf("schema: time_range.end before time_range.start")
		}
	}

	return nil
}
