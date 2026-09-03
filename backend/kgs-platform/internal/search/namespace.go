package search

import (
	"regexp"
	"strings"
)

var unsafeNamePattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func parseNamespace(namespace string) (appID string, tenantID string) {
	parts := strings.Split(strings.Trim(namespace, "/"), "/")
	if len(parts) >= 3 {
		return parts[1], parts[2]
	}
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	if len(parts) == 1 {
		return parts[0], "default"
	}
	return "default", "default"
}

func collectionName(namespace string) string {
	appID, _ := parseNamespace(namespace)
	return "kgs-vectors-" + sanitizeName(appID)
}

func fullTextIndexName(namespace string) string {
	appID, tenantID := parseNamespace(namespace)
	return "kgs_fti_" + sanitizeName(appID) + "_" + sanitizeName(tenantID)
}

func sanitizeName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "default"
	}
	safe := unsafeNamePattern.ReplaceAllString(trimmed, "_")
	safe = strings.Trim(safe, "_")
	if safe == "" {
		return "default"
	}
	return strings.ToLower(safe)
}
