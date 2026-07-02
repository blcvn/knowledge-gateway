# CodeGraph Integration — ba-agent-service
**Design Document v1.0**
*Phạm vi: Phase 1 (SQLite baseline) + Phase 2 (KG Hybrid)*

---

## 1. Mục tiêu

Tích hợp CodeGraph vào `ba-agent-service` để:

- **Phase 1** — Giảm token và tool call khi Claude Code navigate codebase Go của service (structural query: tìm function, trace call graph, blast radius trước refactor).
- **Phase 2** — Mở rộng sang KG hybrid: sync graph lên KG service sẵn có, thêm `codegraph-sync` cùng các MCP tool mới (`kg_semantic_search`, `kg_fulltext_search`, `kg_code_template_query`) phục vụ semantic search, full-text search, và template-backed traversal mà SQLite FTS5 không đáp ứng được.

Nguyên tắc thiết kế xuyên suốt:

> SQLite là **hot path** — không thay thế, không bypass.
> KG service là **persistent layer** — sync bất đồng bộ, không block agent query.
> Hai tầng phục vụ hai loại câu hỏi khác nhau, không cạnh tranh.

---

## 2. Kiến trúc tổng quan

```
┌─────────────────────────────────────────────────────────┐
│                    Developer machine                     │
│                                                         │
│  ┌──────────────┐    MCP (stdio)   ┌─────────────────┐  │
│  │  Claude Code │◄────────────────►│  CodeGraph MCP  │  │
│  │  / Cursor    │                  │  server         │  │
│  └──────────────┘                  └────────┬────────┘  │
│                                             │           │
│                                    ┌────────▼────────┐  │
│                                    │  SQLite + FTS5  │  │
│                                    │  .codegraph/db  │  │
│                                    └────────┬────────┘  │
│                                             │ post-index│
│                                    ┌────────▼────────┐  │
│                                    │  KG Sync Bridge │  │
│                                    │  (Node.js)      │  │
│                                    └────────┬────────┘  │
└─────────────────────────────────────────────┼───────────┘
                                              │ HTTP/Bolt/gRPC
                              ┌───────────────▼────────────────┐
                              │         KG Service             │
                              │    (tài liệu đính kèm riêng)   │
                              └────────────────────────────────┘
```

### Query path phân loại

| Loại câu hỏi | Tool | Backend | Latency |
|---|---|---|---|
| Structural: "GenerateFeatureAction gọi gì?" | `codegraph_explore` | SQLite local | <5ms |
| Structural: blast radius, callers/callees | `codegraph_impact` | SQLite local | <5ms |
| Template-backed traversal: callers/callees/impact/implements | `kg_code_template_query` | KG service | 10–50ms |
| Semantic: "những handler nào xử lý tax routing?" | `kg_semantic_search` | KG service | 20–80ms |
| Full-text: "sync bridge" trong docstring/tên symbol | `kg_fulltext_search` | KG service | 10–40ms |

---

## 3. Phase 1 — SQLite baseline

### 3.1 Cài đặt

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh | sh

# Mở terminal mới (installer thêm codegraph vào PATH)
codegraph install   # auto-detect Claude Code, Cursor, Codex CLI
```

### 3.2 Init từng project

```bash
# ba-agent-service
cd /path/to/ba-agent-service
codegraph init -i   # tạo .codegraph/ và index ngay

# Nếu có Aevlex repo
cd /path/to/aevlex
codegraph init -i
```

`codegraph init -i` làm hai việc: tạo `.codegraph/codegraph.db` (SQLite) và chạy full index bằng tree-sitter. Với codebase Go ~10k–50k lines, lần đầu mất khoảng 30–120 giây. Các lần sau là incremental (<2s).

### 3.3 Cấu hình .codegraph/config.json

```json
{
  "ignore": [
    "vendor/**",
    "**/*_test.go",
    ".codegraph/**",
    "dist/**",
    "*.pb.go"
  ],
  "languages": ["go"],
  "autoSync": true
}
```

Bỏ `*_test.go` và `*.pb.go` khỏi index để tránh noise — agent không cần navigate file generated hay test khi đang làm BA task.

### 3.4 CLAUDE.md — hướng dẫn agent dùng tool

Tạo file `CLAUDE.md` ở root của mỗi project (CodeGraph installer đã wire vào Claude Code instructions, nhưng cần override với context cụ thể):

```markdown
# ba-agent-service — Agent instructions

## CodeGraph tools (ưu tiên cao nhất)

Với mọi câu hỏi về cấu trúc code, **luôn dùng CodeGraph trước**,
không grep, không Read file trực tiếp:

- `codegraph_explore` — tìm symbol, xem source + relationship map
- `codegraph_search` — tìm symbol theo tên
- `codegraph_callers` / `codegraph_callees` — trace call graph
- `codegraph_impact` — blast radius của một symbol
- `codegraph_files` — cấu trúc file (thay ls/find)

Thứ tự ưu tiên khi cần hiểu một function:
1. `codegraph_explore("<tên function>")` — một call, đủ dùng
2. Nếu cần thêm context: `codegraph_callers` / `codegraph_callees`
3. Chỉ đọc file trực tiếp khi cần xem toàn bộ implementation

## Cấu trúc pipeline ba-agent-service

- Step 1: `internal/step1/` — Feature Action generation
- Step 2/3A: `internal/step2/` — Main Flow + AF trigger map
- Step 3B: `internal/step3b/` — Alternative Flow steps
- Step 3C: `internal/step3c/` — Exception patches
- Prompt files: `prompts/` — injectable architecture patterns
- Models: `internal/models/` — schema types (FeatureAction, Flow, etc.)
```

### 3.5 Post-commit hook (giữ index fresh)

```bash
# .git/hooks/post-commit
#!/bin/sh
cd "$(git rev-parse --show-toplevel)"
codegraph index --incremental 2>/dev/null &
echo "✓ CodeGraph incremental sync started"
```

`--incremental` chỉ re-index file thay đổi trong commit. Chạy background (`&`) không block commit flow.

### 3.6 Kiểm tra

```bash
codegraph status          # xem số node/edge đã index
codegraph search "GenerateFeatureAction"   # test tìm symbol
codegraph serve --mcp     # test MCP server chạy được
```

---

## 4. Phase 2 — KG Hybrid

### 4.1 Tổng quan thay đổi so với Phase 1

Phase 2 thêm một **sync bridge** chạy sau mỗi lần index và ba **MCP tool mới** đăng ký song song với CodeGraph. Không có gì thay đổi trong Phase 1 — `codegraph_explore` và SQLite vẫn là hot path.

```
Phase 1:  agent → codegraph_explore → SQLite
                                         ↑
Phase 2:  thêm:  SQLite ──async──► codegraph-sync bridge ──► KG service
                 agent → kg_semantic_search      ─────────► KG service
                 agent → kg_fulltext_search       ─────────► KG service
                 agent → kg_code_template_query   ─────────► KG service
```

### 4.2 Cấu trúc thư mục codegraph-sync

```
codegraph-sync/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts          # entry point, CLI: node src/index.ts sync
│   ├── extractor.ts      # đọc graph từ CodeGraph API
│   ├── serializer.ts     # chuyển NodeRecord → KG-specific format
│   ├── adapter.ts        # interface KGAdapter (abstract)
│   ├── adapters/
│   │   └── your-kg.ts    # implement cho KG service cụ thể (điền sau)
│   ├── mcp/
│   │   ├── server.ts     # MCP server expose kg_semantic_search + kg_fulltext_search + kg_code_template_query
│   │   └── tools.ts      # tool definitions và handlers
│   └── watcher.ts        # file watcher → trigger sync
├── .env.example
└── README.md
```

### 4.3 KGAdapter interface

Interface này là điểm duy nhất cần implement theo tài liệu KG service cụ thể. Phần còn lại của bridge không thay đổi.

```typescript
// src/adapter.ts

export interface SymbolNode {
  id: string;           // CodeGraph node ID (stable across syncs)
  name: string;         // function/class/method name
  kind: NodeKind;       // function | method | class | struct | interface | ...
  file: string;         // relative path từ project root
  line: number;
  language: string;     // "go" | "typescript" | ...
  projectId: string;    // định danh project (e.g. "ba-agent-service")
  commitSha?: string;   // git SHA tại thời điểm sync
}

export type EdgeKind =
  | 'calls'
  | 'imports'
  | 'implements'
  | 'extends'
  | 'references'
  | 'instantiates'
  | 'contains';

export interface SymbolEdge {
  fromId: string;
  toId: string;
  kind: EdgeKind;
  provenance: 'ast' | 'heuristic';  // CodeGraph đánh dấu edge tổng hợp
  projectId: string;
}

export interface KGQueryResult {
  nodes: SymbolNode[];
  edges: SymbolEdge[];
  explanation?: string;   // optional: KG service tự generate
}

export interface KGAdapter {
  /**
   * Upsert batch nodes + edges. Idempotent — gọi nhiều lần với cùng data = safe.
   * @param nodes  - danh sách symbol nodes
   * @param edges  - danh sách edges giữa nodes
   */
  upsertBatch(nodes: SymbolNode[], edges: SymbolEdge[]): Promise<void>;

  /**
   * Xoá toàn bộ data của một project (dùng khi full re-index).
   */
  clearProject(projectId: string): Promise<void>;

  /**
   * Template-backed traversal query — dùng cho callers/callees/impact/implements.
   * Input: template name và params bind vào template phía KG service.
   * Output: records trả về từ template.
   */
  templateQuery(templateName: string, params: Record<string, unknown>): Promise<{
    results: Array<Record<string, unknown>>;
    queryTimeMs: number;
  }>;

  /**
   * Semantic search — tìm symbol theo nghĩa, không phải tên.
   * Input: mô tả tự nhiên ("handler xử lý authentication").
   * Output: top-K nodes sắp xếp theo relevance score.
   */
  semanticSearch(query: string, opts?: {
    projectIds?: string[];
    topK?: number;           // mặc định 10
    kindFilter?: NodeKind[];
  }): Promise<Array<SymbolNode & { score: number }>>;

  /**
   * Health check — trả về true nếu KG service reachable.
   */
  ping(): Promise<boolean>;

  close(): Promise<void>;
}
```

### 4.4 Extractor — đọc graph từ CodeGraph API

```typescript
// src/extractor.ts
import CodeGraph from '@colbymchenry/codegraph';
import type { SymbolNode, SymbolEdge } from './adapter.js';

export interface ExtractResult {
  nodes: SymbolNode[];
  edges: SymbolEdge[];
  stats: { nodeCount: number; edgeCount: number; durationMs: number };
}

export async function extractGraph(
  projectPath: string,
  projectId: string,
  commitSha?: string
): Promise<ExtractResult> {
  const start = Date.now();
  const cg = await CodeGraph.open(projectPath);

  try {
    // FTS5 empty query = match all symbols
    const allSymbols = cg.searchNodes('');
    const nodes: SymbolNode[] = [];
    const edgeMap = new Map<string, SymbolEdge>();

    for (const { node } of allSymbols) {
      // Bỏ qua file node và import node — chỉ lấy semantic symbols
      if (node.kind === 'file' || node.kind === 'import' || node.kind === 'export') continue;

      nodes.push({
        id:        node.id,
        name:      node.name,
        kind:      node.kind as any,
        file:      node.file,
        line:      node.line ?? 0,
        language:  node.language ?? 'unknown',
        projectId,
        commitSha,
      });

      // Collect CALLS edges (callees của node này)
      const callees = cg.getCallees(node.id);
      for (const callee of callees) {
        const edgeId = `${node.id}::calls::${callee.id}`;
        if (!edgeMap.has(edgeId)) {
          edgeMap.set(edgeId, {
            fromId:     node.id,
            toId:       callee.id,
            kind:       'calls',
            provenance: (callee as any).provenance === 'heuristic' ? 'heuristic' : 'ast',
            projectId,
          });
        }
      }
    }

    // Collect IMPACT edges thêm (references, implements...)
    // CodeGraph getImpactRadius trả về transitive — chỉ lấy depth=1 để avoid duplicates
    for (const { node } of allSymbols) {
      const impact = cg.getImpactRadius(node.id, 1);
      for (const impacted of impact) {
        if (impacted.id === node.id) continue;
        const edgeId = `${node.id}::references::${impacted.id}`;
        if (!edgeMap.has(edgeId)) {
          edgeMap.set(edgeId, {
            fromId:     node.id,
            toId:       impacted.id,
            kind:       'references',
            provenance: 'ast',
            projectId,
          });
        }
      }
    }

    const edges = Array.from(edgeMap.values());
    return {
      nodes,
      edges,
      stats: { nodeCount: nodes.length, edgeCount: edges.length, durationMs: Date.now() - start },
    };
  } finally {
    cg.close();
  }
}
```

### 4.5 Main sync entry point

```typescript
// src/index.ts
import { extractGraph } from './extractor.js';
import { createAdapter } from './adapters/your-kg.js';
import { execSync } from 'child_process';
import * as path from 'path';

interface SyncOptions {
  projectPath: string;
  projectId: string;
  fullReindex?: boolean;    // xoá data cũ trước khi upsert
  dryRun?: boolean;         // chỉ extract, không ghi vào KG
}

export async function syncProjectToKG(opts: SyncOptions) {
  const { projectPath, projectId, fullReindex = false, dryRun = false } = opts;

  // Lấy git SHA nếu có
  let commitSha: string | undefined;
  try {
    commitSha = execSync('git rev-parse HEAD', { cwd: projectPath })
      .toString().trim().slice(0, 8);
  } catch { /* không phải git repo */ }

  console.log(`[sync] Extracting graph: ${projectId} @ ${commitSha ?? 'unknown'}`);
  const { nodes, edges, stats } = await extractGraph(projectPath, projectId, commitSha);
  console.log(`[sync] Extracted: ${stats.nodeCount} nodes, ${stats.edgeCount} edges (${stats.durationMs}ms)`);

  if (dryRun) {
    console.log('[sync] Dry run — không ghi vào KG');
    return { nodes, edges, stats };
  }

  const adapter = await createAdapter();
  try {
    if (!(await adapter.ping())) {
      throw new Error('KG service unreachable');
    }
    if (fullReindex) {
      console.log(`[sync] Full reindex — clearing project ${projectId}...`);
      await adapter.clearProject(projectId);
    }
    await adapter.upsertBatch(nodes, edges);
    console.log(`[sync] Done: ${projectId} synced to KG`);
  } finally {
    await adapter.close();
  }

  return { nodes, edges, stats };
}

// CLI entrypoint
if (process.argv[2] === 'sync') {
  const projectPath = process.env.PROJECT_PATH ?? process.cwd();
  const projectId   = process.env.PROJECT_ID   ?? path.basename(projectPath);
  const fullReindex = process.argv.includes('--full');
  const dryRun      = process.argv.includes('--dry-run');

  syncProjectToKG({ projectPath, projectId, fullReindex, dryRun })
    .then(({ stats }) => {
      console.log(`[sync] Complete: ${stats.nodeCount} nodes, ${stats.edgeCount} edges`);
      process.exit(0);
    })
    .catch(err => {
      console.error('[sync] Failed:', err.message);
      process.exit(1);
    });
}
```

### 4.6 Adapter placeholder — điền theo tài liệu KG service

```typescript
// src/adapters/your-kg.ts
// TODO: implement theo tài liệu KG service đính kèm
//
// Cần implement toàn bộ KGAdapter interface từ src/adapter.ts:
//   - upsertBatch(nodes, edges)
//   - clearProject(projectId)
//   - templateQuery(templateName, params)
//   - semanticSearch(query, opts)
//   - ping()
//   - close()
//
// Xem src/adapter.ts để biết input/output types.
// Xem src/mcp/tools.ts để biết context mà tool handlers truyền vào adapter.

import type { KGAdapter } from '../adapter.js';

export async function createAdapter(): Promise<KGAdapter> {
  throw new Error('KG adapter chưa được implement — xem tài liệu KG service');
}
```

### 4.7 MCP server cho KG tools

```typescript
// src/mcp/tools.ts
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { z } from 'zod';
import { createAdapter } from '../adapters/your-kg.js';

const PROJECT_IDS = (process.env.KG_PROJECT_IDS ?? 'ba-agent-service')
  .split(',').map(s => s.trim());

export async function startKGMCPServer() {
  const adapter = await createAdapter();
  const server  = new McpServer({ name: 'kg-tools', version: '1.0.0' });

  // ── Tool 1: kg_code_template_query ────────────────────────────────
  server.tool(
    'kg_code_template_query',
    `Execute a persistent code-graph traversal template.
Use when the question is callers, callees, impact, or implements lookup
that should be resolved through the KG service template path.

Use for: "what calls TaxRouter?",
         "blast radius if I change this interface",
         "which handlers implement X pattern".
Do NOT use for: one-off semantic lookup (use kg_semantic_search instead).`,
    {
      templateName: z.string().describe(
        'Tên template persistent traversal. Ví dụ: "code_callers", ' +
        '"code_callees", "code_impact", "code_implements".'
      ),
      params: z.record(z.any()).default({}).describe(
        'Tham số template sẽ bind vào query template của KG service.'
      ),
    },
    async ({ templateName, params }) => {
      try {
        const result = await adapter.templateQuery(templateName, params);

        return {
          content: [{ type: 'text' as const, text: summarizeTemplate(result) }],
        };
      } catch (err: any) {
        return {
          content: [{ type: 'text' as const, text: `KG template query failed: ${err.message}` }],
          isError: true,
        };
      }
    }
  );

  // ── Tool 2: kg_semantic_search ────────────────────────────────────
  server.tool(
    'kg_semantic_search',
    `Search the knowledge graph by semantic meaning, not just symbol name.
Use when you don't know the exact function name but know what it does.
Returns ranked symbols with file locations and relevance scores.

Use for: "function nào handle authentication?",
         "tìm code liên quan đến tax routing",
         "handler nào sinh swimlane diagram?".
Do NOT use for: exact name lookup (use codegraph_search instead).`,
    {
      query: z.string().describe(
        'Mô tả semantic về chức năng cần tìm. Tiếng Việt OK. ' +
        'Ví dụ: "generate swimlane diagram", "xử lý exception trong flow".'
      ),
      projectIds: z.array(z.string()).optional(),
      topK: z.number().int().min(1).max(50).optional().default(10),
      kindFilter: z.array(z.enum([
        'function', 'method', 'class', 'struct', 'interface',
      ])).optional().describe('Lọc theo loại symbol.'),
    },
    async ({ query, projectIds, topK, kindFilter }) => {
      try {
        const results = await adapter.semanticSearch(query, {
          projectIds: projectIds ?? PROJECT_IDS,
          topK,
          kindFilter: kindFilter as any,
        });

        const lines = results.map((r, i) =>
          `${i + 1}. [${r.kind}] ${r.name}\n   File: ${r.file}:${r.line}\n   Project: ${r.projectId}\n   Score: ${r.score.toFixed(3)}`
        );

        return {
          content: [{
            type: 'text' as const,
            text: results.length > 0
              ? `Found ${results.length} symbols:\n\n${lines.join('\n\n')}`
              : 'No matching symbols found.',
          }],
        };
      } catch (err: any) {
        return {
          content: [{ type: 'text' as const, text: `Semantic search failed: ${err.message}` }],
          isError: true,
        };
      }
    }
  );

  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error('[kg-mcp] KG MCP server running on stdio');
}

function formatGraphResult(result: import('../adapter.js').KGQueryResult): string {
  if (result.nodes.length === 0) return 'No results found in knowledge graph.';

  const nodesByProject = Map.groupBy(result.nodes, n => n.projectId);
  const sections: string[] = [];

  for (const [projectId, nodes] of nodesByProject) {
    const nodeLines = nodes.map(n =>
      `  - [${n.kind}] ${n.name} (${n.file}:${n.line})`
    ).join('\n');
    sections.push(`**${projectId}** (${nodes.length} nodes):\n${nodeLines}`);
  }

  const edgeSummary = result.edges.length > 0
    ? `\n\nRelationships: ${result.edges.length} edges`
    : '';

  const explanation = result.explanation
    ? `\n\n${result.explanation}`
    : '';

  return sections.join('\n\n') + edgeSummary + explanation;
}

startKGMCPServer().catch(err => {
  console.error('[kg-mcp] Fatal:', err);
  process.exit(1);
});
```

### 4.8 Đăng ký KG MCP server vào Claude Code

Thêm vào `.claude.json` (hoặc `~/.claude.json` để global):

```json
{
  "mcpServers": {
    "codegraph": {
      "command": "codegraph",
      "args": ["serve", "--mcp"],
      "cwd": "/path/to/ba-agent-service"
    },
    "kg-tools": {
      "command": "node",
      "args": ["/path/to/codegraph-sync/src/mcp/tools.js"],
      "env": {
        "KG_PROJECT_IDS": "ba-agent-service,aevlex",
        "KG_SERVICE_URL": "http://localhost:7474"
      }
    }
  }
}
```

Hai MCP server chạy song song, độc lập nhau. Claude Code có thể gọi tool từ cả hai.

### 4.9 CLAUDE.md cập nhật cho Phase 2

Thêm section sau vào `CLAUDE.md`:

```markdown
## KG tools — khi nào dùng (Phase 2)

Ba tool bổ sung, chỉ dùng khi `codegraph_explore` không đủ:

### kg_semantic_search
Dùng khi không biết tên chính xác:
- "Tìm function xử lý alternative flow exception"
- "Code nào liên quan đến swimlane rendering?"
- "Handler nào generate prompt cho Step 3B?"

KHÔNG dùng cho: tên symbol đã biết (dùng codegraph_search).

### kg_fulltext_search
Dùng khi biết từ khóa hoặc cụm từ, nhưng không cần semantic ranking:
- "sync bridge"
- "template query"
- "docstring traversal"

KHÔNG dùng cho: câu hỏi ý nghĩa rộng (dùng `kg_semantic_search`).

### kg_code_template_query
Dùng khi muốn gọi persistent traversal có tên template rõ ràng:
- "callers của một symbol"
- "callees của một symbol"
- "impact/implements lookup"

KHÔNG dùng cho: truy vấn một-off không có template (dùng `kg_semantic_search` hoặc đọc file trực tiếp).

### Thứ tự ưu tiên tổng hợp

1. `codegraph_explore` — structural, single project, tên đã biết
2. `codegraph_impact` — blast radius trong project
3. `kg_code_template_query` — persistent callers/callees/impact/implements
4. `kg_semantic_search` — semantic, tên chưa biết
5. `kg_fulltext_search` — keyword/phrase lookup
6. Đọc file trực tiếp — chỉ khi cần toàn bộ implementation
```

### 4.10 Watcher — tự động sync khi code thay đổi

```typescript
// src/watcher.ts
import { syncProjectToKG } from './index.js';
import * as path from 'path';

const DEBOUNCE_MS = 30_000;  // sync tối đa 1 lần/30s

interface WatchConfig {
  projectPath: string;
  projectId: string;
}

export function startWatcher(configs: WatchConfig[]) {
  const timers = new Map<string, NodeJS.Timeout>();

  for (const config of configs) {
    scheduleSync(config, timers, 0);  // sync ngay lần đầu
  }
}

function scheduleSync(
  config: WatchConfig,
  timers: Map<string, NodeJS.Timeout>,
  delay: number
) {
  const existing = timers.get(config.projectId);
  if (existing) clearTimeout(existing);

  const timer = setTimeout(async () => {
    try {
      await syncProjectToKG({
        projectPath: config.projectPath,
        projectId:   config.projectId,
        fullReindex: false,
      });
    } catch (err: any) {
      console.error(`[watcher] Sync failed for ${config.projectId}:`, err.message);
    }
  }, delay);

  timers.set(config.projectId, timer);
}
```

### 4.11 CI integration — GitHub Actions

```yaml
# .github/workflows/codegraph-sync.yml
name: CodeGraph → KG sync

on:
  push:
    branches: [main, develop]
  workflow_dispatch:
    inputs:
      full_reindex:
        description: 'Full reindex (xoá data cũ)'
        type: boolean
        default: false

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '22'

      - name: Install CodeGraph
        run: npm i -g @colbymchenry/codegraph

      - name: Index codebase
        run: |
          codegraph init -i
          codegraph status

      - name: Install sync bridge
        run: |
          cd codegraph-sync
          npm ci

      - name: Sync to KG
        run: |
          ARGS="sync"
          if [ "${{ inputs.full_reindex }}" = "true" ]; then ARGS="$ARGS --full"; fi
          node codegraph-sync/src/index.js $ARGS
        env:
          PROJECT_PATH: ${{ github.workspace }}
          PROJECT_ID: ba-agent-service
          KG_SERVICE_URL: ${{ secrets.KG_SERVICE_URL }}
          # Thêm env vars khác theo tài liệu KG service
```

### 4.12 package.json cho codegraph-sync

```json
{
  "name": "codegraph-sync",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "sync": "node src/index.js sync",
    "sync:full": "node src/index.js sync --full",
    "sync:dry": "node src/index.js sync --dry-run",
    "mcp": "node src/mcp/tools.js",
    "build": "tsc",
    "watch": "tsc --watch"
  },
  "dependencies": {
    "@colbymchenry/codegraph": "latest",
    "@modelcontextprotocol/sdk": "^1.0.0",
    "zod": "^3.22.0"
  },
  "devDependencies": {
    "typescript": "^5.4.0",
    "@types/node": "^22.0.0"
  }
}
```

---

## 5. Quyết định thiết kế — rationale

### Tại sao giữ SQLite trong query path

`codegraph_explore` đạt <5ms vì toàn bộ pipeline (FTS5 search → call graph resolve → source read → markdown format) chạy **in-process** với MCP server, không có network. Thay bằng KG service sẽ thêm ít nhất một TCP round-trip (~10–50ms), phá benchmark 58% fewer tool calls vì agent sẽ timeout context sớm hơn và generate thêm calls để bù lại.

### Tại sao không merge hai MCP server thành một

CodeGraph MCP server được maintain bởi upstream — nếu merge, mỗi lần upstream update phải rebase lại. Hai server riêng cho phép upgrade CodeGraph độc lập, và KG tools có thể được deploy riêng (ví dụ: chỉ trên CI, không trên máy developer).

### Tại sao sync bất đồng bộ

KG service là persistent layer cho analytics và cross-project query — không cần fresh hơn 30 giây. Nếu sync đồng bộ (block agent query), một lần KG service slow sẽ làm chậm toàn bộ developer workflow. Bất đồng bộ đảm bảo hot path không bao giờ bị ảnh hưởng bởi KG availability.

### Tại sao KGAdapter là interface, không phải implementation cụ thể

Mỗi KG service có query language và auth khác nhau (Cypher vs Qdrant REST vs gRPC). Interface cô lập phần này vào một file duy nhất (`src/adapters/your-kg.ts`). Phần còn lại của bridge (extractor, serializer, MCP tools) không thay đổi khi switch backend.

---

## 6. Checklist implementation

### Phase 1
- [ ] Cài CodeGraph CLI
- [ ] `codegraph install` — wire vào Claude Code / Cursor
- [ ] `codegraph init -i` trong `ba-agent-service`
- [ ] Tạo `.codegraph/config.json` với ignore patterns
- [ ] Tạo `CLAUDE.md` tại root project
- [ ] Thêm post-commit hook
- [ ] Verify: `codegraph status` và `codegraph search "<symbol>"`

### Phase 2 (sau khi có tài liệu KG service)
- [ ] Tạo thư mục `codegraph-sync/` với cấu trúc trên
- [ ] Implement `src/adapters/your-kg.ts` theo tài liệu KG service
- [ ] Test adapter: `npm run sync:dry` → xem extracted nodes/edges
- [ ] Test sync: `npm run sync` → verify data trong KG service
- [ ] Test MCP: `npm run mcp` → call từ Claude Code
- [ ] Đăng ký `kg-tools` vào `.claude.json`
- [ ] Cập nhật `CLAUDE.md` với KG tool instructions
- [ ] Setup CI workflow
- [ ] Verify: agent dùng đúng tool cho đúng loại câu hỏi

---

## 7. Files cần tạo — tóm tắt

| File | Ghi chú |
|------|---------|
| `.codegraph/config.json` | Ignore patterns cho Go project |
| `CLAUDE.md` | Agent instructions, cập nhật ở Phase 2 |
| `.git/hooks/post-commit` | Incremental sync sau commit |
| `codegraph-sync/package.json` | Dependencies |
| `codegraph-sync/src/adapter.ts` | KGAdapter interface |
| `codegraph-sync/src/extractor.ts` | Đọc graph từ CodeGraph API |
| `codegraph-sync/src/index.ts` | Sync entry point |
| `codegraph-sync/src/adapters/your-kg.ts` | **Implement theo tài liệu KG service** |
| `codegraph-sync/src/mcp/tools.ts` | KG MCP server |
| `.claude.json` | Đăng ký cả hai MCP server |
| `.github/workflows/codegraph-sync.yml` | CI sync |
