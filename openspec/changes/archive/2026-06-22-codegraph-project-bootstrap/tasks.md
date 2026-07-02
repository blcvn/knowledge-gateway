# Tasks

- [x] **T1** — Cài CodeGraph CLI và xác nhận workflow local hoạt động trên repo.
- [x] **T2** — Chạy `codegraph init -i` tại repo root.
- [x] **T3** — Tạo `.codegraph/config.json` với ignore patterns phù hợp cho Go project.
- [x] **T4** — Cập nhật `CLAUDE.md` để ưu tiên `codegraph_*` trước grep/read cho repo-local navigation.
- [x] **T5** — Thêm local automation để refresh incremental index sau commit hoặc khi developer yêu cầu.
- [x] **T6** — Viết user guide cho việc bootstrap CodeGraph trên project Go khác.
- [x] **T7** — Verify `codegraph status`, `codegraph search`, và MCP server hoạt động.

> Note: In CodeGraph v1.0.1, the CLI uses `codegraph query` for symbol search, which is the
> equivalent of the older `search` wording used in the task/spec text.
