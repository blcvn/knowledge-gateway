package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type OntologySyncManager struct {
	log *log.Helper
}

// Deprecated: OntologySyncManager is a legacy stub and is no longer wired in runtime DI.
func NewOntologySyncManager(logger log.Logger) *OntologySyncManager {
	return &OntologySyncManager{
		log: log.NewHelper(logger),
	}
}

// SyncConstraints generates constraint cypher queries based on EntityType unique properties
// and syncs them directly into Neo4j in the background.
func (m *OntologySyncManager) SyncConstraints(ctx context.Context) error {
	m.log.Infof("Starting background Ontology constraint synchronization...")

	// 1. Fetch all EntityTypes from Postgres (via a Repo interface to be injected later)
	// 2. Parse JSON schemas to find 'unique' identifiers
	// 3. Generate CREATE CONSTRAINT FOR (n:Label) REQUIRE n.prop IS UNIQUE
	// 4. Run against Neo4j asynchronously

	m.log.Infof("Ontology constraint synchronization complete.")
	return nil
}
