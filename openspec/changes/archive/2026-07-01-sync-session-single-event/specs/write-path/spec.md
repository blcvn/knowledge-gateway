## MODIFIED Requirements

### Requirement: Couple writes and outbox events atomically

KG Service MUST giữ invariant rằng một data interaction được hoàn tất bằng `CommitSyncSession`
thì tương ứng đúng một durable graph-version handoff.

#### Scenario: Sync session commit là một atomic handoff

- GIVEN một sync session hợp lệ đã materialize source mutations cho cùng một `graph_scope`
- WHEN `CommitSyncSession` thành công
- THEN transaction đó SHALL finalize graph version, persist đúng một `GRAPH_VERSION_SEALED`
  outbox event, và release scope lease trong cùng transaction
- AND caller SHALL quan sát đúng một version change cho interaction đó

#### Scenario: Session-mode writes không phát outbox event riêng

- GIVEN node bulk, relationship bulk, hoặc session-aware relationship delete chạy với
  `graph_version_id`
- WHEN request đó commit source mutation thành công
- THEN request đó SHALL append entities vào graph version đang mở
- AND request đó SHALL NOT tạo outbox event riêng

### Requirement: Preserve graph identity boundaries on source writes

Session write path MUST materialize mutations vào đúng graph scope mà caller đã mở session.

#### Scenario: Session write xác thực graph scope của mutation

- GIVEN caller đã mở sync session với `graph_scope = project:repo-a`
- WHEN caller gửi một session-mode write mang `graph_version_id` của session đó
- THEN service SHALL verify mutation thuộc đúng `graph_scope = project:repo-a`
- AND service SHALL từ chối mutation nếu scope thực tế không khớp session đã mở
