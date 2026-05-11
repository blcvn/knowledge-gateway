---
id: TASK-DEP-001
title: Deprecate Zep User Service
service: zep-user
status: Done
---

# Objective
Formally deprecate the `zep-user` standalone service.

# Requirements
1. Verify `zep-core` has successfully absorbed the User responsibilities.
2. Remove any CI/CD deployment pipelines for `zep-user` binary.
3. Archive the codebase and update README to point to `zep-core`.
