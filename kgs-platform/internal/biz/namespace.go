package biz

import "strings"

func ComputeNamespace(appID, tenantID string) string {
	appID = strings.TrimSpace(appID)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = "default"
	}
	return "graph/" + appID + "/" + tenantID
}
