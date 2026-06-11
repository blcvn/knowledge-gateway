package overlay

import "fmt"

const (
	PolicyKeepOverlay   = "KEEP_OVERLAY"
	PolicyKeepBase      = "KEEP_BASE"
	PolicyMerge         = "MERGE"
	PolicyRequireManual = "REQUIRE_MANUAL"
)

type Conflict struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

var ErrOverlayConflict = fmt.Errorf("overlay conflicts require manual resolution")

func DetectConflicts(baseVersionID, latestVersionID string) []Conflict {
	if baseVersionID == "" || latestVersionID == "" || baseVersionID == latestVersionID {
		return nil
	}
	return []Conflict{
		{
			Type:    "BASE_DRIFT",
			Message: "base version changed during overlay lifetime",
		},
	}
}

func ResolveConflicts(policy string, conflicts []Conflict) (int, error) {
	if len(conflicts) == 0 {
		return 0, nil
	}
	switch policy {
	case "", PolicyKeepOverlay, PolicyKeepBase, PolicyMerge:
		return len(conflicts), nil
	case PolicyRequireManual:
		return 0, ErrOverlayConflict
	default:
		return 0, fmt.Errorf("unknown conflict policy: %s", policy)
	}
}
