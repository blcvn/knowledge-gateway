package biz

import "strings"

func ComputeNamespace(appID, tenantID string, orgID ...string) string {
	appID = strings.TrimSpace(appID)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = "default"
	}
	if len(orgID) > 0 {
		if oid := strings.TrimSpace(orgID[0]); oid != "" {
			return "graph/" + oid + "/" + appID + "/" + tenantID
		}
	}
	return "graph/" + appID + "/" + tenantID
}
