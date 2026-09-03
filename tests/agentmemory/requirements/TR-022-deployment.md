# TR-022: Deployment & Configuration Test Requirements

**Module:** Deployment, Configuration, CLI  
**Nguồn:** SRS §9, §4.5, PRD §9, URD §3.1, §3.7, §3.8  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-022-DEP-001 — One-command install và start
🔴 P0 | `[E2E]` | **UR-001**

**Given:** Node.js LTS installed, iii-engine available  
**When:**
```bash
npm install -g @agentmemory/agentmemory
agentmemory
```
**Then:** Server starts và HTTP 200 từ `:3111/agentmemory/health` trong < 30 giây

**Traceability:** UR-001, SRS §4.5

---

## TR-022-DEP-002 — npx: zero-config start
🔴 P0 | `[E2E]` | **UR-001**

**Given:** Node.js LTS installed  
**When:** `npx @agentmemory/agentmemory`  
**Then:** Server starts thành công trong < 30 giây

**Traceability:** UR-001, SRS §4.5

---

## TR-022-DEP-003 — connect claude-code: hook installation
🔴 P0 | `[E2E]` | **UR-002**

**Given:** Claude Code installed, agentmemory running  
**When:** `agentmemory connect claude-code`  
**Then:**
- Hook scripts được install vào Claude Code hooks directory
- 12 hooks được registered

**Traceability:** UR-002, UR-003

---

## TR-022-DEP-004 — connect codex: hook installation
🟠 P1 | `[E2E]`

**Given:** Codex CLI installed  
**When:** `agentmemory connect codex`  
**Then:** 6 hooks được registered cho Codex

**Traceability:** PRD §8 (Codex 6 hooks)

---

## TR-022-DEP-005 — Data directory: ~/.agentmemory/
🔴 P0 | `[E2E]` | **SRS §2.3**

**Given:** Default config  
**When:** Server start và đầu tiên sử dụng  
**Then:** `~/.agentmemory/` directory được tạo với:
- `.env` (config file)
- `bm25-index.json` (sau khi data được indexed)
- `vector-index.json` (nếu embedding provider)

**Traceability:** SRS §2.3, Architecture §9.2

---

## TR-022-DEP-006 — Docker deployment
🟠 P1 | `[E2E]` | **UR-005**

**Given:** Docker Compose config  
**When:** `docker-compose up`  
**Then:**
- Container start thành công
- REST API accessible tại mapped port
- Data persisted đến volume

**Traceability:** UR-005, SRS §9.2

---

## TR-022-DEP-007 — Environment variables: complete reference
🔴 P0 | `[UNIT]` | **UR-027**

**Given:** Mỗi environment variable từ SRS §9.3  
**When:** Biến được set và server start  
**Then:** Config được loaded đúng:

| Variable | Default | Expected |
|---|---|---|
| `III_REST_PORT` | 3111 | REST port |
| `TOKEN_BUDGET` | 2000 | Context budget |
| `MAX_OBS_PER_SESSION` | 500 | Session limit |
| `BM25_WEIGHT` | 0.4 | Search weight |
| `VECTOR_WEIGHT` | 0.6 | Search weight |
| `AGENTMEMORY_GRAPH_WEIGHT` | 0.3 | Graph weight |
| `AUTO_FORGET_INTERVAL_MS` | 3600000 | 1 hour |
| `CONSOLIDATION_INTERVAL_MS` | 7200000 | 2 hours |

**Traceability:** SRS §9.3, UR-027

---

## TR-022-DEP-008 — Multi-instance port conflict
🟡 P2 | `[E2E]`

**Given:** 2 instances cố start trên cùng port  
**When:** Second instance start  
**Then:** Second instance fail với clear error message về port conflict

**Traceability:** SRS §9.3

---

## TR-022-DEP-009 — upgrade command
🟡 P2 | `[E2E]` | **UR-033**

**Given:** agentmemory installed  
**When:** `agentmemory upgrade`  
**Then:**
- New version được installed (nếu available)
- Existing data preserved
- No data loss

**Traceability:** UR-033

---

## TR-022-DEP-010 — iii-engine version pin: 0.11.2
🔴 P0 | `[UNIT]`

**Given:** package.json  
**When:** iii dependency được check  
**Then:** iii-engine version = 0.11.2 (pinned, không higher)

**Traceability:** SRS §10.1

---

## TR-022-DEP-011 — Windows: WSL2 required message
🟡 P2 | `[UNIT]`

**Given:** Windows platform detected  
**When:** `agentmemory connect` được chạy  
**Then:** Clear message về WSL2 requirement

**Traceability:** SRS §10.1

---

## TR-022-DEP-012 — fly.io deployment template
🟡 P2 | `[E2E]` | **UR-005**

**Given:** `deploy/fly/` templates  
**When:** Deployed đến fly.io  
**Then:**
- App healthy
- HMAC secret auto-generated
- Persistent volume mounted
- iii binary included

**Traceability:** UR-005, PRD §9
