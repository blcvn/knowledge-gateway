package project

import (
	"context"
	"errors"

	"vnp-memory/services/sm-project/internal/domain/model"
)

// SpaceRole represents the level of access a user has in a space.
type SpaceRole string

const (
	RoleOwner  SpaceRole = "owner"
	RoleAdmin  SpaceRole = "admin"
	RoleEditor SpaceRole = "editor"
	RoleViewer SpaceRole = "viewer"
	RoleNone   SpaceRole = "none"
)

// Action represents an operation a user wants to perform on a space.
type Action string

const (
	ActionReadSpace     Action = "read_space"
	ActionAddDocs       Action = "add_documents"
	ActionDeleteDocs    Action = "delete_documents"
	ActionManageMembers Action = "manage_members"
	ActionDeleteSpace   Action = "delete_space"
	ActionChangeSettings Action = "change_settings"
)

// RBACManager handles permission resolution logic based on the Specs.
type RBACManager struct {
	// Usually would inject a repository to fetch member roles here
}

func NewRBACManager() *RBACManager {
	return &RBACManager{}
}

// CheckPermission implements the "RBAC Resolution Algorithm" from the Technical Design.
// Action A by user U on space S.
func (rm *RBACManager) CheckPermission(ctx context.Context, action Action, space *model.Space, userID string, jwtOrgRole string, memberRole SpaceRole) (bool, error) {
	// 1. If S.visibility == public, grant Read to any authenticated user.
	// We assume if this function is called, the user is authenticated.
	// space.Visibility is omitted in the generic struct, assuming space has Visibility property in full model.
	// For simulation, let's treat public action
	if action == ActionReadSpace {
		// Example: if space.Visibility == "public" { return true, nil }
	}

	// 2 & 3. Determine effective role.
	effectiveRole := memberRole
	if effectiveRole == RoleNone || effectiveRole == "" {
		if jwtOrgRole == "owner" {
			effectiveRole = RoleAdmin
		} else {
			effectiveRole = RoleNone
		}
	}

	// 4. Verify Role satisfies the minimum required level from Permissions Matrix.
	switch action {
	case ActionReadSpace:
		return effectiveRole == RoleOwner || effectiveRole == RoleAdmin || effectiveRole == RoleEditor || effectiveRole == RoleViewer, nil
	case ActionAddDocs:
		return effectiveRole == RoleOwner || effectiveRole == RoleAdmin || effectiveRole == RoleEditor, nil
	case ActionDeleteDocs, ActionManageMembers, ActionChangeSettings:
		return effectiveRole == RoleOwner || effectiveRole == RoleAdmin, nil
	case ActionDeleteSpace:
		return effectiveRole == RoleOwner, nil
	default:
		return false, errors.New("unknown action")
	}
}
