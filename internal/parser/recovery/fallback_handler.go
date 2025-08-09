package recovery

import (
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// FallbackHandler implements interfaces.FallbackHandler by extracting minimal
// information from the raw response and original query to construct a
// StructuredQuery with safe defaults.
type FallbackHandler struct{}

func NewFallbackHandler() interfaces.FallbackHandler { return &FallbackHandler{} }

// CreateMinimalQuery creates a conservative StructuredQuery based on heuristics.
func (h *FallbackHandler) CreateMinimalQuery(raw *types.RawResponse, _ string, originalQuery string) (*types.StructuredQuery, error) {
	q := &types.StructuredQuery{LogSource: "kube-apiserver", Limit: 20}

	content := ""
	if raw != nil {
		content = raw.Content
	}
	lower := strings.ToLower(content + " " + originalQuery)
	switch {
	case strings.Contains(lower, "oauth"):
		q.LogSource = "oauth-server"
	case strings.Contains(lower, "openshift"):
		q.LogSource = "openshift-apiserver"
	}

	// simple timeframe hints
	if strings.Contains(lower, "today") {
		q.Timeframe = "today"
	} else if strings.Contains(lower, "yesterday") {
		q.Timeframe = "yesterday"
	} else if strings.Contains(lower, "hour") {
		q.Timeframe = "1_hour_ago"
	}

	return q, nil
}
