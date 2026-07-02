## MODIFIED Requirements

### Requirement: Per-entity sync status API qua Admin MCP

Hệ thống SHALL cho phép query trạng thái sync của một entity cụ thể qua Admin MCP tool.

#### Scenario: Query entity đã sync đầy đủ ở graph head tương ứng

- GIVEN node `node-abc` thuộc logical graph `kg-123`
- AND graph backend head của `kg-123` đã apply version liên quan tới `node-abc`
- AND vector backend head tương ứng cũng đã apply version đó
- WHEN admin query `EntitySyncStatus("node-abc", "kg_node")`
- THEN response SHALL có `GraphLagClass="SYNCED"` và `VectorLagClass="SYNCED"`

#### Scenario: Query entity đang in-flight trước khi graph head advance

- GIVEN node `node-xyz` thuộc logical graph `kg-123`
- AND logical graph đó đã có source version mới được seal hoặc đang được project
- AND graph backend head của `kg-123` chưa advance tới version liên quan
- WHEN admin query `EntitySyncStatus("node-xyz", "kg_node")`
- THEN response SHALL có một trạng thái graph non-synced biểu thị "đang sync"
- AND `LastGraphSyncedAt` SHALL không phải zero value

#### Scenario: Query entity bị stuck

- GIVEN node `node-zzz` thuộc logical graph `kg-123`
- AND graph backend head của `kg-123` không advance dù version liên quan đã vượt retry hoặc lag threshold
- WHEN admin query `EntitySyncStatus("node-zzz", "kg_node")`
- THEN response SHALL có `GraphLagClass="STUCK"`

#### Scenario: Query entity does not report SYNCED when graph head has not made the node queryable

- GIVEN node `node-probe` vẫn tồn tại trong `relationshipdb`
- AND some node-level graph version signal appears current for `node-probe`
- AND graph backend head for the relevant logical graph has not yet proven the version queryable
- AND a graph projection probe consistent with non-realtime read still cannot read `node-probe`
- WHEN admin query `EntitySyncStatus("node-probe", "kg_node")`
- THEN response SHALL NOT có `GraphLagClass="SYNCED"`
- AND response SHALL expose enough detail for operators or validation to identify the projection as unreadable
