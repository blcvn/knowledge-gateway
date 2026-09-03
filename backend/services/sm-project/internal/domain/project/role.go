package project

import (
\t"errors"
\t"time"
)

// Role defines the access level a user has within a Space.
type Role string

const (
\tRoleOwner  Role = "OWNER"
\tRoleEditor Role = "EDITOR"
\tRoleViewer Role = "VIEWER"
)

var (
\tErrInvalidRole   = errors.New("invalid role specified")
\tErrAccessDenied  = errors.New("access denied: insufficient permissions")
)

// Permission specifies an action that can be performed in a Space.
type Permission string

const (
\tPermCreateDocument Permission = "document:create"
\tPermReadDocument   Permission = "document:read"
\tPermDeleteDocument Permission = "document:delete"
\tPermManageMembers  Permission = "member:manage"
\tPermDeleteSpace    Permission = "space:delete"
)

// rbacMatrix maps each Role to its allowed Permissions.
var rbacMatrix = map[Role]map[Permission]bool{
\tRoleOwner: {
\t\tPermCreateDocument: true,
\t\tPermReadDocument:   true,
\t\tPermDeleteDocument: true,
\t\tPermManageMembers:  true,
\t\tPermDeleteSpace:    true,
\t},
\tRoleEditor: {
\t\tPermCreateDocument: true,
\t\tPermReadDocument:   true,
\t\tPermDeleteDocument: false, // Editors cannot delete documents
\t\tPermManageMembers:  false,
\t\tPermDeleteSpace:    false,
\t},
\tRoleViewer: {
\t\tPermCreateDocument: false,
\t\tPermReadDocument:   true,
\t\tPermDeleteDocument: false,
\t\tPermManageMembers:  false,
\t\tPermDeleteSpace:    false,
\t},
}

// HasPermission checks if the given role has the right to execute a specific permission.
func (r Role) HasPermission(perm Permission) bool {
\tpermissions, exists := rbacMatrix[r]
\tif !exists {
\t\treturn false
\t}
\treturn permissions[perm]
}

// Validate ensures the role string is a known Role.
func (r Role) Validate() error {
\tswitch r {
\tcase RoleOwner, RoleEditor, RoleViewer:
\t\treturn nil
\t}
\treturn ErrInvalidRole
}

// SpaceMember represents a user's membership and role within a Space.
type SpaceMember struct {
\tUserID    string
\tSpaceID   string
\tRole      Role
\tJoinedAt  time.Time
}

// NewSpaceMember creates a new member object and validates the role.
func NewSpaceMember(userID, spaceID string, role Role) (*SpaceMember, error) {
\tif err := role.Validate(); err != nil {
\t\treturn nil, err
\t}
\t
\treturn &SpaceMember{
\t\tUserID:   userID,
\t\tSpaceID:  spaceID,
\t\tRole:     role,
\t\tJoinedAt: time.Now().UTC(),
\t}, nil
}
