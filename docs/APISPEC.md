# KGS Platform API Specification (v1)

This document provides the formal specification for the Knowledge Graph Service (KGS) Platform APIs. All APIs follow a RESTful style over gRPC/HTTP and use JSON for payloads.

## Base URL
`http://<kgs-platform-host>:<port>`

## Authentication
Most endpoints require an API Key passed in the `Authorization` header:
`Authorization: Bearer <kgs_api_key>`

---

## 1. App Registry Service
Manage applications, quotas, and API keys.

### 1.1 Create Application
- **Method**: `POST`
- **Endpoint**: `/v1/apps`
- **Request Body**:
  ```json
  {
    "app_name": "string",
    "description": "string",
    "owner": "string"
  }
  ```
- **Success Response** (200 OK):
  ```json
  {
    "app_id": "string",
    "status": "string"
  }
  ```

### 1.2 Get Application Details
- **Method**: `GET`
- **Endpoint**: `/v1/apps/{app_id}`
- **Success Response** (200 OK):
  ```json
  {
    "app_id": "string",
    "app_name": "string",
    "description": "string",
    "owner": "string",
    "status": "string"
  }
  ```

### 1.3 List Applications
- **Method**: `GET`
- **Endpoint**: `/v1/apps`
- **Success Response** (200 OK):
  ```json
  {
    "apps": [ ...GetAppReply... ]
  }
  ```

### 1.4 Issue API Key
- **Method**: `POST`
- **Endpoint**: `/v1/apps/{app_id}/keys`
- **Request Body**:
  ```json
  {
    "name": "string",
    "scopes": "string",
    "ttl_seconds": 0
  }
  ```
- **Success Response** (200 OK):
  ```json
  {
    "api_key": "string",
    "key_hash": "string",
    "key_prefix": "string"
  }
  ```
  *Note: `api_key` is only returned once upon creation.*

### 1.5 Revoke API Key
- **Method**: `DELETE`
- **Endpoint**: `/v1/keys/{key_hash}`
- **Success Response** (200 OK):
  ```json
  {
    "success": true
  }
  ```

---

## 2. Ontology Service
Define the schema (Entity and Relation types) for your knowledge graph.

### 2.1 Create Entity Type
- **Method**: `POST`
- **Endpoint**: `/v1/ontology/entities`
- **Request Body**:
  ```json
  {
    "name": "string",
    "description": "string",
    "schema": "string (JSON Schema)"
  }
  ```
- **Success Response** (200 OK):
  ```json
  {
    "id": 1,
    "name": "string",
    "status": "string"
  }
  ```

### 2.2 Create Relation Type
- **Method**: `POST`
- **Endpoint**: `/v1/ontology/relations`
- **Request Body**:
  ```json
  {
    "name": "string",
    "description": "string",
    "properties_schema": "string (JSON Schema)",
    "source_types": ["string"],
    "target_types": ["string"]
  }
  ```
- **Success Response** (200 OK):
  ```json
  {
    "id": 1,
    "name": "string",
    "status": "string"
  }
  ```

---

## 3. Graph Service
Core operations for interacting with nodes and edges in the Knowledge Graph.

### 3.1 Create Node
- **Method**: `POST`
- **Endpoint**: `/v1/graph/nodes`
- **Request Body**:
  ```json
  {
    "label": "string",
    "properties_json": "string (JSON string)"
  }
  ```

### 3.2 Get Node
- **Method**: `GET`
- **Endpoint**: `/v1/graph/nodes/{node_id}`

### 3.3 Create Edge
- **Method**: `POST`
- **Endpoint**: `/v1/graph/edges`
- **Request Body**:
  ```json
  {
    "source_node_id": "uuid",
    "target_node_id": "uuid",
    "relation_type": "string",
    "properties_json": "string (JSON string)"
  }
  ```

### 3.4 Get Context (Neighborhood)
Fetch local subgraph around a node.
- **Method**: `GET`
- **Endpoint**: `/v1/graph/nodes/{node_id}/context`
- **Query Parameters**:
  - `depth`: int (default 1)
  - `direction`: "INCOMING", "OUTGOING", "BOTH"

### 3.5 Get Impact (Downstream)
- **Method**: `GET`
- **Endpoint**: `/v1/graph/nodes/{node_id}/impact`
- **Query Parameters**:
  - `max_depth`: int

### 3.6 Get Coverage (Upstream)
- **Method**: `GET`
- **Endpoint**: `/v1/graph/nodes/{node_id}/coverage`

---

## 4. Rule Engine
Manage business rules for graph enrichment and consistency.

### 4.1 Create Rule
- **Method**: `POST`
- **Endpoint**: `/v1/rules`
- **Request Body**:
  ```json
  {
    "name": "string",
    "description": "string",
    "trigger_type": "SCHEDULED | ON_WRITE",
    "cron": "string (optional)",
    "cypher_query": "string",
    "action": "string",
    "payload_json": "string"
  }
  ```

---

## 5. Access Control
Manage OPA policies for attribute-based access control.

### 5.1 Create Policy
- **Method**: `POST`
- **Endpoint**: `/v1/policies`
- **Request Body**:
  ```json
  {
    "name": "string",
    "description": "string",
    "rego_content": "string (Rego code)"
  }
  ```

### 5.2 List Policies
- **Method**: `GET`
- **Endpoint**: `/v1/policies`
