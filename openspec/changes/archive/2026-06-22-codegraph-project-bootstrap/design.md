# Design: CodeGraph Project Bootstrap

## Overview

Change này chỉ xử lý local developer experience. SQLite/FTS index của CodeGraph là hot path cho:

- exact symbol lookup
- caller/callee lookup
- blast radius trong cùng repo

## Key Decisions

### 1. Bootstrap phải độc lập với `kg-service`

Không phụ thuộc ontology, search API, hay sync bridge. Mục tiêu là tạo giá trị ngay cả khi các
change sau chưa bắt đầu.

### 2. Repo guidance phải explicit

`CLAUDE.md` hoặc tài liệu tương đương phải nêu thứ tự ưu tiên:

1. `codegraph_explore`
2. `codegraph_search`
3. `codegraph_callers` / `codegraph_callees`
4. đọc file trực tiếp khi local graph chưa đủ

### 3. User guide phải tách khỏi repo-specific details

Guide cần nói rõ phần nào là reusable cho project khác và phần nào là riêng của `kg-service`.
