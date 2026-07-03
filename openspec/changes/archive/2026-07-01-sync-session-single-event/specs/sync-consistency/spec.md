## MODIFIED Requirements

### Requirement: Outbox-driven projection runtime batches and coalesces entity work

Projection runtime SHALL coi một committed sync session như một graph-version work unit duy nhất
ở mức outbox.

#### Scenario: Worker nhận một event cho toàn bộ sync interaction

- GIVEN một sync session commit của graph `kg-123` gồm nhiều node upserts, relationship
  upserts, và relationship deletes
- WHEN worker claim outbox
- THEN worker SHALL nhận đúng một `GRAPH_VERSION_SEALED` event cho interaction đó
- AND worker SHALL load entity work từ `kg_graph_version_entities` thay vì từ nhiều legacy
  node/relationship events riêng lẻ

#### Scenario: Bridge relationship không bị project hai lần trong session flow

- GIVEN một node bulk session write tạo bridge relationships embeddable
- WHEN session đó được commit và worker xử lý graph-version event
- THEN bridge relationships SHALL chỉ được project qua change set của graph version đó
- AND worker SHALL NOT nhận thêm legacy relationship event riêng cho cùng bridge relationship
