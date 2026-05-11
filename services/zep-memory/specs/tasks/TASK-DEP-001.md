---
id: TASK-DEP-001
title: Deprecate Zep Memory Service
service: zep-memory
status: Done
---

# Objective
Formally deprecate the `zep-memory` standalone service.

# Requirements
1. Verify `zep-core` has successfully absorbed the Memory responsibilities (PutMemory, GetMemory).
2. Remove any CI/CD deployment pipelines for `zep-memory` binary.
3. Archive the codebase and update README to point to `zep-core`.
