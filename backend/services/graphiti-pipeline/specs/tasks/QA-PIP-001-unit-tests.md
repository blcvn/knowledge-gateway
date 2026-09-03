---
id: QA-PIP-001
title: Comprehensive Unit Tests
feature: SOL-001
status: Done
---

## Objective
Đảm bảo chất lượng mã nguồn thông qua unit tests bao phủ >= 80% dựa trên kế hoạch chất lượng của SOL-001.

## Tasks
1. Unit tests Domain Layer
   - Validation methods, type constructors.
   - (Target >= 90%).

2. Unit tests Usecase Layer
   - Saga state machine, dedup logic, etc.
   - Sử dụng mocked adapters.
   - (Target >= 80%).

3. Unit tests Adapter Layer
   - gRPC handlers, LLM client, repositories, NATS publisher.
   - (Target >= 80%).

4. Unit tests Infra Layer
   - Config validation, server interceptors.
   - (Target >= 70%).

5. Cấu hình CI/CD
   - Cập nhật Makefile target `make test`.
   - Setup CI check để reject nếu coverage giảm dưới mức yêu cầu.
