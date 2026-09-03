---
id: TASK-DEP-001
title: Deprecate Zep Thread Service
service: zep-thread
status: Done
---

# Objective
Formally deprecate the `zep-thread` standalone service.

# Requirements
1. Verify `zep-core` has successfully absorbed the Thread responsibilities.
2. Remove any CI/CD deployment pipelines for `zep-thread` binary.
3. Archive the codebase and update README to point to `zep-core`.
