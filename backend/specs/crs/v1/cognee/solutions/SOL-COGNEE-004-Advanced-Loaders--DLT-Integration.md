# SOL-COGNEE-004 — Solution: Advanced Loaders & DLT Integration

| Field | Value |
|---|---|
| **Solution ID** | SOL-COGNEE-004 |
| **CR** | [CR-COGNEE-004](../../../../docs/crs/v1/cognee/CR-COGNEE-004*.md) |
| **TDD ref** | [02-cognee-services.md](../../../tdd/architecture/02-cognee-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |

---
## 1. Giải pháp

Thêm loaders cho GitHub, Confluence, Notion via DLT (data load tool) connectors.

### 1.1 `services/cognee-ingestion/internal/adapter/loader/` [NEW directory]

```go
// github_loader.go
type GitHubLoader struct { token, org, repo string }
func (l *GitHubLoader) Load(ctx context.Context) ([]DataItem, error) {
    // Fetch issues, PRs, wikis via GitHub API
    // Return as DataItem{Source: "github", Content: ...}
}

// confluence_loader.go  
type ConfluenceLoader struct { baseURL, token, spaceKey string }

// notion_loader.go
type NotionLoader struct { token, pageID string }
```

### 1.2 Loader registry

```go
var loaders = map[string]Loader{
    "github":     &GitHubLoader{},
    "confluence": &ConfluenceLoader{},
    "notion":     &NotionLoader{},
    "url":        &URLLoader{},
    "file":       &FileLoader{},
}
```

## 2. Acceptance Criteria

- [ ] GitHub loader: issues, PRs, README ingested
- [ ] Confluence loader: pages by space key
- [ ] Notion loader: pages and databases
- [ ] All loaders return standard DataItem format
