package interfaces

import "genai-processing/pkg/types"

// Normalizer defines a component that standardizes a StructuredQuery
// according to system-wide conventions.
type Normalizer interface {
	// Normalize standardizes field shapes, defaults, and formats.
	Normalize(query *types.StructuredQuery) (*types.StructuredQuery, error)
}

// FieldMapper defines a component that maps non-canonical or legacy field
// variants into the canonical fields of StructuredQuery.
type FieldMapper interface {
	// MapFields applies mapping rules and returns the mapped query.
	MapFields(query *types.StructuredQuery) (*types.StructuredQuery, error)
}

// SchemaValidator defines structural/schema validation for StructuredQuery.
// It is distinct from safety validation, focusing purely on schema correctness.
type SchemaValidator interface {
	// ValidateSchema validates that the query conforms to type expectations and
	// allowable value ranges.
	ValidateSchema(query *types.StructuredQuery) error
}

// ExtractorFactory creates Parser implementations for different model types
// and can also provide a delegating parser that chooses at parse time.
type ExtractorFactory interface {
	// Register associates a modelType key (and its aliases) with a Parser.
	Register(modelType string, parser Parser, aliases ...string)

	// CreateExtractor returns a Parser for a specific model type or an error if
	// unsupported. Implementations should fall back to a generic parser when
	// reasonable.
	CreateExtractor(modelType string) (Parser, error)

	// CreateDelegatingParser returns a Parser that can handle multiple model
	// types by delegating at parse time.
	CreateDelegatingParser() Parser

	// GetSupportedModelTypes returns the set of registered model type keys.
	GetSupportedModelTypes() []string
}

// FallbackHandler produces a minimal but sensible StructuredQuery when parsing
// fails entirely.
type FallbackHandler interface {
	// CreateMinimalQuery analyzes the raw response and original query to
	// construct a structured fallback query.
	CreateMinimalQuery(raw *types.RawResponse, modelType string, originalQuery string) (*types.StructuredQuery, error)
}
