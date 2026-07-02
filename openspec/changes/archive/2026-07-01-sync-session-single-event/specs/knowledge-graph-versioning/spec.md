## MODIFIED Requirements

### Requirement: Version sealing SHALL là durable handoff cho async projection

Hệ thống SHALL cho phép một session write nhiều bước materialize nhiều source mutations vào
cùng một `graph_version_id`, nhưng durable handoff sang worker vẫn chỉ xảy ra đúng một lần khi
session đó được commit thành công.

#### Scenario: Một sync session commit phát đúng một graph-version event

- GIVEN một sync session của graph `kg-123` đã append nhiều node upserts, relationship upserts,
  và relationship deletes vào cùng một change set
- WHEN caller commit session đó thành công
- THEN hệ thống SHALL seal đúng một graph version mới cho interaction đó
- AND hệ thống SHALL phát đúng một `GRAPH_VERSION_SEALED` outbox event cho version đó
- AND hệ thống SHALL NOT phát thêm node-level hay relationship-level outbox event cho các
  mutations đã thuộc cùng session commit

#### Scenario: Session chưa commit không được phát graph-version event

- GIVEN một sync session của graph `kg-123` đang ở trạng thái `PENDING_ENTITIES`
- AND source mutations đã được ghi theo session mode
- WHEN session đó chưa được commit hoặc bị abandon
- THEN hệ thống SHALL NOT phát `GRAPH_VERSION_SEALED` event cho session đó

#### Scenario: Một interaction chứa stale relationship deletes vẫn chỉ có một event

- GIVEN một lần reconcile của graph `kg-123` phát hiện cả relationship upserts và stale
  relationship deletes
- WHEN các mutations đó đều được append vào cùng một sync session
- THEN change set của graph version đó SHALL chứa cả `UPSERT` và `DELETE` entity entries
- AND khi commit thành công hệ thống vẫn SHALL chỉ phát đúng một `GRAPH_VERSION_SEALED` event
