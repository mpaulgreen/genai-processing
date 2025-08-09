package normalizers

import (
	"fmt"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// FieldMapper implements interfaces.FieldMapper and maps common aliases and
// value variants to the canonical StructuredQuery representation.
type FieldMapper struct{}

func NewFieldMapper() interfaces.FieldMapper { return &FieldMapper{} }

// MapFields applies mapping rules. Since we operate on a typed StructuredQuery,
// this primarily normalizes canonical values and known synonyms.
func (m *FieldMapper) MapFields(q *types.StructuredQuery) (*types.StructuredQuery, error) {
	if q == nil {
		return nil, fmt.Errorf("field mapper: query is nil")
	}
	out := *q

	// Normalize log source aliases
	switch strings.ToLower(strings.TrimSpace(out.LogSource)) {
	case "oauth-apiserver", "oauth_api_server", "oauthserver":
		out.LogSource = "oauth-server"
	case "openshiftapiserver", "openshift_api_server":
		out.LogSource = "openshift-apiserver"
	case "kube_api_server", "kubeapiserver":
		out.LogSource = "kube-apiserver"
	}

	// Normalize verb synonyms/casing
	normalizeListValues := func(sa types.StringOrArray, mapping map[string]string) types.StringOrArray {
		if sa.IsString() {
			v := strings.ToLower(strings.TrimSpace(sa.GetString()))
			if nv, ok := mapping[v]; ok {
				return *types.NewStringOrArray(nv)
			}
			return *types.NewStringOrArray(v)
		}
		if arr := sa.GetArray(); arr != nil {
			res := make([]string, 0, len(arr))
			for _, s := range arr {
				v := strings.ToLower(strings.TrimSpace(s))
				if nv, ok := mapping[v]; ok {
					res = append(res, nv)
				} else if v != "" {
					res = append(res, v)
				}
			}
			return *types.NewStringOrArray(res)
		}
		return sa
	}

	verbMap := map[string]string{"create": "create", "post": "create", "get": "get", "read": "get", "list": "list", "update": "update", "patch": "patch", "delete": "delete"}
	out.Verb = normalizeListValues(out.Verb, verbMap)

	// Normalize response status values like "200 ok" -> "200"
	statusMap := map[string]string{"ok": "200"}
	out.ResponseStatus = normalizeListValues(out.ResponseStatus, statusMap)

	return &out, nil
}
