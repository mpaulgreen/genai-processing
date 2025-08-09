package extractors

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// ExtractorFactory provides thread-safe registration and creation of parsers
// for different model types with alias support and generic fallback.
type ExtractorFactory struct {
	mu      sync.RWMutex
	byType  map[string]interfaces.Parser
	aliases map[string]string
	generic interfaces.Parser
}

func NewExtractorFactory() *ExtractorFactory {
	return &ExtractorFactory{byType: make(map[string]interfaces.Parser), aliases: make(map[string]string)}
}

// Register adds a parser for a modelType and optional aliases.
func (f *ExtractorFactory) Register(modelType string, parser interfaces.Parser, aliases ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(modelType))
	f.byType[key] = parser
	for _, a := range aliases {
		a = strings.ToLower(strings.TrimSpace(a))
		if a != "" {
			f.aliases[a] = key
		}
	}
}

// SetGeneric sets the generic fallback parser.
func (f *ExtractorFactory) SetGeneric(parser interfaces.Parser) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.generic = parser
}

// CreateExtractor returns a parser for the given modelType, resolving aliases
// and falling back to the generic parser if available.
func (f *ExtractorFactory) CreateExtractor(modelType string) (interfaces.Parser, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(modelType))
	if key == "" {
		if f.generic != nil {
			return f.generic, nil
		}
		return nil, fmt.Errorf("no model type provided and no generic parser configured")
	}
	if t, ok := f.aliases[key]; ok {
		key = t
	}
	if p, ok := f.byType[key]; ok {
		return p, nil
	}
	if f.generic != nil {
		return f.generic, nil
	}
	return nil, fmt.Errorf("unsupported model type: %s", modelType)
}

// CreateDelegatingParser returns a parser that delegates to a registered
// concrete parser at parse time, defaulting to generic.
func (f *ExtractorFactory) CreateDelegatingParser() interfaces.Parser {
	f.mu.RLock()
	defer f.mu.RUnlock()
	// Build a snapshot to avoid locking during parse
	snapshot := make(map[string]interfaces.Parser, len(f.byType))
	for k, v := range f.byType {
		snapshot[k] = v
	}
	generic := f.generic

	return &delegatingParser{byType: snapshot, generic: generic}
}

// GetSupportedModelTypes returns sorted keys for deterministic output.
func (f *ExtractorFactory) GetSupportedModelTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	keys := make([]string, 0, len(f.byType))
	for k := range f.byType {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// delegatingParser implements interfaces.Parser by delegating based on modelType.
type delegatingParser struct {
	byType  map[string]interfaces.Parser
	generic interfaces.Parser
}

func (d *delegatingParser) ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error) {
	key := strings.ToLower(strings.TrimSpace(modelType))
	if p, ok := d.byType[key]; ok && p.CanHandle(modelType) {
		return p.ParseResponse(raw, modelType)
	}
	if d.generic != nil {
		return d.generic.ParseResponse(raw, modelType)
	}
	return nil, fmt.Errorf("no parser available for model type: %s", modelType)
}

func (d *delegatingParser) CanHandle(modelType string) bool {
	key := strings.ToLower(strings.TrimSpace(modelType))
	if p, ok := d.byType[key]; ok && p != nil {
		return true
	}
	return d.generic != nil
}

func (d *delegatingParser) GetConfidence() float64 {
	// Return conservative default; caller should use underlying parser's confidence
	return 0.8
}
