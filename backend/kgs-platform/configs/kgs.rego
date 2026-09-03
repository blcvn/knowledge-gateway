package kgs

import rego.v1

# Default deny
default allow := false

# Allow if app_id is "demo-app" (for testing)
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
