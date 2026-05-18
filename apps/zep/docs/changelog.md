# Changelog

All notable changes to the `Zep Monolith` application will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Khởi tạo kiến trúc Zep Monolith với Embedded Supervisor.
- Tích hợp 6 domain services (`zep-user`, `zep-thread`, `zep-memory`, `zep-graph`, `zep-search`, `zep-admin`).
- Tích hợp `gateway` expose public REST/gRPC.
- Cấu hình Unified Configuration để chống conflict port.
- Thêm Dockerfile, Makefile, docker-compose.yml phục vụ build và triển khai.
- Hoàn thiện hệ thống specs (TASK-001 tới TASK-005) và docs (README, API, Architecture, Runbook, Configuration).
