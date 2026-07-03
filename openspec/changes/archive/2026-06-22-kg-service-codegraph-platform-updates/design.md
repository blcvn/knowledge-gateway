# Design: KG Service CodeGraph Platform Updates

## Overview

Đây là change tối ưu platform, không phải prerequisite cho baseline `code-graph` integration.

## Candidate Endpoints

- `POST /v1/kg/write/nodes/bulk`
- `POST /v1/kg/write/relationships/bulk`
- `DELETE /v1/kg/write/nodes:by-external-ref-prefix`
- `POST /v1/kg/search/graph`

## Key Decisions

### 1. Additive only

Các endpoint mới không thay thế:

- `/v1/kg/write/nodes`
- `/v1/kg/write/relationships`
- `/v1/kg/read/template/{domain_id}/{template_name}`

### 2. Semantic equivalence

Kết quả và visibility semantics phải nhất quán với core path.

### 3. Raw graph search không được bypass policy

Nếu có `/v1/kg/search/graph`, endpoint này phải reuse auth, ACL, query strategy, và search profile
rules hiện có.
