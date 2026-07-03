package integrity

import "time"

type CheckResult struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Result int    `json:"result"`
	Status string `json:"status"`
}

type TenantIntegrityResponse struct {
	Checks  []CheckResult `json:"checks"`
	Overall string        `json:"overall"`
}

type MissingBridgeItem struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	DomainID string `json:"domain_id"`
}

type OrphanRelationshipItem struct {
	RelationshipID string `json:"relationship_id"`
	RelType        string `json:"rel_type"`
	FromNodeID     string `json:"from_node_id"`
	ToNodeID       string `json:"to_node_id"`
	DomainID       string `json:"domain_id"`
}

type OrphanVectorDocItem struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	DomainID string `json:"domain_id"`
}

type OrphanScanResponse struct {
	RelationshipOrphans []OrphanRelationshipItem `json:"relationship_orphans"`
	VectorOrphans       []OrphanVectorDocItem    `json:"vector_orphans"`
}

type RepairReport struct {
	RebuiltNodes         int       `json:"rebuilt_nodes"`
	RebuiltRelationships int       `json:"rebuilt_relationships"`
	PurgedRelationships  int       `json:"purged_relationships"`
	PurgedVectorDocs     int       `json:"purged_vector_docs"`
	ScannedAt            time.Time `json:"scanned_at"`
}
