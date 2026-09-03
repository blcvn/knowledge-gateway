# C1 — System Context Diagram

> **C4 Level 1:** VNP Memory trong hệ sinh thái AI — Ai tương tác với hệ thống và hệ thống nào bên ngoài được kết nối.

---

## Diagram

```mermaid
C4Context
    title System Context — VNP Memory

    Person(p1, "AI Agent Developer", "Builds AI agents with\npersistent memory via REST/SDK")
    Person(p2, "Platform Engineer", "Deploys & operates\nmemory infrastructure")
    Person(p3, "ML/AI Engineer", "Tunes ontology, evaluates\nretrieval quality")
    Person(p4, "Enterprise Architect", "Governs AI memory:\nGDPR, audit, policies")
    Person(p5, "IDE Plugin User", "Uses AI coding assistant\n(Claude Code, Copilot)")
    Person(p7, "AI Power User", "Interacts with AI daily,\nwants personalization")

    System_Boundary(vnp, "VNP Memory System") {
        System(memory, "VNP Memory", "Unified Cognitive Infrastructure Layer\n6 memory engines + AgentMemory Layer\nREST :8080 | MCP :8082 | Health :8083")
    }

    System_Ext(llm, "LLM Providers", "OpenAI, Anthropic,\nGoogle, Ollama\n(via Bifrost router)")
    System_Ext(claude, "Claude Code / IDE", "AI coding assistant\nusing MCP protocol")
    System_Ext(framework, "AI Frameworks", "LangChain, CrewAI,\nAutoGen, Mastra")
    System_Ext(connectors, "External Data Sources", "Google Drive, Gmail,\nNotion, OneDrive, GitHub")
    System_Ext(sso, "Identity Provider", "Google OAuth2\nSSO login")
    System_Ext(monitoring, "Monitoring Stack", "Prometheus, Grafana\nOpenTelemetry Collector")

    Rel(p1, memory, "Store/Recall/Forget memory\nPOST /v1/memory/store", "REST JSON")
    Rel(p2, memory, "Deploy & monitor\nmake infra-up && make dev", "CLI / Admin API")
    Rel(p3, memory, "Tune ontology, evaluate\nPOST /v1/zep/graph/ontology", "REST JSON")
    Rel(p4, memory, "Audit, GDPR forget\nGET /v1/console/governance/audit", "REST JSON")
    Rel(p5, memory, "memory_store, ov_grep\nPersistent project context", "MCP SSE/HTTP")
    Rel(p7, memory, "View/edit profile\nGET /v1/console/profiles", "REST JSON / Console UI")

    Rel(memory, llm, "LLM calls for:\nextract, classify, summarize", "HTTPS / Bifrost")
    Rel(claude, memory, "22 MCP tools\nmemory_store, ov_grep...", "MCP JSON-RPC 2.0")
    Rel(framework, memory, "REST API integration\nLangChain, AutoGen...", "REST JSON")
    Rel(memory, connectors, "Sync external data\nGDrive, Notion, GitHub", "HTTPS OAuth2")
    Rel(sso, memory, "Google OAuth2 token\n→ VNP JWT", "HTTPS OIDC")
    Rel(memory, monitoring, "Metrics, traces, logs\nprometheus scrape :8083", "HTTP / OTLP")
```

---

## Actors (8 personas)

| Actor | Vai trò | Điểm vào chính |
|---|---|---|
| **P1** AI Agent Developer | Xây dựng AI agents với persistent memory | REST API :8080 hoặc SDK |
| **P2** Platform Engineer | Deploy, vận hành, monitor | Admin API, `make` commands |
| **P3** ML/AI Engineer | Tối ưu retrieval, ontology | REST API + Console |
| **P4** Enterprise Architect | Governance, GDPR, audit | Console Governance |
| **P5** IDE Plugin User | AI coding assistant context | MCP Server :8082 |
| **P6** Framework Integrator | LangChain, AutoGen integration | REST API / MCP |
| **P7** AI Power User | Personalization, profile control | Console UI |
| **P8** Product Manager | Analytics từ conversations | Console Dashboard |

## External Systems

| System | Mục đích | Protocol |
|---|---|---|
| **LLM Providers** | Extraction, classification, summarization | HTTPS → Bifrost multi-provider router |
| **Claude Code / IDE** | MCP client — AI assistant integration | MCP JSON-RPC 2.0 over SSE/HTTP |
| **AI Frameworks** | LangChain, CrewAI, AutoGen, Mastra | REST JSON |
| **Google Drive/Gmail/Notion** | External data connectors (Supermemory) | HTTPS OAuth2 |
| **Google OAuth2** | SSO authentication | OIDC |
| **Prometheus/Grafana** | Metrics scraping | HTTP pull |
| **OpenTelemetry Collector** | Distributed tracing | OTLP gRPC/HTTP |

---

*[← README](./README.md) | [→ C2 Container](./C2-container.md)*
