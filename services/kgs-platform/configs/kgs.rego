package kgs

import rego.v1

# Default deny
default allow := false

# Allow demo-app (for testing)
allow if {
	input.app_id == "demo-app"
}

# Allow system-level access (used by kgs-platform internal operations)
allow if {
	input.app_id == "system"
}

# Allow any app_id with valid tenant__project format
# Pattern: <tenant_id>__<project_uuid>  e.g. "default__00fbfbd2-e420-4ee6-96ca-6267c145407d"
allow if {
	contains(input.app_id, "__")
	count(input.app_id) > 3
}
