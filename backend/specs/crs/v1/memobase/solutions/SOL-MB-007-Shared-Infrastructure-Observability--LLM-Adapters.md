# SOL-MB-007 — Solution: Shared Infrastructure: Observability & LLM Adapters

| Field | Value |
|---|---|
| **Solution ID** | SOL-MB-007 |
| **CR** | CR-MB-007 |
| **TDD ref** | [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `shared/pkg` |

---

## 1. Giải pháp

Shared telemetry + LLM adapter for all memobase services via shared/pkg/telemetry + shared/pkg/adapters.

```go
// In each memobase service wire:
_ = telemetry.Init() // registers Prometheus metrics
llm := adapters.NewBifrostLLMClient(cfg.BifrostURL)
tokenizer := tokenizer.NewTiktoken("gpt-4o")
```

## 2. File Changes

| File | Action |
|---|---|
| `shared/pkg/telemetry/memobase.go` | NEW — memobase-specific metrics |
| `shared/pkg/adapters/bifrost.go` | VERIFY — Bifrost LLM adapter exists |

## 3. Acceptance Criteria

- [ ] All memobase services emit Prometheus metrics
- [ ] Bifrost LLM adapter used (no direct OpenAI calls)
- [ ] Tokenizer available for token budget calculation

