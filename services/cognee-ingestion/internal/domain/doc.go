// Package domain defines the core entities, value objects, events, and errors
// for the cognee-ingestion service.
//
// This package has ZERO external dependencies beyond the Go stdlib and uuid.
// It represents the innermost ring of Clean Architecture and must not import
// any adapter, infrastructure, or framework code.
package domain
