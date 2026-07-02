# Proposal: KG Service CodeGraph Platform Updates

## Problem

Core `code-graph` integration có thể chạy trên API surface hiện có, nhưng có thể cần thêm platform
capabilities để tối ưu throughput, cleanup, hoặc raw graph-query ergonomics.

## Proposed Solution

Định nghĩa các update additive, tùy chọn cho `kg-service`:

- bulk node writes
- bulk relationship writes
- delete by external-ref prefix
- raw graph search endpoint

## Scope

### In scope

- optional API additions cho `kg-service`
- semantics alignment với auth, ACL, query strategy, search profile hiện có

### Out of scope

- ontology bootstrap
- bridge implementation
- CodeGraph local bootstrap

## Success Criteria

- extension endpoints, nếu được implement, cho kết quả tương đương core path
- extension path không bypass auth/ACL/search-profile semantics
