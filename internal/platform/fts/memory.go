package fts

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"unicode"
)

type InMemoryFTSAdapter struct {
	mu   sync.RWMutex
	docs map[string]FTSDocument
}

func NewInMemoryFTSAdapter() *InMemoryFTSAdapter {
	return &InMemoryFTSAdapter{docs: map[string]FTSDocument{}}
}

func (a *InMemoryFTSAdapter) Index(_ context.Context, doc FTSDocument) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.docs[doc.NodeID] = doc
	return nil
}

func (a *InMemoryFTSAdapter) Delete(_ context.Context, nodeID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.docs, nodeID)
	return nil
}

func (a *InMemoryFTSAdapter) Search(_ context.Context, query FTSQuery, filter FTSFilter, opts SearchOptions) ([]FTSResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	results := make([]FTSResult, 0, len(a.docs))
	for _, doc := range a.docs {
		if doc.IsDeleted {
			continue
		}
		if len(filter.DomainIDs) > 0 && !contains(filter.DomainIDs, doc.DomainID) {
			continue
		}
		if len(filter.NodeTypes) > 0 && !contains(filter.NodeTypes, doc.NodeType) {
			continue
		}
		if len(filter.OwnerTenantIDs) > 0 && !contains(filter.OwnerTenantIDs, doc.OwnerTenantID) {
			continue
		}
		if len(filter.OwnerAppIDs) > 0 && !contains(filter.OwnerAppIDs, doc.OwnerAppID) {
			continue
		}
		if len(filter.ACLVisibleTo) > 0 && !intersects(filter.ACLVisibleTo, doc.ACLVisibleTo) {
			continue
		}
		if !matchesFTS(doc, query) {
			continue
		}
		score := scoreFTS(doc, query)
		results = append(results, FTSResult{Document: doc, Score: score})
	}

	slices.SortFunc(results, func(a, b FTSResult) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		return strings.Compare(a.Document.NodeID, b.Document.NodeID)
	})
	if opts.TopK > 0 && len(results) > opts.TopK {
		results = results[:opts.TopK]
	}
	return results, nil
}

func matchesFTS(doc FTSDocument, query FTSQuery) bool {
	text := selectedText(doc, query.Fields)
	if text == "" {
		return false
	}
	switch normalizeMode(query.Mode) {
	case "any_token":
		for _, token := range tokenize(query.Text) {
			if strings.Contains(text, token) {
				return true
			}
		}
		return false
	case "phrase":
		return strings.Contains(text, normalizeText(query.Text))
	default:
		for _, token := range tokenize(query.Text) {
			if !strings.Contains(text, token) {
				return false
			}
		}
		return len(tokenize(query.Text)) > 0
	}
}

func scoreFTS(doc FTSDocument, query FTSQuery) float64 {
	text := selectedText(doc, query.Fields)
	if text == "" {
		return 0
	}
	tokens := tokenize(query.Text)
	if len(tokens) == 0 {
		return 0
	}
	switch normalizeMode(query.Mode) {
	case "any_token":
		matched := 0
		for _, token := range tokens {
			if strings.Contains(text, token) {
				matched++
			}
		}
		return float64(matched) / float64(len(tokens))
	case "phrase":
		if strings.Contains(text, normalizeText(query.Text)) {
			return 1
		}
		return 0
	default:
		matched := 0
		for _, token := range tokens {
			if strings.Contains(text, token) {
				matched++
			}
		}
		return math.Min(1, float64(matched)/float64(len(tokens)))
	}
}

func selectedText(doc FTSDocument, fields []string) string {
	if len(fields) == 0 {
		parts := make([]string, 0, len(doc.Fields)+len(doc.DomainProps))
		fieldKeys := make([]string, 0, len(doc.Fields))
		for key := range doc.Fields {
			fieldKeys = append(fieldKeys, key)
		}
		slices.Sort(fieldKeys)
		for _, key := range fieldKeys {
			value := doc.Fields[key]
			if strings.TrimSpace(value) != "" {
				parts = append(parts, value)
			}
		}
		propKeys := make([]string, 0, len(doc.DomainProps))
		for key := range doc.DomainProps {
			propKeys = append(propKeys, key)
		}
		slices.Sort(propKeys)
		for _, key := range propKeys {
			if text := strings.TrimSpace(stringify(doc.DomainProps[key])); text != "" {
				parts = append(parts, text)
			}
		}
		return normalizeText(strings.Join(parts, " "))
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if value, ok := doc.Fields[field]; ok && strings.TrimSpace(value) != "" {
			parts = append(parts, value)
			continue
		}
		if value, ok := doc.DomainProps[field]; ok {
			if text := strings.TrimSpace(stringify(value)); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return normalizeText(strings.Join(parts, " "))
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "any_token", "or":
		return "any_token"
	case "phrase":
		return "phrase"
	default:
		return "all_tokens"
	}
}

func normalizeText(text string) string {
	return strings.ToLower(strings.TrimSpace(strings.Join(tokenize(text), " ")))
}

func tokenize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(text)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			tokens = append(tokens, field)
		}
	}
	return tokens
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func intersects(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for _, left := range a {
		for _, right := range b {
			if left == right {
				return true
			}
		}
	}
	return false
}

func stringify(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}
