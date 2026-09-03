# SOL-SM-005 — Solution: External Connector Service

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-005 |
| **CR** | CR-SM-005 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/sm-connector` |

---

## 1. Giải pháp

Connectors for external data: Notion, Slack, GitHub, Gmail → auto-ingest via webhooks or polling.

```go
type ConnectorService struct {
    notion    NotionConnector
    slack     SlackConnector
    github    GitHubConnector
    scheduler Scheduler
}

func (s *ConnectorService) SyncAll(ctx context.Context, tenantID string) {
    configs, _ := s.configRepo.GetActive(ctx, tenantID)
    for _, cfg := range configs {
        connector := s.getConnector(cfg.Type)
        docs, _ := connector.FetchSince(ctx, cfg.LastSyncAt)
        for _, doc := range docs { s.ingest.IngestDocument(ctx, doc) }
        s.configRepo.UpdateLastSync(ctx, cfg.ID)
    }
}
```

## 2. Acceptance Criteria

- [ ] Notion connector: pages + databases
- [ ] Slack connector: channels, threads
- [ ] GitHub connector: issues, PRs, README
- [ ] Webhook support for real-time updates

