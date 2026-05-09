#!/bin/bash

# ==============================================================================
# KGS Platform API Test Script
# 
# Requires: python3 or jq (for JSON parsing if needed, but not strictly required 
# if we just want to output the response). We will use jq if available.
#
# Usage:
#   ./test_api.sh [BASE_URL]
#   Example: ./test_api.sh http://localhost:8000
# ==============================================================================

BASE_URL=${1:-"http://localhost:8000"}

bold=$(tput bold 2>/dev/null || echo "")
green=$(tput setaf 2 2>/dev/null || echo "")
red=$(tput setaf 1 2>/dev/null || echo "")
reset=$(tput sgr0 2>/dev/null || echo "")

print_heading() {
    echo -e "\n${bold}======================================================================${reset}"
    echo -e "${bold}$1${reset}"
    echo -e "${bold}======================================================================${reset}"
}

print_subheading() {
    echo -e "\n${bold}▶ $1${reset}"
}

# Check if jq is installed for variable extraction
if ! command -v jq &> /dev/null; then
    echo -e "${red}Warning: 'jq' is not installed. Some automated ID extractions might fail.${reset}"
    echo "You can install it with: sudo apt install jq / brew install jq"
    HAS_JQ=false
else
    HAS_JQ=true
fi

echo -e "Starting API tests against BASE_URL: ${bold}$BASE_URL${reset}"

# ------------------------------------------------------------------------------
print_heading "1. App Registry Service"
# ------------------------------------------------------------------------------

print_subheading "1.1 Create Application"
CREATE_APP_RESP=$(curl -s -X POST "$BASE_URL/v1/apps" \
  -H "Content-Type: application/json" \
  -d '{
    "app_name": "Test Application",
    "description": "Application created by automated test script",
    "owner": "admin"
  }')
echo "$CREATE_APP_RESP" | jq . 2>/dev/null || echo "$CREATE_APP_RESP"

if [ "$HAS_JQ" = true ]; then
    APP_ID=$(echo "$CREATE_APP_RESP" | jq -r '.app_id | select(.!=null)')
    if [ -z "$APP_ID" ] || [ "$APP_ID" == "null" ]; then
        echo -e "${red}Failed to extract APP_ID. Using fallback 'dummy_app_id'${reset}"
        APP_ID="dummy_app_id"
    else
        echo -e "${green}Successfully created App with ID: $APP_ID${reset}"
    fi
else
    APP_ID="<YOUR_APP_ID>"
fi

print_subheading "1.2 List Applications"
curl -s -X GET "$BASE_URL/v1/apps" -H "Accept: application/json" | jq . 2>/dev/null || curl -s -X GET "$BASE_URL/v1/apps"

print_subheading "1.3 Get Application Details"
curl -s -X GET "$BASE_URL/v1/apps/$APP_ID" -H "Accept: application/json" | jq . 2>/dev/null || curl -s -X GET "$BASE_URL/v1/apps/$APP_ID"

print_subheading "1.4 Issue API Key"
ISSUE_KEY_RESP=$(curl -s -X POST "$BASE_URL/v1/apps/$APP_ID/keys" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test API Key",
    "scopes": "all",
    "ttl_seconds": 3600
  }')
echo "$ISSUE_KEY_RESP" | jq . 2>/dev/null || echo "$ISSUE_KEY_RESP"

if [ "$HAS_JQ" = true ]; then
    API_KEY=$(echo "$ISSUE_KEY_RESP" | jq -r '.api_key | select(.!=null)')
    KEY_HASH=$(echo "$ISSUE_KEY_RESP" | jq -r '.key_hash | select(.!=null)')
    if [ -z "$API_KEY" ] || [ "$API_KEY" == "null" ]; then
        echo -e "${red}Failed to extract API_KEY. Tests requiring auth might fail.${reset}"
        API_KEY="dummy_api_key"
        KEY_HASH="dummy_key_hash"
    else
        echo -e "${green}Successfully generated API Key: $API_KEY${reset}"
    fi
else
    API_KEY="<YOUR_API_KEY>"
    KEY_HASH="<YOUR_KEY_HASH>"
fi

AUTH_HEADER="Authorization: Bearer $API_KEY"

# ------------------------------------------------------------------------------
print_heading "2. Ontology Service"
# ------------------------------------------------------------------------------

print_subheading "2.1 Create Entity Type"
curl -s -X POST "$BASE_URL/v1/ontology/entities" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "name": "User",
    "description": "A system user",
    "schema": "{\"type\":\"object\",\"properties\":{\"username\":{\"type\":\"string\"},\"age\":{\"type\":\"integer\"}}}"
  }' | jq . 2>/dev/null || echo "Done"

print_subheading "2.2 Create Relation Type"
curl -s -X POST "$BASE_URL/v1/ontology/relations" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "name": "FRIEND_OF",
    "description": "Represents a friendship between users",
    "properties_schema": "{\"type\":\"object\",\"properties\":{\"since_year\":{\"type\":\"integer\"}}}",
    "source_types": ["User"],
    "target_types": ["User"]
  }' | jq . 2>/dev/null || echo "Done"


# ------------------------------------------------------------------------------
print_heading "3. Graph Service"
# ------------------------------------------------------------------------------

print_subheading "3.1 Create Source Node"
CREATE_NODE1_RESP=$(curl -s -X POST "$BASE_URL/v1/graph/nodes" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "label": "User",
    "properties_json": "{\"username\": \"alice123\", \"age\": 25}"
  }')
echo "$CREATE_NODE1_RESP" | jq . 2>/dev/null || echo "$CREATE_NODE1_RESP"

print_subheading "3.1 Create Target Node"
CREATE_NODE2_RESP=$(curl -s -X POST "$BASE_URL/v1/graph/nodes" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "label": "User",
    "properties_json": "{\"username\": \"bob456\", \"age\": 28}"
  }')
echo "$CREATE_NODE2_RESP" | jq . 2>/dev/null || echo "$CREATE_NODE2_RESP"

if [ "$HAS_JQ" = true ]; then
    NODE1_ID=$(echo "$CREATE_NODE1_RESP" | jq -r '.id // .node_id // .uid | select(.!=null)')
    NODE2_ID=$(echo "$CREATE_NODE2_RESP" | jq -r '.id // .node_id // .uid | select(.!=null)')
else
    NODE1_ID="<SOURCE_NODE_ID>"
    NODE2_ID="<TARGET_NODE_ID>"
fi

print_subheading "3.2 Get Node Data (Node 1)"
if [ -n "$NODE1_ID" ] && [ "$NODE1_ID" != "null" ]; then
    curl -s -X GET "$BASE_URL/v1/graph/nodes/$NODE1_ID" \
      -H "$AUTH_HEADER" | jq . 2>/dev/null || echo "Done"
else
    echo "Skipped: Node ID is null or missing"
fi

print_subheading "3.3 Create Edge"
if [ -n "$NODE1_ID" ] && [ "$NODE1_ID" != "null" ] && [ -n "$NODE2_ID" ] && [ "$NODE2_ID" != "null" ]; then
    curl -s -X POST "$BASE_URL/v1/graph/edges" \
      -H "Content-Type: application/json" \
      -H "$AUTH_HEADER" \
      -d '{
        "source_node_id": "'"$NODE1_ID"'",
        "target_node_id": "'"$NODE2_ID"'",
        "relation_type": "FRIEND_OF",
        "properties_json": "{\"since_year\": 2023}"
      }' | jq . 2>/dev/null || echo "Done"
else
    echo "Skipped: Missing Source/Target Node IDs"
fi

print_subheading "3.4 Get Context (Neighborhood)"
if [ -n "$NODE1_ID" ] && [ "$NODE1_ID" != "null" ]; then
    curl -s -X GET "$BASE_URL/v1/graph/nodes/$NODE1_ID/context?depth=1&direction=BOTH" \
      -H "$AUTH_HEADER" | jq . 2>/dev/null || echo "Done"
else
    echo "Skipped: Node ID is null or missing"
fi

print_subheading "3.5 Get Impact (Downstream)"
if [ -n "$NODE1_ID" ] && [ "$NODE1_ID" != "null" ]; then
    curl -s -X GET "$BASE_URL/v1/graph/nodes/$NODE1_ID/impact?max_depth=3" \
      -H "$AUTH_HEADER" | jq . 2>/dev/null || echo "Done"
else
    echo "Skipped: Node ID is null or missing"
fi

print_subheading "3.6 Get Coverage (Upstream)"
if [ -n "$NODE2_ID" ] && [ "$NODE2_ID" != "null" ]; then
    curl -s -X GET "$BASE_URL/v1/graph/nodes/$NODE2_ID/coverage" \
      -H "$AUTH_HEADER" | jq . 2>/dev/null || echo "Done"
else
    echo "Skipped: Node ID is null or missing"
fi


# ------------------------------------------------------------------------------
print_heading "4. Rule Engine"
# ------------------------------------------------------------------------------

print_subheading "4.1 Create Rule"
curl -s -X POST "$BASE_URL/v1/rules" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "name": "Auto-Friendship Enrichment",
    "description": "Automatically update friendship properties",
    "trigger_type": "SCHEDULED",
    "cron": "0 0 * * *",
    "cypher_query": "MATCH (a:User)-[r:FRIEND_OF]->(b:User) RETURN a,r,b",
    "action": "LOG",
    "payload_json": "{}"
  }' | jq . 2>/dev/null || echo "Done"


# ------------------------------------------------------------------------------
print_heading "5. Access Control"
# ------------------------------------------------------------------------------

print_subheading "5.1 Create Policy"
curl -s -X POST "$BASE_URL/v1/policies" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "name": "Node Read Access",
    "description": "Allow reading nodes if user has read scope",
    "rego_content": "package kgs.authz\n\ndefault allow = false\n\nallow {\n  input.user.scopes[_] == \"read\"\n}"
  }' | jq . 2>/dev/null || echo "Done"

print_subheading "5.2 List Policies"
curl -s -X GET "$BASE_URL/v1/policies" \
  -H "$AUTH_HEADER" | jq . 2>/dev/null || echo "Done"


# ------------------------------------------------------------------------------
print_heading "Cleanup (App Registry)"
# ------------------------------------------------------------------------------

print_subheading "1.5 Revoke API Key"
if [ -n "$KEY_HASH" ] && [ "$KEY_HASH" != "null" ]; then
    curl -s -X DELETE "$BASE_URL/v1/keys/$KEY_HASH" \
      -H "$AUTH_HEADER" | jq . 2>/dev/null || echo "Done"
else
    echo "Skipped: Key Hash is null or missing"
fi

echo -e "\n${bold}API Tests Completed!${reset}"
