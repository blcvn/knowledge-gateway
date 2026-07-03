## MODIFIED Requirements

### Requirement: Sync is idempotent by external_ref

Codegraph sync SHALL gom toàn bộ source mutations của cùng một graph interaction vào một sync
session trước khi handoff projection.

#### Scenario: Một lần SyncProject tạo đúng một version change và một event

- GIVEN `SyncProject` chạy cho `project_id = repo-a`
- AND reconcile của lần chạy đó tạo node upserts, relationship upserts, và stale relationship
  deletes cho cùng graph scope
- WHEN sync run hoàn tất thành công
- THEN bridge SHALL commit đúng một sync session cho `graph_scope = project:repo-a`
- AND KG Service SHALL tạo đúng một graph version change cho run đó
- AND KG Service SHALL phát đúng một `GRAPH_VERSION_SEALED` event cho run đó

#### Scenario: Scope conflict không tạo partial event stream

- GIVEN một sync run thứ hai cố mở session trên cùng `graph_scope` khi session trước còn active
- WHEN KG Service trả `SYNC_SCOPE_LOCKED`
- THEN bridge SHALL fail run đó rõ ràng
- AND bridge SHALL NOT fallback sang legacy bulk writes phát nhiều outbox events riêng lẻ
