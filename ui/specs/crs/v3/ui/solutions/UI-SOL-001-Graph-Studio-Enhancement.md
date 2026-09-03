# UI Solution: UI-SOL-001 — Graph Studio Enhancement

**Solution ID:** UI-SOL-001  
**CR References:** [CR-UI-001](../../../../docs/crs/v3/ui/CR-UI-001-Graph-Studio.md)  
**Feature:** Graph Studio — Interactive Knowledge Graph Visualization  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/graph-studio/`

---

## 1. Mục Đích

Xây dựng đầy đủ Graph Studio UI:
- Force-directed interactive graph visualization (Cytoscape.js)
- Entity detail panel khi click node
- Timeline animation — graph changes over time
- Ontology editor — entity/relationship types
- Cypher query console (read-only)

---

## 2. Backend API Contract

```http
POST /v1/console/graph/subgraph    → SubgraphData (nodes + edges)
GET  /v1/console/graph/entity/{id} → entity detail + neighbors
POST /v1/console/graph/timeline    → temporal subgraph
GET  /v1/console/graph/ontology    → OntologySchema
PUT  /v1/console/graph/ontology    → OntologySchema (update)
POST /v1/console/graph/query       → SubgraphData (Cypher/NL query)
```

### TypeScript Types

```typescript
interface SubgraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

interface GraphNode {
  id:          string;
  label:       string;
  type:        string;       // Person, Org, Concept, Event, etc.
  properties?: Record<string, unknown>;
}

interface GraphEdge {
  id:          string;
  source:      string;
  target:      string;
  type:        string;       // relationship label
  properties?: Record<string, unknown>;  // valid_from, valid_to, etc.
}

interface OntologySchema {
  classes:       string[];
  relationships: string[];
  properties:    Record<string, string[]>;
}
```

---

## 3. Components Architecture

### 3.1 Graph Studio Layout

```
GraphStudioPage
├── Toolbar (top)
│   ├── SearchInput             ← seed entity search
│   ├── DepthSelector           ← 1-5 hops
│   ├── EntityTypeFilter        ← multi-select entity types
│   ├── TimeRangePicker         ← for timeline view
│   ├── LayoutSelector          ← force | hierarchical | circular
│   └── ViewTabs                ← Graph | Timeline | Ontology | Query
├── MainCanvas (Cytoscape.js)   ← interactive force-directed graph
│   ├── Nodes                   ← colored by entity type
│   ├── Edges                   ← labeled with relationship type
│   └── MiniMap                 ← bottom-right corner navigation
├── EntityDetailPanel (right, slide-in on node click)
│   ├── EntityHeader            ← label + type badge
│   ├── PropertiesTable         ← key-value pairs
│   ├── TemporalEdges           ← valid_from/valid_to per edge
│   ├── SourceEpisodes          ← provenance links
│   └── NeighborsCount          ← "Connected to 12 nodes"
└── StatusBar (bottom)          ← node count, edge count, truncation warning
```

### 3.2 Node Color Scheme

```typescript
const NODE_COLORS: Record<string, string> = {
  Person:    '#3B82F6',   // blue
  Org:       '#10B981',   // green
  Concept:   '#8B5CF6',   // purple
  Event:     '#F97316',   // orange
  Location:  '#EF4444',   // red
  Product:   '#F59E0B',   // amber
  Default:   '#6B7280',   // gray
};
```

### 3.3 Cytoscape.js Integration

```typescript
// ui/src/pages/graph-studio/GraphCanvas.tsx

import cytoscape from 'cytoscape';
import fcose from 'cytoscape-fcose';  // force-directed layout

cytoscape.use(fcose);

export function GraphCanvas({ data }: { data: SubgraphData }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);
  
  useEffect(() => {
    if (!containerRef.current) return;
    
    cyRef.current = cytoscape({
      container: containerRef.current,
      elements: [
        ...data.nodes.map(n => ({
          group: 'nodes' as const,
          data: { id: n.id, label: n.label, type: n.type, ...n.properties },
        })),
        ...data.edges.map(e => ({
          group: 'edges' as const,
          data: { id: e.id, source: e.source, target: e.target, label: e.type },
        })),
      ],
      style: [
        {
          selector: 'node',
          style: {
            'background-color': (el) => NODE_COLORS[el.data('type')] ?? NODE_COLORS.Default,
            'label': 'data(label)',
            'text-valign': 'bottom',
            'font-size': '10px',
          }
        },
        {
          selector: 'edge',
          style: {
            'label': 'data(label)',
            'font-size': '8px',
            'curve-style': 'bezier',
            'target-arrow-shape': 'triangle',
          }
        }
      ],
      layout: { name: 'fcose' },
    });
    
    // Click handler → show entity detail panel
    cyRef.current.on('tap', 'node', (evt) => {
      const node = evt.target;
      onNodeClick(node.id());
    });
    
  }, [data]);
  
  return <div ref={containerRef} className="w-full h-full" />;
}
```

### 3.4 Timeline Animation

```typescript
// Time slider for graph evolution
function TimelineAnimator({ timeRange }: { timeRange: TimeRange }) {
  const [currentTime, setCurrentTime] = useState(timeRange.from);
  const [playing, setPlaying] = useState(false);
  
  // Fetch timeline snapshots from backend
  const { data: timeline } = useQuery({
    queryKey: ['graph', 'timeline', timeRange],
    queryFn: () => graphApi.getTimeline(timeRange),
  });
  
  // Play: advance currentTime by interval, update graph
  return (
    <div>
      <input type="range" 
        min={timeRange.from} max={timeRange.to}
        value={currentTime}
        onChange={e => setCurrentTime(e.target.value)} />
      <PlayButton onClick={() => setPlaying(!playing)} />
      <SpeedSelector speeds={[1, 2, 5]} />
    </div>
  );
}
```

### 3.5 Ontology Editor

```
OntologyEditorPanel
├── EntityTypesSection
│   ├── TypesList            ← Person, Org, Concept, Event, ...
│   ├── AddTypeInput
│   └── RemoveButton
├── RelationshipTypesSection
│   ├── RelationsList        ← works_at, knows, part_of, ...
│   ├── AddRelationInput
│   └── RemoveButton
└── SaveButton               ← PUT /v1/console/graph/ontology
    ↓ warning: "Changes affect future LLM extractions only"
```

### 3.6 Cypher Query Console

```
CypherConsole
├── QueryInput              ← Monaco editor with Cypher syntax
├── RunButton
├── QueryHistory            ← dropdown: last 20 queries
└── ResultSection
    ├── TableView           ← column/row table
    └── InlineGraphView     ← mini graph for node results
    
// Security: MATCH/RETURN/WITH/WHERE/ORDER BY/LIMIT only
// CREATE/DELETE/MERGE → rejected with "Read-only mode" toast
```

---

## 4. Performance Considerations

```typescript
// Result cap: max 200 nodes, 500 edges
// If truncated: show warning banner
// "Graph truncated to 200 nodes. Refine your search to see more."

// Code splitting: Cytoscape.js lazy loaded
const GraphCanvas = React.lazy(() => import('./GraphCanvas'));
// Use with: <Suspense fallback={<GraphSkeleton />}>
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] Subgraph renders within `< 2s` for depth=3, ≤ 200 nodes
- [ ] Nodes colored by entity type (7 colors defined)
- [ ] Edge labels show relationship type
- [ ] Node click → entity detail panel slides in
- [ ] Entity detail: properties table + temporal edges with valid_from/to
- [ ] Timeline slider: scrub through graph evolution
- [ ] Ontology editor: add/remove entity and relationship types
- [ ] Cypher console: read-only (CREATE/DELETE rejected → toast)
- [ ] Truncation warning when > 200 nodes
- [ ] MiniMap shown for graphs > 50 nodes
- [ ] Tenant isolation: backend injects `tenant_id` filter (UI trusts backend)
