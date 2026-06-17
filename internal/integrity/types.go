package integrity

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
