package normalizers

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// JSONNormalizer standardizes JSON structure and field formats within StructuredQuery.
type JSONNormalizer struct{}

func NewJSONNormalizer() interfaces.Normalizer { return &JSONNormalizer{} }

// Normalize converts empty/null variants to defaults, coerces string/number
// representations, and normalizes timestamp/timeframe formats.
func (n *JSONNormalizer) Normalize(q *types.StructuredQuery) (*types.StructuredQuery, error) {
	if q == nil {
		return nil, fmt.Errorf("normalize: query is nil")
	}

	out := *q // shallow copy OK; fields are values or small structs/pointers

	// LogSource default
	if strings.TrimSpace(out.LogSource) == "" {
		out.LogSource = "kube-apiserver"
	}

	// Limit sane default and bounds
	if out.Limit <= 0 {
		out.Limit = 20
	}
	if out.Limit > 1000 {
		out.Limit = 1000
	}

	// Normalize common string-or-array fields by trimming whitespace entries
	normalizeStringOrArray := func(sa types.StringOrArray) types.StringOrArray {
		if sa.IsString() {
			return *types.NewStringOrArray(strings.TrimSpace(sa.GetString()))
		}
		if arr := sa.GetArray(); arr != nil {
			trimmed := make([]string, 0, len(arr))
			for _, s := range arr {
				s = strings.TrimSpace(s)
				if s != "" {
					trimmed = append(trimmed, s)
				}
			}
			return *types.NewStringOrArray(trimmed)
		}
		return sa
	}

	out.Verb = normalizeStringOrArray(out.Verb)
	out.Resource = normalizeStringOrArray(out.Resource)
	out.Namespace = normalizeStringOrArray(out.Namespace)
	out.User = normalizeStringOrArray(out.User)
	out.ResponseStatus = normalizeStringOrArray(out.ResponseStatus)
	out.SourceIP = normalizeStringOrArray(out.SourceIP)
	out.GroupBy = normalizeStringOrArray(out.GroupBy)

	// Normalize timeframe keywords
	switch strings.ToLower(strings.TrimSpace(out.Timeframe)) {
	case "", "recent", "default":
		// leave empty or set a system default if required
	case "1h", "1_hour", "1-hour", "hour", "last_hour":
		out.Timeframe = "1_hour_ago"
	case "today", "current_day":
		out.Timeframe = "today"
	case "yesterday", "prev_day":
		out.Timeframe = "yesterday"
	}

	// Sanity check time range order
	if out.TimeRange != nil {
		if out.TimeRange.End.Before(out.TimeRange.Start) {
			// swap
			start := out.TimeRange.Start
			out.TimeRange.Start = out.TimeRange.End
			out.TimeRange.End = start
		}
		// clamp to reasonable window if identical
		if out.TimeRange.End.Equal(out.TimeRange.Start) {
			out.TimeRange.End = out.TimeRange.Start.Add(time.Hour)
		}
	}

	return &out, nil
}
