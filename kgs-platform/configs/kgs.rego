package kgs

import rego.v1

# Default deny
default allow := false

# Allow any registered app (dev mode - no app_id restriction)
allow if {
	input.app_id != ""
}

# Fallback: allow demo-app explicitly for backwards-compat
allow if {
	input.app_id == "demo-app"
}

# In a real environment, validation logic could be more complex:
# allow if {
# 	user_has_role("admin")
# }
# 
# allow if {
# 	input.action == "CREATE_NODE"
# 	input.resource == "Person"
# 	user_has_permission("write:person")
# }
