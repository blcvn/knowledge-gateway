// Package domain defines Ontology types for cognee-cognify schema management.
package domain

import "github.com/google/uuid"

// OntologyCategory defines a class in the ontology schema.
type OntologyCategory struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parents     []string `json:"parents"`
}

// Ontology is a named schema defining entity types and their hierarchy.
type Ontology struct {
	ID         uuid.UUID          `json:"id"`
	TenantID   string             `json:"tenant_id"`
	Name       string             `json:"name"`
	Categories []OntologyCategory `json:"categories"`
}
