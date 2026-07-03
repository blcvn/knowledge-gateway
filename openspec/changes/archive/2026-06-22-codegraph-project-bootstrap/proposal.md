# Proposal: CodeGraph Project Bootstrap

## Problem

Repo hiện chưa có workflow chuẩn để dùng CodeGraph cho local source-code navigation. Việc này làm
agent và developer vẫn phải grep/read file thủ công nhiều hơn cần thiết, và chưa có tài liệu tái
sử dụng cho project khác.

## Proposed Solution

Bootstrap CodeGraph cho `kg-service` như một deliverable độc lập:

- cài và cấu hình CodeGraph cho repo
- thêm config và local automation cần thiết
- cập nhật hướng dẫn agent ưu tiên CodeGraph trước grep/read
- viết user guide để áp dụng cùng pattern cho project Go khác

## Scope

### In scope

- `.codegraph/config.json`
- local indexing workflow
- `CLAUDE.md` guidance
- post-commit refresh hook hoặc tương đương
- tài liệu user guide tái sử dụng

### Out of scope

- ontology `code-graph`
- sync vào `kg-service`
- MCP tools riêng cho kg-service-backed lookup
- thay đổi API của `kg-service`

## Success Criteria

- `codegraph status` trả về index hợp lệ trên repo
- agent có guidance rõ để dùng `codegraph_*` cho repo-local navigation
- có user guide ngắn gọn để áp dụng cho project khác
