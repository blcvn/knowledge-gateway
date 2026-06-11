package data

import (
	"strings"
)

const (
	TopicOverlayCommittedPrefix = "overlay.committed"
	TopicOverlayDiscardedPrefix = "overlay.discarded"
	TopicSessionClosePrefix     = "session.close"
	TopicBudgetStopPrefix       = "budget.stop"
)

func TopicOverlayCommitted(namespace string) string {
	return TopicOverlayCommittedPrefix + "." + sanitizeTopicToken(extractTenantID(namespace))
}

func TopicOverlayDiscarded(namespace string) string {
	return TopicOverlayDiscardedPrefix + "." + sanitizeTopicToken(extractTenantID(namespace))
}

func TopicSessionClose(sessionID string) string {
	return TopicSessionClosePrefix + "." + sanitizeTopicToken(sessionID)
}

func TopicBudgetStop(sessionID string) string {
	return TopicBudgetStopPrefix + "." + sanitizeTopicToken(sessionID)
}

func TopicSessionClosePattern() string {
	return TopicSessionClosePrefix + ".*"
}

func TopicBudgetStopPattern() string {
	return TopicBudgetStopPrefix + ".*"
}

func extractTenantID(namespace string) string {
	parts := strings.Split(strings.Trim(namespace, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	if len(parts) >= 2 {
		return parts[1]
	}
	return "default"
}

func sanitizeTopicToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	value = strings.ReplaceAll(value, ".", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}
