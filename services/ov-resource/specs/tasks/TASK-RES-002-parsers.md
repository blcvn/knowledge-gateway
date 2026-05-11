---
id: TASK-RES-002
service: ov-resource
status: Done
---

# TASK-RES-002: Implement Parser Registry and Strategies

## Objective
Develop the file parsing infrastructure capable of extracting structured chunks from various file formats as defined in `architecture.md` and `api.md`.

## Requirements
1. **Parser Registry (`internal/adapter/parser/registry.go`)**:
   - Implement factory/registry mapping extensions to the appropriate strategy.
   - Inject configurations: `CHUNK_SIZE_TOKENS` (default 512), `CHUNK_OVERLAP_TOKENS` (default 50), and `TREESITTER_ENABLED`.
2. **Parser Implementations (`internal/adapter/parser/`)**:
   - **Tree-sitter Parser (`treesitter.go`)**: Parse `.go`, `.py`, `.js`, `.ts`, `.rs`, `.java`. AST-aware extraction (functions, classes, methods). Target ~500 tokens/chunk. Requires CGo. Skip if `TREESITTER_ENABLED=false`.
   - **Markdown Parser (`markdown.go`)**: Parse `.md`, `.mdx` using section-based heading levels (H1-H6). Target ~800 tokens.
   - **Document Parser (`document.go`)**: Parse `.pdf`, `.docx` into page-based chunks with configured overlap. Target ~1000 tokens.
   - **Default Parser**: Fallback for `.txt`, `.csv`, `.log`. Paragraph-based (double newline). Target ~500 tokens.
3. **Token Estimation**:
   - Calculate `total_tokens` for generated chunks to populate the `IngestResponse` and DB records.

## Dependencies
- `tree-sitter` Go bindings (CGo required).
- PDF/DOCX parsing Go libraries.
